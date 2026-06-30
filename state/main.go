package state

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/EvieePy/Echo/database"
	"github.com/EvieePy/Echo/logger"
	"github.com/EvieePy/Echo/models"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gopkg.in/yaml.v3"
)

const purgeDeletedInterval = 12 * time.Hour

type Context struct {
	Server        *echo.Echo
	Config        *models.Config
	Logger        *logger.Logger
	RequestLogger *logger.Logger
	Database      database.Database
	VersionInfo   *VersionInfo
	Valkey        *database.Valkey
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

func loadValkey(config *models.Config, appLogger *logger.Logger) *database.Valkey {
	if config.Cache.DSN == "" {
		appLogger.Warn("No Valkey configuration was found. Ratelimits will not be implemented.")
		return nil
	}

	valk, err := database.NewValkey(config)
	if err != nil {
		panic(err)
	}
	appLogger.Info("Successfully connected to Valkey.")
	return valk
}

func startPurgeLoop(db database.Database, appLogger *logger.Logger) {
	ticker := time.NewTicker(purgeDeletedInterval)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			if err := db.PurgeDeleted(); err != nil {
				appLogger.Errorf("Failed to purge soft-deleted data: %v", err)
			}
		}
	}()
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
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Safety-Token"},
	}))
}

func NewContext() *Context {
	_, dockerNet, _ := net.ParseCIDR("172.16.0.0/12")
	server := echo.New()
	server.IPExtractor = echo.ExtractIPFromXFFHeader(
		echo.TrustIPRange(dockerNet),
		echo.TrustLoopback(true),
	)

	appLogger := logger.New("APP", logger.ColourApp)
	reqLogger := logger.New("REQUEST", logger.ColourRequest)
	server.Logger = slog.New(logger.NewHandler("SERVER", logger.ColourServer))

	config := loadConfig(appLogger)
	db := loadDatabase(&config, appLogger)
	startPurgeLoop(db, appLogger)
	loadMiddleware(server, &config, reqLogger)
	verInfo := NewVersionInfo(appLogger)
	valk := loadValkey(&config, appLogger)

	return &Context{
		Server:        server,
		Config:        &config,
		Logger:        appLogger,
		RequestLogger: reqLogger,
		Database:      db,
		VersionInfo:   &verInfo,
		Valkey:        valk,
	}
}
