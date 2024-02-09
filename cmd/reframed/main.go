// Package main configures and runs application.
package main

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/rshelekhov/reframed/config"
	"github.com/rshelekhov/reframed/src/handlers"
	"github.com/rshelekhov/reframed/src/logger"
	"github.com/rshelekhov/reframed/src/server"
	"github.com/rshelekhov/reframed/src/server/middleware/jwtoken"
	"github.com/rshelekhov/reframed/src/storage/postgres"
	"log/slog"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.AppEnv)

	// A field with information about the current environment
	// will be added to each message
	log = log.With(slog.String("env", cfg.AppEnv))

	log.Info(
		"initializing server",
		slog.String("address", cfg.HTTPServer.Address))
	log.Debug("logger debug mode enabled")

	tokenAuth := jwtoken.NewJWTAuth(
		cfg.JWTAuth.Secret,
		jwt.SigningMethodHS256,
		cfg.JWTAuth.AccessTokenTTL,
		cfg.JWTAuth.RefreshTokenTTL,
		cfg.JWTAuth.RefreshTokenCookieDomain,
		cfg.JWTAuth.RefreshTokenCookiePath,
	)

	// Storage
	pg, err := postgres.NewStorage(cfg)
	if err != nil {
		log.Error("failed to init storage", logger.Err(err))
	}
	log.Debug("storage initiated")

	userStorage := postgres.NewUserStorage(pg)
	listStorage := postgres.NewListStorage(pg)
	taskStorage := postgres.NewTaskStorage(pg)
	headingStorage := postgres.NewHeadingStorage(pg)
	tagStorage := postgres.NewTagStorage(pg)

	// Handlers
	user := handlers.NewUserHandler(log, tokenAuth, userStorage, listStorage)
	list := handlers.NewListHandler(log, tokenAuth, listStorage, headingStorage)
	task := handlers.NewTaskHandler(log, tokenAuth, taskStorage, headingStorage, tagStorage)
	heading := handlers.NewHeadingHandler(log, tokenAuth, headingStorage)
	tag := handlers.NewTagHandler(log, tokenAuth, tagStorage)

	// HTTP Server
	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	srv := server.NewServer(cfg, log, tokenAuth, user, list, task, heading, tag)
	srv.Start()
}
