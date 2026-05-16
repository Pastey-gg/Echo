package routes

import (
	"net/http"

	"github.com/EvieePy/Echo/state"
	"github.com/labstack/echo/v5"
)

const version = "1.0.0"

type MetaView struct {
	ctx *state.Context
}

func (v *MetaView) LoadRoutes() {
	v.ctx.Server.GET("/health", v.health)
	v.ctx.Server.GET("/version", v.version)
}

func (v *MetaView) health(c *echo.Context) error {
	if err := v.ctx.Database.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Database unavailable."})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (v *MetaView) version(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"version": version})
}
