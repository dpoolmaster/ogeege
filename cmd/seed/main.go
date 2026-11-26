package main

import (
	"context"
	"fmt"
	"os"

	"oge-ege/internal/content"
	"oge-ege/internal/content/seed"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/config"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("загрузить конфигурацию: %w", err)
	}

	logger := logging.New(cfg.Log.Level)
	ctx := context.Background()

	pool, err := db.Open(ctx, cfg.DB.URL, cfg.DB.MinConns, cfg.DB.MaxConns)
	if err != nil {
		return fmt.Errorf("подключиться к базе данных: %w", err)
	}
	defer pool.Close()

	txManager := db.NewTxManager(pool)
	service := content.NewService(content.NewPgRepository(), pool, txManager, clock.New(), nil, logger)

	if err := seed.Run(ctx, service, logger); err != nil {
		return err
	}

	logger.Info("стартовый контент создан")

	return nil
}
