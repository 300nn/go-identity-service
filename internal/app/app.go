package app

import (
	"CrudTutorialProject/internal/config"
	"CrudTutorialProject/internal/httpserver"
	"CrudTutorialProject/internal/response"
	"CrudTutorialProject/internal/user"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() error {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	mux := http.NewServeMux()

	initUserModule(mux, logger)

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
			"name":        cfg.AppName,
			"version":     cfg.Version,
			"environment": cfg.Environment,
		})
	})

	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"name":    cfg.AppName,
			"version": cfg.Version,
		})
	})

	server := httpserver.New(httpserver.Config{
		Port: cfg.HttpPort,
	}, mux)

	errCh := make(chan error, 1)

	go func() {
		logger.Info("starting server", "addr", server.Addr)

		if err := server.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		errCh <- nil
	}()

	shutDownCh := make(chan os.Signal, 1)
	signal.Notify(shutDownCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-shutDownCh:
		logger.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		logger.Info("server stopped gracefully")
		return nil
	}

}

func initUserModule(mux *http.ServeMux, logger *slog.Logger) {
	userRepo := user.NewMemoryRepository()
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService, logger)
	userHandler.RegisterRouts(mux)
}
