package state

import (
	"log"
	"os"

	"github.com/EvieePy/Echo/models"
	"github.com/labstack/echo/v5"
	"gopkg.in/yaml.v3"
)

type Context struct {
	Server *echo.Echo
	Config *models.Config
}

func NewContext() *Context {
	// Echo Web Server...
	server := echo.New()

	// Config...
	data, err := os.ReadFile("config.yaml")
	var config models.Config

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Unhandled exception. Unable to load 'config.yaml': %v", err)
	}

	// Context...
	ctx := Context{server, &config}
	return &ctx
}
