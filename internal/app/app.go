package app

import (
	"CrudTutorialProject/internal/auth"
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
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	var shuttingDown atomic.Bool

	mux := http.NewServeMux()

	dbPool, err := postgres.NewPool(ctx, &cfg.Database)

	if err != nil {
		return err
	}

	defer dbPool.Close()

	userRepo := user.NewPostgresRepository(dbPool)

	validator := validation.New()

	initHealthModule(mux, dbPool, log, &shuttingDown, cfg)

	authMiddleware := initAuthModule(mux, cfg, log, userRepo, validator)

	initUserModule(mux, log, dbPool, userRepo, validator, authMiddleware)

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

	shuttingDown.Store(true)

	shutDownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutDownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	log.Info("server stopped gracefully")
	return nil
}

func initHealthModule(mux *http.ServeMux, dbPool *pgxpool.Pool, log *slog.Logger, shuttingDown *atomic.Bool, cfg *config.Config) {
	healthHandlers := NewHealthHandlers(dbPool, log, shuttingDown)

	mux.HandleFunc("GET /ready", healthHandlers.Ready)

	mux.HandleFunc("GET /health", healthHandlers.Health)

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

	if cfg.App.Environment == "local" {
		mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
			panic("user_test panic")
		})
	}

}

func initUserModule(mux *http.ServeMux, logger *slog.Logger, pool *pgxpool.Pool, userRepo user.Repository, validator *validation.Validator, ware *auth.MiddleWare) {
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	userService := user.NewService(userRepo, txFactory)
	userHandler := user.NewHandler(userService, logger, validator)
	userHandler.RegisterRouts(
		mux,
		ware.RequireRole(user.RoleAdmin),
		ware.RequireSelfOrRole(
			auth.PathInt64Param("id"),
			user.RoleAdmin,
		),
	)
}

func initAuthModule(mux *http.ServeMux, cfg *config.Config, log *slog.Logger, userRepo user.Repository, validator *validation.Validator) *auth.MiddleWare {
	tokenManager := auth.NewTokenManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.App.Name,
	)

	authService := auth.NewService(
		userRepo,
		auth.NewPasswordHasher(),
		tokenManager,
	)

	authHandler := auth.NewHandler(authService, log, validator)
	authMiddleware := auth.NewMiddleWare(tokenManager)

	authHandler.RegisterRoutes(mux, authMiddleware)

	return authMiddleware
}
