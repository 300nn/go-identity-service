package testutils

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	redisOnce sync.Once
	redisMu   sync.Mutex

	redisClient    *redis.Client
	redisContainer testcontainers.Container
	redisErr       error
)

func NewTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping redis test in short mode")
	}

	redisMu.Lock()

	t.Cleanup(func() {
		if redisClient != nil {
			cleanRedis(t, redisClient)
		}
		redisMu.Unlock()
	})

	client := sharedRedisClient(t)

	cleanRedis(t, client)

	return client
}

func sharedRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	redisOnce.Do(func() {
		ctx := context.Background()

		req := testcontainers.ContainerRequest{
			Image:        "redis:8-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		}

		redisContainer, redisErr = testcontainers.GenericContainer(
			ctx,
			testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			},
		)
		if redisErr != nil {
			return
		}

		host, err := redisContainer.Host(ctx)
		if err != nil {
			redisErr = err
			return
		}

		port, err := redisContainer.MappedPort(ctx, "6379")
		if err != nil {
			redisErr = err
			return
		}

		redisClient = redis.NewClient(&redis.Options{
			Addr: host + ":" + port.Port(),
		})

		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisErr = err
			return
		}
	})

	if redisErr != nil {
		t.Fatalf("start shared redis: %v", redisErr)
	}

	return redisClient
}

func cleanRedis(t *testing.T, client *redis.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("clean redis: %v", err)
	}
}
