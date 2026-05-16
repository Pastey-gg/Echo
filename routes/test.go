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
	v.ctx.Server.GET("/test", v.test)
}

func (v *TestView) test(c *echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
