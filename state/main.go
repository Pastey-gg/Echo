package state

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/EvieePy/Echo/database"
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
	Database      database.Database
	VersionInfo   *VersionInfo
}

func loadConfig(appLogger *logger.Logger) models.Config {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		appLogger.Fatalf("Unable to read 'config.yaml': %v", err)
	}

	var config models.Config
	if err = yaml.Unmarshal(data, &config); err != nil {
		appLogger.Fatalf("Unable to parse 'config.yaml': %v", err)
	}

	return config
}

func loadDatabase(config *models.Config, appLogger *logger.Logger) database.Database {
	var db database.Database

	if config.Database.DSN != "" {
		pg, err := database.NewPostgres(config)
		if err != nil {
			appLogger.Fatalf("Unable to connect to database: %v", err)
		}
		db = pg
		appLogger.Infof("Successfully connected to Database.")
	} else {
		db = database.NewMemory(config)
		appLogger.Infof("No Database DSN provided. Using in-memory stores instead.")
	}

	return db
}

func loadMiddleware(server *echo.Echo, config *models.Config, reqLogger *logger.Logger) {
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

	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowMethods: []string{http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodDelete},
		AllowOrigins: config.General.AllowedOrigins,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
}

func NewContext() *Context {
	server := echo.New()

	appLogger := logger.New("APP", logger.ColourApp)
	reqLogger := logger.New("REQUEST", logger.ColourRequest)
	server.Logger = slog.New(logger.NewHandler("SERVER", logger.ColourServer))

	config := loadConfig(appLogger)
	db := loadDatabase(&config, appLogger)
	loadMiddleware(server, &config, reqLogger)
	verInfo := NewVersionInfo(appLogger)

	return &Context{
		Server:        server,
		Config:        &config,
		Logger:        appLogger,
		RequestLogger: reqLogger,
		Database:      db,
		VersionInfo:   &verInfo,
	}
}
