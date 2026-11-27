package main

import (
	"context"
	"fmt"
	"os"

	"oge-ege/internal/content"
	contentseed "oge-ege/internal/content/seed"
	"oge-ege/internal/publicsite"
	publicseed "oge-ege/internal/publicsite/seed"
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
	clk := clock.New()
	service := content.NewService(content.NewPgRepository(), pool, txManager, clk, nil, logger)

	if err := contentseed.Run(ctx, service, logger); err != nil {
		return err
	}

	publicService := publicsite.NewService(
		publicsite.NewPgRepository(), pool, txManager, clk, service, nil, nil, logger,
	)
	if err := publicseed.Run(ctx, publicService, logger); err != nil {
		return err
	}

	logger.Info("стартовый контент создан")

	return nil
}
