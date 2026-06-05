package database

import "github.com/EvieePy/Echo/models"

func createPasteResponse(paste models.Paste, files []models.File, hasPassword bool, safetyToken string) models.CreatePasteResponse {
	return models.CreatePasteResponse{
		SafetyToken: safetyToken,
		PasteResponse: models.PasteResponse{
			Id:             paste.Id,
			CreatedAt:      paste.CreatedAt,
			Web:            paste.Web,
			Views:          paste.Views,
			ExpiresAt:      paste.ExpiresAt,
			RemainingViews: paste.RemainingViews,
			HasPassword:    hasPassword,
			Files:          files,
		},
	}
}
