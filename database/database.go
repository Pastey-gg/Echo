package database

import (
	"crypto/rand"
	"math/big"

	"github.com/EvieePy/Echo/models"
)

const idChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Database interface {
	CreatePaste(p models.CreatePaste) (models.CreatePasteResponse, error)
	FetchPaste(id string, password *string) (models.PasteResponse, error)
	FetchSecurity(token string) (models.Security, error)
	DeletePaste(token string) error
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
