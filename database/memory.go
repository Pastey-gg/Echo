package database

import (
	"strings"
	"sync"
	"time"

	"github.com/EvieePy/Echo/models"
	"golang.org/x/crypto/bcrypt"
)

type storedPaste struct {
	paste          models.Paste
	hashedPassword []byte
	safetyToken    string
}

type Memory struct {
	mu     sync.RWMutex
	pastes map[string]*storedPaste // id -> storedPaste
	tokens map[string]string       // safetyToken -> paste id
	config *models.Config
}

func NewMemory(config *models.Config) *Memory {
	return &Memory{
		pastes: make(map[string]*storedPaste),
		tokens: make(map[string]string),
		config: config,
	}
}

func (m *Memory) purgeDeleted(now time.Time) {
	cutoff := now.Add(-softDeleteRetention)

	for id, sp := range m.pastes {
		if sp.paste.DeletedAt != nil && sp.paste.DeletedAt.Before(cutoff) {
			delete(m.pastes, id)
			delete(m.tokens, sp.safetyToken)
			continue
		}

		activeFiles := sp.paste.Files[:0]
		for _, f := range sp.paste.Files {
			if f.DeletedAt == nil || !f.DeletedAt.Before(cutoff) {
				activeFiles = append(activeFiles, f)
			}
		}
		sp.paste.Files = activeFiles
	}
}

func (m *Memory) PurgeDeleted() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.purgeDeleted(time.Now().UTC())
	return nil
}

func (m *Memory) Ping() error { return nil }

func (m *Memory) CreatePaste(p models.CreatePaste) (models.CreatePasteResponse, error) {
	var hashedPw []byte
	var err error
	if p.Password != nil && *p.Password != "" {
		hashedPw, err = bcrypt.GenerateFromPassword([]byte(*p.Password), bcrypt.DefaultCost)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}
	}

	files := make([]models.File, len(p.Files))
	for i, f := range p.Files {
		fileID, err := generateID(m.config.Pastes.IdLen)
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

	m.mu.Lock()
	defer m.mu.Unlock()

	var id string
	for {
		id, err = generateID(m.config.Pastes.IdLen)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}

		if _, exists := m.pastes[id]; !exists {
			break
		}
	}

	var token string
	for {
		token, err = generateID(m.config.Pastes.TokenLen)
		if err != nil {
			return models.CreatePasteResponse{}, err
		}

		if _, exists := m.tokens[token]; !exists {
			break
		}
	}

	paste := models.Paste{
		Id:             id,
		CreatedAt:      time.Now().UTC(),
		Web:            p.Web,
		ExpiresAt:      p.ExpiresAt,
		RemainingViews: p.RemainingViews,
		Files:          files,
	}

	m.pastes[id] = &storedPaste{paste: paste, hashedPassword: hashedPw, safetyToken: token}
	m.tokens[token] = id

	return models.CreatePasteResponse{
		SafetyToken: token,
		PasteResponse: models.PasteResponse{
			Id:             paste.Id,
			CreatedAt:      paste.CreatedAt,
			Web:            paste.Web,
			ExpiresAt:      paste.ExpiresAt,
			RemainingViews: paste.RemainingViews,
			HasPassword:    hashedPw != nil,
			Files:          files,
		},
	}, nil
}

func (m *Memory) authorizeAndCountPaste(id string, sp *storedPaste, options models.FetchPasteOptions) error {
	if sp.paste.DeletedAt != nil {
		return models.ErrNotFound
	}

	if sp.paste.ExpiresAt != nil && time.Now().After(*sp.paste.ExpiresAt) {
		delete(m.pastes, id)
		delete(m.tokens, sp.safetyToken)
		return models.ErrNotFound
	}

	skipView := false
	if options.SkipView {
		if options.SafetyTokenHeader != nil && *options.SafetyTokenHeader == sp.safetyToken {
			skipView = true
		} else if sp.hashedPassword != nil && options.PasswordHeader != nil && *options.PasswordHeader != "" {
			if err := bcrypt.CompareHashAndPassword(sp.hashedPassword, []byte(*options.PasswordHeader)); err == nil {
				skipView = true
			}
		}

		if !skipView {
			return models.ErrUnauthorized
		}
	}

	if !skipView && sp.paste.RemainingViews != nil && *sp.paste.RemainingViews <= 0 {
		delete(m.pastes, id)
		delete(m.tokens, sp.safetyToken)
		return models.ErrNotFound
	}

	if !skipView && sp.hashedPassword != nil {
		if options.PasswordHeader == nil || *options.PasswordHeader == "" {
			return models.ErrUnauthorized
		}

		if err := bcrypt.CompareHashAndPassword(sp.hashedPassword, []byte(*options.PasswordHeader)); err != nil {
			return models.ErrUnauthorized
		}
	}

	if !skipView {
		sp.paste.Views++
		if sp.paste.RemainingViews != nil {
			remaining := *sp.paste.RemainingViews - 1
			sp.paste.RemainingViews = &remaining
		}
	}

	return nil
}

func (m *Memory) FetchPaste(id string, options models.FetchPasteOptions) (models.PasteResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sp, ok := m.pastes[id]
	if !ok {
		return models.PasteResponse{}, models.ErrNotFound
	}

	if err := m.authorizeAndCountPaste(id, sp, options); err != nil {
		return models.PasteResponse{}, err
	}

	files := make([]models.File, 0, len(sp.paste.Files))
	for _, f := range sp.paste.Files {
		if f.DeletedAt == nil {
			files = append(files, f)
		}
	}

	return models.PasteResponse{
		Id:             sp.paste.Id,
		CreatedAt:      sp.paste.CreatedAt,
		Web:            sp.paste.Web,
		Views:          sp.paste.Views,
		ExpiresAt:      sp.paste.ExpiresAt,
		RemainingViews: sp.paste.RemainingViews,
		HasPassword:    sp.hashedPassword != nil,
		Files:          files,
	}, nil
}

func (m *Memory) FetchFile(pasteID, fileID string, options models.FetchPasteOptions) (models.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sp, ok := m.pastes[pasteID]
	if !ok {
		return models.File{}, models.ErrNotFound
	}

	if err := m.authorizeAndCountPaste(pasteID, sp, options); err != nil {
		return models.File{}, err
	}

	for _, f := range sp.paste.Files {
		if f.Id == fileID {
			if f.DeletedAt != nil {
				return models.File{}, models.ErrNotFound
			}

			return f, nil
		}
	}

	return models.File{}, models.ErrNotFound
}

func (m *Memory) FetchSecurity(token string) (models.Security, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, ok := m.tokens[token]

	if !ok {
		return models.Security{}, models.ErrNotFound
	}

	sp, ok := m.pastes[id]
	if !ok || sp.paste.DeletedAt != nil {
		return models.Security{}, models.ErrNotFound
	}

	return models.Security{PasteID: id, SafetyToken: token}, nil
}

func (m *Memory) DeleteFile(pasteID, fileID, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

	id, ok := m.tokens[token]
	if !ok || id != pasteID {
		return models.ErrNotFound
	}

	sp, ok := m.pastes[pasteID]
	if !ok || sp.paste.DeletedAt != nil {
		return models.ErrNotFound
	}

	idx := -1
	activeFileCount := 0
	for i, f := range sp.paste.Files {
		if f.DeletedAt == nil {
			activeFileCount++
		}

		if f.Id == fileID && f.DeletedAt == nil {
			idx = i
		}
	}

	if idx == -1 {
		return models.ErrNotFound
	}

	if activeFileCount <= 1 {
		return models.ErrConflict
	}

	sp.paste.Files[idx].DeletedAt = &now
	return nil
}

func (m *Memory) DeletePaste(pasteID, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

	id, ok := m.tokens[token]
	if !ok || id != pasteID {
		return models.ErrNotFound
	}

	sp, ok := m.pastes[pasteID]
	if !ok || sp.paste.DeletedAt != nil {
		return models.ErrNotFound
	}

	sp.paste.DeletedAt = &now
	return nil
}
