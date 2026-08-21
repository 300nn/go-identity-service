package app

import (
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/config"
	userapiv1 "CrudTutorialProject/internal/gen/api/user/v1"
	"CrudTutorialProject/internal/grpcapi"
	"CrudTutorialProject/internal/httpserver"
	"CrudTutorialProject/internal/kafkaconsumer"
	"CrudTutorialProject/internal/middleware"
	"CrudTutorialProject/internal/outbox"
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
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ShutdownFunc func()

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

	tokenManager := auth.NewTokenManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.App.Name,
	)

	userRepo := user.NewPostgresRepository(dbPool)

	validator := validation.New()

	hasher := auth.NewPasswordHasher()

	initHealthModule(mux, dbPool, log, &shuttingDown, cfg)

	authMiddleware := initAuthModule(mux, cfg, log, userRepo, validator, hasher, dbPool, redisClient, tokenManager)

	userTxFactory := user.NewPostgresTxRepositoryFactory(dbPool)
	userCache := user.NewRedisCache(redisClient, "go-crud")
	userService := user.NewService(
		userRepo,
		userTxFactory,
		hasher,
		user.WithCache(userCache, cfg.Cache.UserTTL),
	)

	grpcShutdown, err := initGRPCModule(cfg, log, userService, tokenManager)
	if err != nil {
		return err
	}
	defer grpcShutdown()

	initUserModule(mux, log, userService, validator, authMiddleware)

	consumerShutdown, err := initKafkaConsumerModule(dbPool, log, cfg)
	if err != nil {
		return err
	}
	defer consumerShutdown()

	outboxShutdown, err := initOutboxModule(dbPool, log, cfg)
	if err != nil {
		return err
	}
	defer outboxShutdown()

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
	userService *user.Service,
	validator *validation.Validator,
	ware *auth.MiddleWare,
) {
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
	redisClient *redis.Client,
	tokenManager *auth.TokenManager,
) *auth.MiddleWare {

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

func initOutboxModule(
	dbPool *pgxpool.Pool,
	log *slog.Logger,
	cfg *config.Config,
) (ShutdownFunc, error) {

	outboxRepo := outbox.NewPostgresRepository(dbPool)

	kafkaPublisher, err := outbox.NewKafkaPublisher(outbox.KafkaPublisherConfig{
		Brokers:               cfg.Kafka.BrokerList(),
		Topic:                 cfg.Kafka.OutboxTopic,
		ProducerLinger:        cfg.Kafka.ProducerLinger,
		ProducerBatchMaxBytes: cfg.Kafka.ProducerBatchMaxBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("create kafka publisher: %w", err)
	}

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	done := make(chan struct{})

	outboxWorker := outbox.NewWorker(
		outboxRepo,
		kafkaPublisher,
		log,
		cfg.Worker,
	)

	go func() {
		defer close(done)

		if err := outboxWorker.Run(workerCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("outbox worker stopped with error", slog.Any("error", err))
		}
	}()

	var shutdownOnce sync.Once

	waitTimeout := cfg.Worker.ProcessTimeout + 5*time.Second

	shutdownFunc := func() {
		shutdownOnce.Do(func() {
			cancelWorker()

			select {
			case <-done:
			case <-time.After(waitTimeout):
				log.Warn("timeout waiting for outbox worker to stop")
			}

			kafkaPublisher.Close()
		})
	}

	return shutdownFunc, nil
}

func initKafkaConsumerModule(
	dbPool *pgxpool.Pool,
	log *slog.Logger,
	cfg *config.Config,
) (ShutdownFunc, error) {
	txFactory := kafkaconsumer.NewPostgresTxFactory(dbPool)

	router := kafkaconsumer.NewRouter()
	router.Register(
		outbox.EventTypeUserRegistered,
		kafkaconsumer.NewUserRegisteredHandler(log),
	)

	consumer, err := kafkaconsumer.NewConsumer(
		kafkaconsumer.Config{
			Brokers:       cfg.Kafka.BrokerList(),
			Topic:         cfg.Kafka.OutboxTopic,
			ConsumerGroup: cfg.Kafka.ConsumerGroup,
		},
		router,
		txFactory,
		log,
	)

	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		if err := consumer.Run(consumerCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("kafka consumer stopped with error", slog.Any("error", err))
		}
	}()
	var shutdownOnce sync.Once

	waitTimeout := cfg.Worker.ProcessTimeout + 5*time.Second

	shutdownFunc := func() {
		shutdownOnce.Do(func() {
			cancelConsumer()

			select {
			case <-done:
			case <-time.After(waitTimeout):
				log.Warn("timeout waiting for kafka consumer to stop")
			}

			consumer.Close()
		})
	}

	return shutdownFunc, nil
}

func initGRPCModule(
	cfg *config.Config,
	log *slog.Logger,
	userService *user.Service,
	tokenManager *auth.TokenManager,
) (ShutdownFunc, error) {
	listener, err := net.Listen("tcp", cfg.GRPC.Address())

	if err != nil {
		return nil, fmt.Errorf("create gRPC listener: %w", err)
	}

	authInterceptor := grpcapi.NewAuthInterceptor(tokenManager)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(authInterceptor.Unary()),
	)

	userapiv1.RegisterUserServiceServer(
		grpcServer,
		grpcapi.NewUserService(userService),
	)

	if cfg.App.Environment == "local" {
		reflection.Register(grpcServer)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		log.Info("starting grpc server", "addr", cfg.GRPC.Address())

		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Error("gRPC server stopped with error", slog.Any("error", err))
		}
	}()

	var shutdownOnce sync.Once

	shutdown := func() {
		shutdownOnce.Do(func() {
			stopped := make(chan struct{})

			go func() {
				grpcServer.GracefulStop()
				close(stopped)
			}()

			select {
			case <-stopped:
			case <-time.After(cfg.GRPC.ShutdownTimeout):
				log.Warn("timeout waiting for grpc graceful stop")
				grpcServer.Stop()
			}

			<-done
		})
	}

	return shutdown, nil
}
