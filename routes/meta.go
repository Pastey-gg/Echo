package routes

import (
	"net/http"

	"github.com/EvieePy/Echo/state"
	"github.com/labstack/echo/v5"
)

type MetaView struct {
	ctx *state.Context
}

func (v *MetaView) LoadRoutes() {
	v.ctx.Server.GET("/health", v.health)
	v.ctx.Server.GET("/version", v.version)
	v.ctx.Server.GET("/version/info", v.versionInfo)
}

func (v *MetaView) health(c *echo.Context) error {
	if err := v.ctx.Database.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Database unavailable."})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (v *MetaView) version(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"version": v.ctx.VersionInfo.Version})
}

func (v *MetaView) versionInfo(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"version":     v.ctx.VersionInfo.Version,
		"commit":      v.ctx.VersionInfo.CommitHash,
		"commit_time": v.ctx.VersionInfo.CommitTime,
	})
}
