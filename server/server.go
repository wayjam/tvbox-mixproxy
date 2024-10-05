package server

import (
	"fmt"
	"io"
	"os"

	"github.com/gofiber/fiber/v3"
	fiberlog "github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/wayjam/tvbox-mixproxy/config"
	"github.com/wayjam/tvbox-mixproxy/pkg/mixer"
)

type server struct {
	app           *fiber.App
	cfg           *config.Config
	sourceManager *mixer.SourceManager
}

func NewServer(cfg *config.Config) *server {
	app := fiber.New()

	app.Use(recoverer.New())
	app.Use(requestid.New())

	// Configure logging middleware
	var logOutput io.Writer
	if cfg.Log.Output == "stdout" || cfg.Log.Output == "" {
		logOutput = os.Stdout
	} else {
		logOutput = &lumberjack.Logger{
			Filename:   cfg.Log.Output,
			MaxSize:    100, // megabytes
			MaxBackups: 3,
			MaxAge:     28, // days
		}
	}

	// Set up custom logger format
	fiberlog.SetOutput(logOutput)
	fiberlog.SetLevel(fiberlog.Level(cfg.Log.Level))

	app.Use(logger.New(logger.Config{
		Output:     logOutput,
		Format:     "${time} ${locals:requestid} [${level}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006/01/02 15:04:05.000000",
		TimeZone:   "Local",
	}))

	sourceManager := mixer.NewSourceManager(cfg.Sources)

	return &server{
		app:           app,
		cfg:           cfg,
		sourceManager: sourceManager,
	}
}

func (s *server) SetupRoutes(app *fiber.App) {
	app.Get("/", Home)
	app.Get("/logo", Logo)
	app.Get("/wallpaper", Wallpaper)

	v1 := app.Group("/v1")
	v1.Get("/repo", NewRepoHandler(s.cfg, s.sourceManager))
	v1.Get("/multi_repo", NewMultiRepoHandler(s.cfg, s.sourceManager))
	v1.Get("/spider", NewSpiderHandler(s.cfg, s.sourceManager))
}

func (s *server) Run() error {
	if !s.cfg.SingleRepoOpt.Disable {
		// Try MixRepo
		_, err := mixer.MixRepo(s.cfg, s.sourceManager)
		if err != nil {
			return fmt.Errorf("failed to initialize MixRepo: %w", err)
		}

	}

	if !s.cfg.MultiRepoOpt.Disable {
		// Try MixMultiRepo
		_, err := mixer.MixMultiRepo(s.cfg, s.sourceManager)
		if err != nil {
			return fmt.Errorf("failed to initialize MixMultiRepo: %w", err)
		}
	}

	s.SetupRoutes(s.app)

	return s.app.Listen(fmt.Sprintf(":%d", s.cfg.ServerPort))
}
