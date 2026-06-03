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

// health godoc
// @Summary Health check
// @Description Health check which returns a 200 Success when healthy.
// @Tags meta
// @Produce application/json
// @Success 200 {map} string
// @Router /health [get]
func (v *MetaView) health(c *echo.Context) error {
	if err := v.ctx.Database.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Database unavailable."})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// version godoc
// @Summary Get application version
// @Description Returns the current application version.
// @Tags meta
// @Produce application/json
// @Success 200 {map} string
// @Router /version [get]
func (v *MetaView) version(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"version": v.ctx.VersionInfo.Version})
}

// versionInfo godoc
// @Summary Get detailed version information
// @Description Returns the application version, commit hash, and commit time.
// @Tags meta
// @Produce application/json
// @Success 200 {map} string
// @Router /version/info [get]
func (v *MetaView) versionInfo(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"version":     v.ctx.VersionInfo.Version,
		"commit":      v.ctx.VersionInfo.CommitHash,
		"commit_time": v.ctx.VersionInfo.CommitTime,
	})
}
