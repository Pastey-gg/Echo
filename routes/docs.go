package routes

import (
	"net/http"

	"github.com/EvieePy/Echo/state"
	scalargo "github.com/bdpiprava/scalar-go"
	"github.com/labstack/echo/v5"
)

type DocsView struct {
	ctx *state.Context
}

func (v *DocsView) LoadRoutes() {
	v.ctx.Server.GET("/docs/schema.json", v.schemaJSON)
	v.ctx.Server.GET("/docs/schema.yaml", v.schemaYAML)
	v.ctx.Server.GET("/docs", v.docs)
}

func (v *DocsView) schemaJSON(c *echo.Context) error {
	return c.File("docs/swagger.json")
}

func (v *DocsView) schemaYAML(c *echo.Context) error {
	return c.File("docs/swagger.yaml")
}

func (v *DocsView) docs(c *echo.Context) error {
	html, err := scalargo.NewV2(
		scalargo.WithSpecDir("docs"),
		scalargo.WithBaseFileName("swagger.yaml"),
	)

	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Docs generation failed."})
	}

	return c.HTML(http.StatusOK, html)
}
