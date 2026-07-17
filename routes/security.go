package routes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/EvieePy/Echo/models"
	"github.com/EvieePy/Echo/state"
	"github.com/labstack/echo/v5"
)

type SecurityView struct {
	ctx *state.Context
}

func (v *SecurityView) LoadRoutes() {
	v.ctx.Server.GET("/security/:token", v.getSecurity)
}

// getSecurity godoc
// @Summary Get security information
// @Description Looks up an active paste by safety token and returns deletion metadata.
// @Tags security
// @Produce application/json
// @Param token path string true "Safety Token"
// @Success 200 {object} models.Security
// @Failure 404 {object} models.ErrorResponse "{"error": "Token not found or paste has been deleted."}"
// @Failure 500 {object} models.ErrorResponse "{"error": "Internal server error."}"
// @Router /security/{token} [get]
func (v *SecurityView) getSecurity(c *echo.Context) error {
	token := c.Param("token")

	sec, err := v.ctx.Database.FetchSecurity(token)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Token not found or paste has been deleted."})
		}

		v.ctx.Logger.Errorf("Failed to fetch security for token %s: %v", token, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	sec.DeleteURL = fmt.Sprintf("%s://%s/pastes/%s", scheme, c.Request().Host, sec.PasteID)

	return c.JSON(http.StatusOK, sec)
}
