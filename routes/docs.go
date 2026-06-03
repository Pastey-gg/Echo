package routes

import (
	"encoding/json"
	"net/http"
	"os"

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
	fileBytes, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read swagger.json"})
	}

	var data map[string]interface{}
	if err := json.Unmarshal(fileBytes, &data); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse swagger JSON"})
	}

	// Compat for Swagger 2 to 3
	if definitions, exists := data["definitions"]; exists {
		components := map[string]any{
			"schemas": definitions,
		}
		data["components"] = components
		delete(data, "definitions")
	}

	data["openapi"] = "3.0.0"
	delete(data, "swagger")

	patchedBytes, err := json.Marshal(data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to serialize updated schema"})
	}

	return c.Blob(http.StatusOK, echo.MIMEApplicationJSON, patchedBytes)
}

func (v *DocsView) schemaYAML(c *echo.Context) error {
	return c.File("docs/swagger.yaml")
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
