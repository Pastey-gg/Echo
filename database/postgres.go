package database

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EvieePy/Echo/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed init.sql
var initSQL string

//go:embed schema.sql
var schema string

// pgErrInvalidCatalogName is the SQLSTATE returned when the target database does not exist.
const pgErrInvalidCatalogName = "3D000"

type Postgres struct {
	pool   *pgxpool.Pool
	config *models.Config
}

func NewPostgres(config *models.Config) (*Postgres, error) {
	ctx := context.Background()

	pool, err := newPoolWithBootstrap(ctx, config.Database.DSN)
	if err != nil {
		return nil, err
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, err
	}

	pg := &Postgres{pool: pool, config: config}
	if err := pg.purgeDeleted(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pg, nil
}

// newPoolWithBootstrap connects to the target database, creating it first if it
// does not yet exist, then applies init.sql when the connected role is a superuser.
func newPoolWithBootstrap(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != pgErrInvalidCatalogName {
			pool.Close()
			return nil, err
		}

		// Database does not exist — create it then reconnect.
		pool.Close()
		if err := createDatabase(ctx, conf); err != nil {
			return nil, err
		}

		pool, err = pgxpool.NewWithConfig(ctx, conf)
		if err != nil {
			return nil, err
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, err
		}
	}

	if err := applyInitSQL(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// createDatabase connects to the postgres maintenance database and creates the
// target database specified in conf. Returns an error with a message if
// the connecting role is not a superuser (e.g. in Docker where POSTGRES_DB
// should have already created the database via the entrypoint).
func createDatabase(ctx context.Context, conf *pgxpool.Config) error {
	targetDB := conf.ConnConfig.Database
	if targetDB == "" {
		return errors.New("database name missing in DSN")
	}

	adminConf := conf.Copy()
	adminConf.ConnConfig.Database = "postgres"

	adminPool, err := pgxpool.NewWithConfig(ctx, adminConf)
	if err != nil {
		// Cannot reach maintenance db — likely a non-superuser or missing role.
		return fmt.Errorf("database %q does not exist and cannot be created automatically (could not connect to maintenance database): %w", targetDB, err)
	}
	defer adminPool.Close()

	if err := adminPool.Ping(ctx); err != nil {
		return fmt.Errorf("database %q does not exist and cannot be created automatically (could not connect to maintenance database): %w", targetDB, err)
	}

	var isSuperuser bool
	if err := adminPool.QueryRow(ctx,
		`SELECT COALESCE((SELECT rolsuper FROM pg_roles WHERE rolname = current_user), false)`,
	).Scan(&isSuperuser); err != nil {
		return err
	}

	if !isSuperuser {
		return fmt.Errorf("database %q does not exist; current user is not a superuser and cannot create it — in Docker ensure POSTGRES_DB is set, or create the database manually", targetDB)
	}

	var exists bool
	if err := adminPool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, targetDB,
	).Scan(&exists); err != nil {
		return err
	}

	if exists {
		return nil
	}

	_, err = adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{targetDB}.Sanitize()))
	return err
}

// applyInitSQL runs init.sql only when the connected role is a superuser, so
// that the echo role and grants are set up automatically on local installs.
// In Docker the Postgres entrypoint already handles this via the mounted init script.
func applyInitSQL(ctx context.Context, pool *pgxpool.Pool) error {
	var isSuperuser bool
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT rolsuper FROM pg_roles WHERE rolname = current_user), false)`,
	).Scan(&isSuperuser); err != nil {
		return err
	}

	if !isSuperuser {
		return nil
	}

	_, err := pool.Exec(ctx, initSQL)
	return err
}

func (p *Postgres) Ping() error {
	return p.pool.Ping(context.Background())
}

func (p *Postgres) PurgeDeleted() error {
	return p.purgeDeleted(context.Background())
}

func (p *Postgres) purgeDeleted(ctx context.Context) error {
	_, err := p.pool.Exec(ctx,
		`DELETE FROM files WHERE deleted_at < now() - interval '24 hours';
		 DELETE FROM pastes WHERE deleted_at < now() - interval '24 hours'`,
	)
	return err
}

func (p *Postgres) CreatePaste(cp models.CreatePaste) (models.CreatePasteResponse, error) {
	ctx := context.Background()

	var hashedPw []byte
	if cp.Password != nil && *cp.Password != "" {
		var err error
		hashedPw, err = bcrypt.GenerateFromPassword([]byte(*cp.Password), bcrypt.DefaultCost)

		if err != nil {
			return models.CreatePasteResponse{}, err
		}
	}

	files := make([]models.File, len(cp.Files))
	for i, f := range cp.Files {

		fileID, err := generateID(p.config.Pastes.IdLen)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}

		files[i] = models.File{
			Id:             fileID,
			CharacterCount: len([]rune(f.Content)),
			LineCount:      strings.Count(f.Content, "\n") + 1,
			CreateFile:     f,
		}
	}

	var id, token string
	var paste models.Paste

	for {
		var err error
		id, err = generateID(p.config.Pastes.IdLen)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}

		token, err = generateID(p.config.Pastes.TokenLen)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}

		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}

		var hashedPwStr *string
		if hashedPw != nil {
			s := string(hashedPw)
			hashedPwStr = &s
		}

		err = tx.QueryRow(ctx,
			`INSERT INTO pastes (id, web, expires_at, remaining_views, hashed_password, safety_token)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id, web, created_at, views, expires_at, remaining_views`,
			id, cp.Web, cp.ExpiresAt, cp.RemainingViews, hashedPwStr, token,
		).Scan(&paste.Id, &paste.Web, &paste.CreatedAt, &paste.Views, &paste.ExpiresAt, &paste.RemainingViews)

		if err != nil {
			tx.Rollback(ctx)
			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue // duplicate id or token, retry
			}

			return models.CreatePasteResponse{}, err
		}

		for _, f := range files {
			_, err = tx.Exec(ctx,
				`INSERT INTO files (id, paste_id, name, language, content, character_count, line_count)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				f.Id, id, f.Name, f.Language, f.Content, f.CharacterCount, f.LineCount,
			)
			if err != nil {
				tx.Rollback(ctx)
				return models.CreatePasteResponse{}, err
			}
		}

		if err = tx.Commit(ctx); err != nil {
			return models.CreatePasteResponse{}, err
		}
		break
	}

	return createPasteResponse(paste, files, hashedPw != nil, token), nil
}

func (p *Postgres) authorizeAndCountPaste(ctx context.Context, id string, paste models.Paste, hashedPw *string, safetyToken string, options models.FetchPasteOptions) (int, *int, error) {
	if paste.ExpiresAt != nil && time.Now().After(*paste.ExpiresAt) {
		p.pool.Exec(ctx, `UPDATE pastes SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
		return 0, nil, models.ErrNotFound
	}

	skipView := false
	if options.SkipView {
		if options.SafetyTokenHeader != nil && *options.SafetyTokenHeader == safetyToken {
			skipView = true
		} else if hashedPw != nil && options.PasswordHeader != nil && *options.PasswordHeader != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(*hashedPw), []byte(*options.PasswordHeader)); err == nil {
				skipView = true
			}
		}

		if !skipView {
			return 0, nil, models.ErrUnauthorized
		}
	}

	if !skipView && paste.RemainingViews != nil && *paste.RemainingViews <= 0 {
		p.pool.Exec(ctx, `UPDATE pastes SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
		return 0, nil, models.ErrNotFound
	}

	if !skipView && hashedPw != nil {
		if options.PasswordHeader == nil || *options.PasswordHeader == "" {
			return 0, nil, models.ErrUnauthorized
		}

		if err := bcrypt.CompareHashAndPassword([]byte(*hashedPw), []byte(*options.PasswordHeader)); err != nil {
			return 0, nil, models.ErrUnauthorized
		}
	}

	newViews := paste.Views
	newRemaining := paste.RemainingViews

	if !skipView {
		err := p.pool.QueryRow(ctx,
			`UPDATE pastes
		 SET views = views + 1,
		     remaining_views = CASE WHEN remaining_views IS NOT NULL THEN remaining_views - 1 ELSE NULL END
		 WHERE id = $1
		 RETURNING views, remaining_views`, id,
		).Scan(&newViews, &newRemaining)

		if err != nil {
			return 0, nil, err
		}
	}

	return newViews, newRemaining, nil
}

func (p *Postgres) fetchPasteForRead(ctx context.Context, id string) (models.Paste, *string, string, error) {
	var hashedPw *string
	var safetyToken string
	var paste models.Paste

	err := p.pool.QueryRow(ctx,
		`SELECT id, web, created_at, views, expires_at, remaining_views, hashed_password, safety_token
		 FROM pastes WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&paste.Id, &paste.Web, &paste.CreatedAt, &paste.Views, &paste.ExpiresAt, &paste.RemainingViews, &hashedPw, &safetyToken)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.Paste{}, nil, "", models.ErrNotFound
	}

	if err != nil {
		return models.Paste{}, nil, "", err
	}

	return paste, hashedPw, safetyToken, nil
}

func (p *Postgres) FetchPaste(id string, options models.FetchPasteOptions) (models.PasteResponse, error) {
	ctx := context.Background()

	paste, hashedPw, safetyToken, err := p.fetchPasteForRead(ctx, id)
	if err != nil {
		return models.PasteResponse{}, err
	}

	newViews, newRemaining, err := p.authorizeAndCountPaste(ctx, id, paste, hashedPw, safetyToken, options)
	if err != nil {
		return models.PasteResponse{}, err
	}

	rows, err := p.pool.Query(ctx,
		`SELECT id, name, language, content, character_count, line_count
		 FROM files WHERE paste_id = $1 AND deleted_at IS NULL`, id,
	)

	if err != nil {
		return models.PasteResponse{}, err
	}

	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var f models.File
		if err := rows.Scan(&f.Id, &f.Name, &f.Language, &f.Content, &f.CharacterCount, &f.LineCount); err != nil {
			return models.PasteResponse{}, err
		}

		files = append(files, f)
	}

	return models.PasteResponse{
		Id:             paste.Id,
		CreatedAt:      paste.CreatedAt,
		Views:          newViews,
		ExpiresAt:      paste.ExpiresAt,
		RemainingViews: newRemaining,
		HasPassword:    hashedPw != nil,
		Files:          files,
		Web:            paste.Web,
	}, nil
}

func (p *Postgres) FetchFile(pasteID, fileID string, options models.FetchPasteOptions) (models.File, error) {
	ctx := context.Background()

	paste, hashedPw, safetyToken, err := p.fetchPasteForRead(ctx, pasteID)
	if err != nil {
		return models.File{}, err
	}

	if _, _, err := p.authorizeAndCountPaste(ctx, pasteID, paste, hashedPw, safetyToken, options); err != nil {
		return models.File{}, err
	}

	var file models.File
	err = p.pool.QueryRow(ctx,
		`SELECT id, name, language, content, character_count, line_count
		 FROM files WHERE paste_id = $1 AND id = $2 AND deleted_at IS NULL`, pasteID, fileID,
	).Scan(&file.Id, &file.Name, &file.Language, &file.Content, &file.CharacterCount, &file.LineCount)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.File{}, models.ErrNotFound
	}

	if err != nil {
		return models.File{}, err
	}

	return file, nil
}

func (p *Postgres) FetchSecurity(token string) (models.Security, error) {
	ctx := context.Background()

	var id string

	err := p.pool.QueryRow(ctx,
		`SELECT id FROM pastes WHERE safety_token = $1 AND deleted_at IS NULL`, token,
	).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.Security{}, models.ErrNotFound
	}

	if err != nil {
		return models.Security{}, err
	}

	return models.Security{PasteID: id, SafetyToken: token}, nil
}

func (p *Postgres) DeleteFile(pasteID, fileID, token string) error {
	ctx := context.Background()

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var fetchedPasteID string
	err = tx.QueryRow(ctx,
		`SELECT id FROM pastes WHERE id = $1 AND safety_token = $2 AND deleted_at IS NULL FOR UPDATE`, pasteID, token,
	).Scan(&fetchedPasteID)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrNotFound
	}

	if err != nil {
		return err
	}

	var fileCount int
	var targetFileCount int
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE id = $2)
		 FROM files WHERE paste_id = $1 AND deleted_at IS NULL`, pasteID, fileID,
	).Scan(&fileCount, &targetFileCount)

	if err != nil {
		return err
	}

	if targetFileCount == 0 {
		return models.ErrNotFound
	}

	if fileCount <= 1 {
		return models.ErrConflict
	}

	tag, err := tx.Exec(ctx,
		`UPDATE files SET deleted_at = now() WHERE id = $1 AND paste_id = $2 AND deleted_at IS NULL`, fileID, pasteID,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return tx.Commit(ctx)
}

func (p *Postgres) DeletePaste(pasteID, token string) error {
	ctx := context.Background()

	tag, err := p.pool.Exec(ctx,
		`UPDATE pastes SET deleted_at = now() WHERE id = $1 AND safety_token = $2 AND deleted_at IS NULL`, pasteID, token,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}
