package routes

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/EvieePy/Echo/models"
	"github.com/EvieePy/Echo/state"
	"github.com/labstack/echo/v5"
)

type PasteView struct {
	ctx *state.Context
}

func (v *PasteView) LoadRoutes() {
	v.ctx.Server.POST("/pastes", v.createPaste)
	v.ctx.Server.GET("/pastes/:id", v.getPaste)
}

func (v *PasteView) createPaste(c *echo.Context) error {
	contentType := c.Request().Header.Get("Content-Type")

	var data models.CreatePaste

	if strings.HasPrefix(contentType, "application/json") {
		if err := c.Bind(&data); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON."})
		}

	} else {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to read body."})
		}

		content := strings.TrimSpace(string(body))
		data = models.CreatePaste{
			Files: []models.CreateFile{{Content: content}},
		}
	}

	if err := validatePaste(v.ctx, data); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	paste, err := v.ctx.Database.CreatePaste(data)
	if err != nil {
		v.ctx.Logger.Errorf("Failed to create paste: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	return c.JSON(http.StatusCreated, paste)
}

func (v *PasteView) getPaste(c *echo.Context) error {
	id := c.Param("id")

	password := c.Request().Header.Get("Authorization")
	var pw *string
	if password != "" {
		pw = &password
	}

	safetyToken := c.Request().Header.Get("X-Safety-Token")
	var token *string
	if safetyToken != "" {
		token = &safetyToken
	}

	options := models.FetchPasteOptions{
		PasswordHeader:    pw,
		SafetyTokenHeader: token,
		SkipView:          c.QueryParam("skip_view") == "true",
	}

	paste, err := v.ctx.Database.FetchPaste(id, options)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Paste not found or has expired."})
		case errors.Is(err, models.ErrUnauthorized):
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or missing password or safety token."})
		}

		v.ctx.Logger.Errorf("Failed to fetch paste %s: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	return c.JSON(http.StatusOK, paste)
}

func validatePaste(ctx *state.Context, p models.CreatePaste) error {
	maxFiles := ctx.Config.Pastes.MaxFiles
	maxFileSize := ctx.Config.Pastes.MaxFileSize

	if len(p.Files) == 0 {
		return errors.New("At least one valid file must be provided.")
	}

	if len(p.Files) > maxFiles {
		return fmt.Errorf("Maximum of %d files per paste allowed.", maxFiles)
	}

	for _, f := range p.Files {
		if f.Content == "" {
			return errors.New("File content cannot be empty.")
		}

		if len([]rune(f.Content)) > maxFileSize {
			return fmt.Errorf("File content exceeds maximum of %d characters.", maxFileSize)
		}
	}

	return nil
}
