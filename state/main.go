package state

import "github.com/labstack/echo/v5"

type Context struct {
	Server *echo.Echo
}

func NewContext() *Context {
	server := echo.New()
	ctx := Context{server}

	return &ctx
}
