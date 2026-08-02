package database

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/EvieePy/Echo/models"
)

const (
	idChars                   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	softDeleteRetentionMonths = 6
)

func softDeleteCutoff(now time.Time) time.Time {
	return now.AddDate(0, -softDeleteRetentionMonths, 0)
}

type Database interface {
	WriteAPIRequestLog(log models.APIRequestLog) error
	CreatePaste(p models.CreatePaste) (models.CreatePasteResponse, error)
	FetchPaste(id string, options models.FetchPasteOptions) (models.PasteResponse, error)
	FetchFile(pasteID, fileID string, options models.FetchPasteOptions) (models.File, error)
	FetchSecurity(token string) (models.Security, error)
	DeleteFile(pasteID, fileID, token string) error
	DeletePaste(pasteID, token string) error
	PurgeDeleted() error
	Ping() error
}

func generateID(length int) (string, error) {
	result := make([]byte, length)

	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(idChars))))
		if err != nil {
			return "", err
		}

		result[i] = idChars[n.Int64()]
	}

	return string(result), nil
}
