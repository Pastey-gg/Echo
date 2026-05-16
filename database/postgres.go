package database

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"time"

	"github.com/EvieePy/Echo/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var schema string

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	if _, err := pool.Exec(context.Background(), schema); err != nil {
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Ping() error {
	return p.pool.Ping(context.Background())
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
		fileID, err := generateID(pasteIDLength)
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
	var createdAt time.Time

	for {
		var err error
		id, err = generateID(pasteIDLength)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}
		token, err = generateID(tokenLength)
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
			`INSERT INTO pastes (id, expires_at, remaining_views, hashed_password, safety_token)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING created_at`,
			id, cp.ExpiresAt, cp.RemainingViews, hashedPwStr, token,
		).Scan(&createdAt)
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

	return models.CreatePasteResponse{
		SafetyToken: token,
		PasteResponse: models.PasteResponse{
			Id:             id,
			CreatedAt:      createdAt,
			ExpiresAt:      cp.ExpiresAt,
			RemainingViews: cp.RemainingViews,
			HasPassword:    hashedPw != nil,
			Files:          files,
		},
	}, nil
}

func (p *Postgres) FetchPaste(id string, password *string) (models.PasteResponse, error) {
	ctx := context.Background()

	var hashedPw *string
	var paste models.Paste
	err := p.pool.QueryRow(ctx,
		`SELECT id, created_at, views, expires_at, remaining_views, hashed_password
		 FROM pastes WHERE id = $1`, id,
	).Scan(&paste.Id, &paste.CreatedAt, &paste.Views, &paste.ExpiresAt, &paste.RemainingViews, &hashedPw)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PasteResponse{}, models.ErrNotFound
	}
	if err != nil {
		return models.PasteResponse{}, err
	}

	if paste.ExpiresAt != nil && time.Now().After(*paste.ExpiresAt) {
		p.pool.Exec(ctx, `DELETE FROM pastes WHERE id = $1`, id)
		return models.PasteResponse{}, models.ErrNotFound
	}
	if paste.RemainingViews != nil && *paste.RemainingViews <= 0 {
		p.pool.Exec(ctx, `DELETE FROM pastes WHERE id = $1`, id)
		return models.PasteResponse{}, models.ErrNotFound
	}

	if hashedPw != nil {
		if password == nil || *password == "" {
			return models.PasteResponse{}, models.ErrUnauthorized
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*hashedPw), []byte(*password)); err != nil {
			return models.PasteResponse{}, models.ErrUnauthorized
		}
	}

	var newViews int
	var newRemaining *int
	err = p.pool.QueryRow(ctx,
		`UPDATE pastes
		 SET views = views + 1,
		     remaining_views = CASE WHEN remaining_views IS NOT NULL THEN remaining_views - 1 ELSE NULL END
		 WHERE id = $1
		 RETURNING views, remaining_views`, id,
	).Scan(&newViews, &newRemaining)
	if err != nil {
		return models.PasteResponse{}, err
	}

	rows, err := p.pool.Query(ctx,
		`SELECT id, name, language, content, character_count, line_count
		 FROM files WHERE paste_id = $1`, id,
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
	}, nil
}

func (p *Postgres) FetchSecurity(token string) (models.Security, error) {
	var id string
	err := p.pool.QueryRow(context.Background(),
		`SELECT id FROM pastes WHERE safety_token = $1`, token,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Security{}, models.ErrNotFound
	}
	if err != nil {
		return models.Security{}, err
	}
	return models.Security{PasteID: id, SafetyToken: token}, nil
}

func (p *Postgres) DeletePaste(token string) error {
	tag, err := p.pool.Exec(context.Background(),
		`DELETE FROM pastes WHERE safety_token = $1`, token,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}
