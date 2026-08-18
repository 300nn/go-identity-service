package app

import (
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/config"
	"CrudTutorialProject/internal/httpserver"
	"CrudTutorialProject/internal/middleware"
	"CrudTutorialProject/internal/postgres"
	"CrudTutorialProject/internal/ratelimit"
	"CrudTutorialProject/internal/redisclient"
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
	"github.com/redis/go-redis/v9"
)

func Run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	var shuttingDown atomic.Bool

	mux := http.NewServeMux()

	dbPool, err := postgres.NewPool(ctx, &cfg.Database)

	if err != nil {
		return err
	}
	defer dbPool.Close()

	redisClient, err := redisclient.New(ctx, cfg.Redis)

	if err != nil {
		return err
	}
	defer func(redisClient *redis.Client) {
		err := redisClient.Close()
		if err != nil {
			log.Error("closing redis connection", slog.Any("error", err))
		}
	}(redisClient)

	userRepo := user.NewPostgresRepository(dbPool)

	validator := validation.New()

	hasher := auth.NewPasswordHasher()

	initHealthModule(mux, dbPool, log, &shuttingDown, cfg)

	authMiddleware := initAuthModule(mux, cfg, log, userRepo, validator, hasher, dbPool, redisClient)

	initUserModule(mux, log, dbPool, userRepo, validator, authMiddleware, hasher, redisClient, cfg)

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

func initUserModule(
	mux *http.ServeMux,
	logger *slog.Logger,
	pool *pgxpool.Pool,
	userRepo user.Repository,
	validator *validation.Validator,
	ware *auth.MiddleWare,
	hasher user.Hasher,
	redisClient *redis.Client,
	cfg *config.Config,
) {
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	userCache := user.NewRedisCache(redisClient, "go-crud")
	userService := user.NewService(
		userRepo,
		txFactory,
		hasher,
		user.WithCache(userCache, cfg.Cache.UserTTL))
	userHandler := user.NewHandler(userService, logger, validator)
	userHandler.RegisterRoutes(
		mux,
		ware.RequireRole(user.RoleAdmin),
		ware.RequireSelfOrRole(
			auth.PathInt64Param("id"),
			user.RoleAdmin,
		),
	)
}

func initAuthModule(
	mux *http.ServeMux,
	cfg *config.Config,
	log *slog.Logger,
	userRepo user.Repository,
	validator *validation.Validator,
	hasher *auth.PasswordHasher,
	db *pgxpool.Pool,
	redisClient *redis.Client) *auth.MiddleWare {

	tokenManager := auth.NewTokenManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.App.Name,
	)

	refreshStore := auth.NewRefreshTokenRepository(db)

	refreshTokens := auth.NewRefreshTokenManager()

	txFactory := auth.NewPostgresTxFactory(db)

	authService := auth.NewService(
		userRepo,
		refreshStore,
		txFactory,
		hasher,
		tokenManager,
		refreshTokens,
		cfg.Auth.RefreshTokenTTL,
	)

	limiter := ratelimit.NewRedisLimiter(redisClient, "go-crud")

	authHandler := auth.NewHandler(authService, log, validator, limiter, cfg.RateLimit)
	authMiddleware := auth.NewMiddleWare(tokenManager)

	authHandler.RegisterRoutes(mux, authMiddleware)

	return authMiddleware
}
