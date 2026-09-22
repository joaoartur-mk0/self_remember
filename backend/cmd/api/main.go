package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "self-remember/backend/internal/adapters/http"
	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/app/auth"
	"self-remember/backend/internal/platform/config"
	"self-remember/backend/internal/platform/database"
	"self-remember/backend/internal/platform/logging"
	"self-remember/backend/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel)
	logger.Info("starting", "http_addr", cfg.HTTPAddr)

	sqlDB, err := database.OpenSQL(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := database.Migrate(context.Background(), sqlDB, migrations.FS); err != nil {
		logger.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}

	gormDB, err := database.NewGORM(sqlDB)
	if err != nil {
		logger.Error("failed to initialize gorm", "error", err)
		os.Exit(1)
	}

	tokens := auth.NewTokenIssuer(cfg.JWTSecret)
	users := postgres.NewUserRepository(gormDB)
	refreshTokens := postgres.NewRefreshTokenRepository(gormDB)
	loginUseCase := auth.NewLoginUseCase(users, refreshTokens, tokens, time.Now)
	refreshUseCase := auth.NewRefreshUseCase(refreshTokens, tokens, time.Now)
	authHandler := httpadapter.NewAuthHandler(loginUseCase, refreshUseCase)

	router := httpadapter.NewRouter(authHandler, tokens)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
