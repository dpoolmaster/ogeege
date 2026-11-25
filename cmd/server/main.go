package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"oge-ege/internal/shared/config"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/httpserver"
	"oge-ege/internal/shared/logging"
	"oge-ege/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("загрузить конфигурацию: %w", err)
	}

	logger := logging.New(cfg.Log.Level)
	setGinMode(cfg.App.Env)

	pool, err := db.Open(context.Background(), cfg.DB.URL, cfg.DB.MinConns, cfg.DB.MaxConns)
	if err != nil {
		return fmt.Errorf("подключиться к базе данных: %w", err)
	}
	defer pool.Close()

	latestMigration, err := migrations.LatestVersion()
	if err != nil {
		return fmt.Errorf("определить версию миграций: %w", err)
	}

	router := httpserver.NewRouter(httpserver.Options{
		Env:              cfg.App.Env,
		Logger:           logger,
		MaxBodyBytes:     cfg.HTTP.MaxBodyBytes,
		Pinger:           pool,
		MigrationChecker: db.NewMigrationChecker(db.NewPgMigrationStateReader(pool), latestMigration),
	})

	server := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP-сервер запущен", "addr", cfg.HTTP.Addr, "env", cfg.App.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("запустить HTTP-сервер: %w", err)
	case sig := <-stop:
		logger.Info("получен сигнал остановки", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("остановить HTTP-сервер: %w", err)
	}

	logger.Info("HTTP-сервер остановлен")

	return nil
}

func setGinMode(env string) {
	switch env {
	case "local", "development":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
}
