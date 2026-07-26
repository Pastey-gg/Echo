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
	v.ctx.Server.GET("/pastes/:id/raw", v.getPasteRaw)
	v.ctx.Server.GET("/pastes/:id/files/:file_id", v.getFile)
	v.ctx.Server.DELETE("/pastes/:id", v.deletePaste)
	v.ctx.Server.DELETE("/pastes/:id/files/:file_id", v.deleteFile)
}

// createPaste godoc
// @Summary Create a new paste
// @Description Creates a new paste. You can send a JSON payload matching the CreatePaste model, or send raw text in the body (which defaults to a single file).
// @Tags pastes
// @Accept application/json
// @Accept plain
// @Produce application/json
// @Param paste body models.CreatePaste false "Paste Creation Payload (when sending JSON)"
// @Success 201 {object} models.CreatePasteResponse
// @Failure 400 {object} models.ErrorResponse "{"error": "Invalid JSON or validation failed."}"
// @Failure 500 {object} models.ErrorResponse "{"error": "Internal server error."}"
// @Router /pastes [post]
func (v *PasteView) createPaste(c *echo.Context) error {
	contentType := c.Request().Header.Get("Content-Type")
	viaWeb := c.QueryParam("web") == "true"

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

	data.Web = viaWeb
	paste, err := v.ctx.Database.CreatePaste(data)
	if err != nil {
		v.ctx.Logger.Errorf("Failed to create paste: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	if v.ctx.RMQ != nil {
		err = v.ctx.RMQ.PublishPaste(paste.Id)
		if err != nil {
			v.ctx.Logger.Errorf("Failed to send paste to Security-Scanner RMQ: %v", err)
		}
	}

	return c.JSON(http.StatusCreated, paste)
}

func fetchPasteOptions(c *echo.Context) models.FetchPasteOptions {
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

	return models.FetchPasteOptions{
		PasswordHeader:    pw,
		SafetyTokenHeader: token,
		SkipView:          c.QueryParam("skip_view") == "true",
	}
}

// getPaste godoc
// @Summary Fetch paste by ID
// @Description Retrieves a paste's metadata and files by its ID. Requires an Authorization header if the paste is password-protected.
// @Tags pastes
// @Produce application/json
// @Param id path string true "Paste ID"
// @Param Authorization header string false "Password for protected pastes"
// @Param X-Safety-Token header string false "Safety token to manage the paste or bypass view limits"
// @Param skip_view query bool false "Set to true to skip incrementing the view count"
// @Success 200 {object} models.PasteResponse
// @Failure 401 {object} models.ErrorResponse "{"error": "Invalid or missing password or safety token."}"
// @Failure 404 {object} models.ErrorResponse "{"error": "Paste not found or has expired."}"
// @Failure 500 {object} models.ErrorResponse "{"error": "Internal server error."}"
// @Router /pastes/{id} [get]
func (v *PasteView) getPaste(c *echo.Context) error {
	id := c.Param("id")
	options := fetchPasteOptions(c)

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

func rawPasteFilename(paste models.PasteResponse) string {
	if len(paste.Files) == 1 && paste.Files[0].Name != nil && *paste.Files[0].Name != "" {
		return strings.NewReplacer("\\", "_", "\"", "_", "\r", "_", "\n", "_").Replace(*paste.Files[0].Name)
	}

	return paste.Id + ".txt"
}

func rawPasteContent(paste models.PasteResponse) string {
	if len(paste.Files) == 1 {
		return paste.Files[0].Content
	}

	parts := make([]string, 0, len(paste.Files))
	for _, file := range paste.Files {
		parts = append(parts, fmt.Sprintf("<File:%s>\n%s", file.Id, file.Content))
	}

	return strings.Join(parts, "\n\n")
}

// getPasteRaw godoc
// @Summary Fetch raw paste content
// @Description Retrieves the raw text content of a paste by its ID. If the paste has multiple files, they are concatenated. Requires an Authorization header if password-protected.
// @Tags pastes
// @Produce plain
// @Param id path string true "Paste ID"
// @Param Authorization header string false "Password for protected pastes"
// @Param X-Safety-Token header string false "Safety token to manage the paste or bypass view limits"
// @Param skip_view query bool false "Set to true to skip incrementing the view count"
// @Success 200 {string} string "Raw paste content"
// @Failure 401 {object} models.ErrorResponse "{"error": "Invalid or missing password or safety token."}"
// @Failure 404 {object} models.ErrorResponse "{"error": "Paste not found or has expired."}"
// @Failure 500 {object} models.ErrorResponse "{"error": "Internal server error."}"
// @Router /pastes/{id}/raw [get]
func (v *PasteView) getPasteRaw(c *echo.Context) error {
	id := c.Param("id")
	options := fetchPasteOptions(c)

	paste, err := v.ctx.Database.FetchPaste(id, options)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Paste not found or has expired."})
		case errors.Is(err, models.ErrUnauthorized):
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or missing password or safety token."})
		}

		v.ctx.Logger.Errorf("Failed to fetch raw paste %s: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", rawPasteFilename(paste)))
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	return c.String(http.StatusOK, rawPasteContent(paste))
}

// getFile godoc
// @Summary Fetch specific files from a paste
// @Description Retrieves a specific file's metadata and content by its ID and its parent paste ID. Requires an Authorization header if the paste is password-protected.
// @Tags pastes
// @Produce application/json
// @Param id path string true "Paste ID"
// @Param file_id path string true "File ID"
// @Param Authorization header string false "Password for protected pastes"
// @Param X-Safety-Token header string false "Safety token to manage the paste or bypass view limits"
// @Param skip_view query bool false "Set to true to skip incrementing the view count"
// @Success 200 {object} models.File
// @Failure 401 {object} models.ErrorResponse "{"error": "Invalid or missing password or safety token."}"
// @Failure 404 {object} models.ErrorResponse "{"error": "Paste or file not found, or paste has expired."}"
// @Failure 500 {object} models.ErrorResponse "{"error": "Internal server error."}"
// @Router /pastes/{id}/files/{file_id} [get]
func (v *PasteView) getFile(c *echo.Context) error {
	pasteID := c.Param("id")
	fileID := c.Param("file_id")
	options := fetchPasteOptions(c)

	file, err := v.ctx.Database.FetchFile(pasteID, fileID, options)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Paste or file not found, or paste has expired."})
		case errors.Is(err, models.ErrUnauthorized):
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or missing password or safety token."})
		}

		v.ctx.Logger.Errorf("Failed to fetch file %s for paste %s: %v", fileID, pasteID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	return c.JSON(http.StatusOK, file)
}

func safetyTokenHeader(c *echo.Context) (string, bool) {
	token := c.Request().Header.Get("X-Safety-Token")
	return token, token != ""
}

// deletePaste godoc
// @Summary Delete a paste
// @Description Deletes a paste by its ID. Requires the one-time safety token provided during the paste's creation.
// @Tags pastes
// @Produce json
// @Param id path string true "Paste ID"
// @Param X-Safety-Token header string true "Safety token required for deletion"
// @Success 204 "No Content"
// @Failure 401 {object} models.ErrorResponse "{"error": "Missing safety token."}"
// @Failure 404 {object} models.ErrorResponse "{"error": "Paste not found or safety token is invalid."}"
// @Failure 500 {object} models.ErrorResponse "{"error": "Internal server error."}"
// @Router /pastes/{id} [delete]
func (v *PasteView) deletePaste(c *echo.Context) error {
	pasteID := c.Param("id")
	token, ok := safetyTokenHeader(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing safety token."})
	}

	if err := v.ctx.Database.DeletePaste(pasteID, token); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Paste not found or safety token is invalid."})
		}

		v.ctx.Logger.Errorf("Failed to delete paste %s: %v", pasteID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	return c.NoContent(http.StatusNoContent)
}

// deleteFile godoc
// @Summary Delete a file from a paste
// @Description Deletes a specific file from a paste by its ID. Requires the safety token.
// @Tags pastes
// @Produce json
// @Param id path string true "Paste ID"
// @Param file_id path string true "File ID"
// @Param X-Safety-Token header string true "Safety token required for deletion"
// @Success 204 "No Content"
// @Failure 401 {object} models.ErrorResponse "{"error": "Missing safety token."}"
// @Failure 404 {object} models.ErrorResponse "{"error": "Paste, file, or safety token not found."}"
// @Failure 409 {object} models.ErrorResponse "{"error": "Cannot delete the last file in a paste; delete the paste instead."}"
// @Failure 500 {object} models.ErrorResponse "{"error": "Internal server error."}"
// @Router /pastes/{id}/files/{file_id} [delete]
func (v *PasteView) deleteFile(c *echo.Context) error {
	pasteID := c.Param("id")
	fileID := c.Param("file_id")
	token, ok := safetyTokenHeader(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing safety token."})
	}

	if err := v.ctx.Database.DeleteFile(pasteID, fileID, token); err != nil {
		switch {
		case errors.Is(err, models.ErrNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Paste, file, or safety token not found."})
		case errors.Is(err, models.ErrConflict):
			return c.JSON(http.StatusConflict, map[string]string{"error": "Cannot delete the last file in a paste; delete the paste instead."})
		}

		v.ctx.Logger.Errorf("Failed to delete file %s for paste %s: %v", fileID, pasteID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	return c.NoContent(http.StatusNoContent)
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

	if p.RemainingViews != nil && (*p.RemainingViews < 1 || *p.RemainingViews > 1000) {
		return errors.New("remaining_views must be between 1 and 1000.")
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
