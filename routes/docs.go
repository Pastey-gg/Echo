package routes

import (
	"net/http"

	"github.com/EvieePy/Echo/state"
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
	if err := c.File("docs/swagger.json"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to serve swagger.json"})
	}
	return nil
}

func (v *DocsView) schemaYAML(c *echo.Context) error {
	if err := c.File("docs/swagger.yaml"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to serve swagger.yaml"})
	}
	return nil
}

func (v *DocsView) docs(c *echo.Context) error {
	htmlContent := `
	<!doctype html>
	<html>
	  <head>
	    <title>Pastey.gg Documentation</title>
	    <meta charset="utf-8" />
	    <meta name="viewport" content="width=device-width, initial-scale=1" />
	    <style>
	      body { margin: 0; }
	    </style>
	  </head>
	  <body>
	    <script
	      id="api-reference"
	      data-url="/docs/schema.json">
	    </script>
	    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
	  </body>
	</html>
	`
	return c.HTML(http.StatusOK, htmlContent)
}
