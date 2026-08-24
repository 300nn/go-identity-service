package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/300nn/go-identity-service/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadDatabase("config.yml")
	if err != nil {
		logger.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DatabaseUrlWithSSL())
	if err != nil {
		logger.Error("open database", slog.Any("error", err))
		os.Exit(1)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			logger.Error("close database", slog.Any("error", err))
			os.Exit(1)
		}
	}(db)

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error("set goose dialect", slog.Any("error", err))
		os.Exit(1)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		logger.Error("run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println("migrations applied")
}
