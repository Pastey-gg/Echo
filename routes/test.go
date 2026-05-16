package routes

import (
	"net/http"

	"github.com/EvieePy/Echo/state"
	"github.com/labstack/echo/v5"
)

type TestView struct {
	ctx *state.Context
}

func (v *TestView) LoadRoutes() {
	v.test("/test")
}

func (v *TestView) test(path string) {
	route := func(c *echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	}

	v.ctx.Server.GET(path, route)
}
