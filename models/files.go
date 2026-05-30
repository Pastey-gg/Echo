package models

import "time"

type CreateFile struct {
	Name     *string `json:"name"`
	Language *string `json:"language"`
	Content  string  `json:"content"`
}

type File struct {
	DeletedAt      *time.Time `json:"-"`
	Id             string     `json:"id"`
	CharacterCount int        `json:"character_count"`
	LineCount      int        `json:"line_count"`
	CreateFile
}
