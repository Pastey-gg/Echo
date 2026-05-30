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

func (m *Memory) Ping() error { return nil }

func (m *Memory) CreatePaste(p models.CreatePaste) (models.CreatePasteResponse, error) {
	id, err := generateID(m.config.Pastes.IdLen)
	if err != nil {
		return models.CreatePasteResponse{}, err
	}

	token, err := generateID(m.config.Pastes.TokenLen)
	if err != nil {
		return models.CreatePasteResponse{}, err
	}

	var hashedPw []byte
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

	paste := models.Paste{
		Id:             id,
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      p.ExpiresAt,
		RemainingViews: p.RemainingViews,
		Files:          files,
	}

	m.mu.Lock()
	m.pastes[id] = &storedPaste{paste: paste, hashedPassword: hashedPw, safetyToken: token}
	m.tokens[token] = id
	m.mu.Unlock()

	return models.CreatePasteResponse{
		SafetyToken: token,
		PasteResponse: models.PasteResponse{
			Id:             paste.Id,
			CreatedAt:      paste.CreatedAt,
			ExpiresAt:      paste.ExpiresAt,
			RemainingViews: paste.RemainingViews,
			HasPassword:    hashedPw != nil,
			Files:          files,
		},
	}, nil
}

func (m *Memory) authorizeAndCountPaste(id string, sp *storedPaste, options models.FetchPasteOptions) error {
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

	return models.PasteResponse{
		Id:             sp.paste.Id,
		CreatedAt:      sp.paste.CreatedAt,
		Views:          sp.paste.Views,
		ExpiresAt:      sp.paste.ExpiresAt,
		RemainingViews: sp.paste.RemainingViews,
		HasPassword:    sp.hashedPassword != nil,
		Files:          sp.paste.Files,
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
			return f, nil
		}
	}

	return models.File{}, models.ErrNotFound
}

func (m *Memory) FetchSecurity(token string) (models.Security, error) {
	m.mu.RLock()
	id, ok := m.tokens[token]
	m.mu.RUnlock()

	if !ok {
		return models.Security{}, models.ErrNotFound
	}

	return models.Security{PasteID: id, SafetyToken: token}, nil
}

func (m *Memory) DeleteFile(token, fileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, ok := m.tokens[token]
	if !ok {
		return models.ErrNotFound
	}

	sp, ok := m.pastes[id]
	if !ok {
		return models.ErrNotFound
	}

	idx := -1
	for i, f := range sp.paste.Files {
		if f.Id == fileID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return models.ErrNotFound
	}

	if len(sp.paste.Files) <= 1 {
		return models.ErrConflict
	}

	sp.paste.Files = append(sp.paste.Files[:idx], sp.paste.Files[idx+1:]...)
	return nil
}

func (m *Memory) DeletePaste(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, ok := m.tokens[token]
	if !ok {
		return models.ErrNotFound
	}

	delete(m.pastes, id)
	delete(m.tokens, token)
	return nil
}
