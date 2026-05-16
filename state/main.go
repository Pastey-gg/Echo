package state

import (
	"log/slog"
	"os"

	"github.com/EvieePy/Echo/logger"
	"github.com/EvieePy/Echo/models"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gopkg.in/yaml.v3"
)

type Context struct {
	Server        *echo.Echo
	Config        *models.Config
	Logger        *logger.Logger
	RequestLogger *logger.Logger
}

func NewContext() *Context {
	server := echo.New()

	appLogger := logger.New("APP", logger.ColourApp)
	reqLogger := logger.New("REQUEST", logger.ColourRequest)

	server.Logger = slog.New(logger.NewHandler("SERVER", logger.ColourServer))

	server.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:  true,
		LogMethod:  true,
		LogURI:     true,
		LogLatency: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				reqLogger.Errorf("%-7s %s → %d (%v) error=%v", v.Method, v.URI, v.Status, v.Latency, v.Error)
			} else {
				reqLogger.Infof("%-7s %s → %d (%v)", v.Method, v.URI, v.Status, v.Latency)
			}
			return nil
		},
	}))

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		appLogger.Fatalf("unable to read 'config.yaml': %v", err)
	}

	var config models.Config
	if err = yaml.Unmarshal(data, &config); err != nil {
		appLogger.Fatalf("unable to parse 'config.yaml': %v", err)
	}

	return &Context{Server: server, Config: &config, Logger: appLogger, RequestLogger: reqLogger}
}
