package app

import (
	"CrudTutorialProject/internal/config"
	"CrudTutorialProject/internal/httpserver"
	"CrudTutorialProject/internal/middleware"
	"CrudTutorialProject/internal/postgres"
	"CrudTutorialProject/internal/response"
	"CrudTutorialProject/internal/user"
	"CrudTutorialProject/internal/validation"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"time"
)

func Run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {

	mux := http.NewServeMux()

	dbPool, err := postgres.NewPool(ctx, &cfg.Database)

	if err != nil {
		return err
	}

	defer dbPool.Close()

	initUserModule(mux, log, dbPool)

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"status": "ready",
		})
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"message": "pong",
		})
	})

	mux.HandleFunc("GET /info", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"name":        cfg.App.Name,
			"version":     cfg.App.Version,
			"environment": cfg.App.Environment,
		})
	})

	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"name":    cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
		panic("user_test panic")
	})

	handler := middleware.Chain(
		mux,
		middleware.RequestId,
		middleware.Logging(log),
		middleware.Recovery(log),
	)

	server := httpserver.New(
		cfg.HTTP,
		handler,
	)

	errCh := make(chan error, 1)

	go func() {
		log.Info("starting server", "addr", server.Addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutDownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutDownCtx); err != nil {
		return fmt.Errorf("shuthdown server: %w", err)
	}

	log.Info("server stopped gracefully")
	return nil
}

func initUserModule(mux *http.ServeMux, logger *slog.Logger, pool *pgxpool.Pool) {
	validator := validation.New()

	userRepo := user.NewPostgresRepository(pool)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	userService := user.NewService(userRepo, txFactory)
	userHandler := user.NewHandler(userService, logger, validator)
	userHandler.RegisterRouts(mux)
}
