package models

import "time"

type CreatePaste struct {
	ExpiresAt      *time.Time `json:"expires_at"`
	RemainingViews *int       `json:"remaining_views"`
	Password       *string    `json:"password"`
	Files          []CreateFile
}

type Paste struct {
	Id             string     `json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	Views          int        `json:"views"`
	ExpiresAt      *time.Time `json:"expires_at"`
	RemainingViews *int       `json:"remaining_views"`
	Password       *string    `json:"password"`
	Files          []File
}
