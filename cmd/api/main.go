package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"CrudTutorialProject/internal/app"
	"CrudTutorialProject/internal/config"
	"CrudTutorialProject/internal/logger"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		slog.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.New(cfg.Log.Level)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer stop()

	if err := app.Run(ctx, cfg, log); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Error("application stopped with error", slog.Any("error", err))
			os.Exit(1)
		}
	}

	log.Info("application stopped")
}
