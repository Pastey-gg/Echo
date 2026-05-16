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
	v.ctx.Server.DELETE("/security/:token", v.deletePaste)
}

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
	sec.DeleteURL = fmt.Sprintf("%s://%s/security/%s", scheme, c.Request().Host, sec.SafetyToken)

	return c.JSON(http.StatusOK, sec)
}

func (v *SecurityView) deletePaste(c *echo.Context) error {
	token := c.Param("token")

	if err := v.ctx.Database.DeletePaste(token); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Token not found or paste has already been deleted."})
		}
		v.ctx.Logger.Errorf("Failed to delete paste for token %s: %v", token, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
	}

	return c.NoContent(http.StatusNoContent)
}
