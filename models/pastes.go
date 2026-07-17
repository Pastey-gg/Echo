package models

import (
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
)

// CreatePaste is the request body for POST /pastes.
type CreatePaste struct {
	ExpiresAt      *time.Time   `json:"expires_at"`
	RemainingViews *int         `json:"remaining_views"`
	Password       *string      `json:"password"`
	Files          []CreateFile `json:"files"`
	Web            bool         `json:"-" swaggerignore:"true"`
}

// Paste is the internal DB representation. Password is never serialised.
type Paste struct {
	Id             string     `json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	Web            bool       `json:"web"`
	Views          int        `json:"views"`
	ExpiresAt      *time.Time `json:"expires_at"`
	DeletedAt      *time.Time `json:"-"`
	RemainingViews *int       `json:"remaining_views"`
	Password       *string    `json:"-"`
	Files          []File     `json:"files"`
}

type FetchPasteOptions struct {
	PasswordHeader    *string
	SafetyTokenHeader *string
	SkipView          bool
}

// PasteResponse is returned by GET /pastes/:id.
type PasteResponse struct {
	Id             string     `json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	Web            bool       `json:"web"`
	Views          int        `json:"views"`
	ExpiresAt      *time.Time `json:"expires_at"`
	RemainingViews *int       `json:"remaining_views"`
	HasPassword    bool       `json:"has_password"`
	Files          []File     `json:"files"`
}

// CreatePasteResponse is returned by POST /pastes and includes the one-time safety token.
type CreatePasteResponse struct {
	Id             string     `json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	Web            bool       `json:"web" example:"false"`
	Views          int        `json:"views"`
	ExpiresAt      *time.Time `json:"expires_at"`
	RemainingViews *int       `json:"remaining_views"`
	HasPassword    bool       `json:"has_password"`
	Files          []File     `json:"files"`
	SafetyToken    string     `json:"safety_token"`
}

// Security is returned by GET /security/:token.
type Security struct {
	PasteID     string `json:"paste_id"`
	SafetyToken string `json:"safety_token"`
	DeleteURL   string `json:"delete_url"`
}
