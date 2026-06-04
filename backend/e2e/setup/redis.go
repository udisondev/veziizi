package setup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer wraps a testcontainers Redis container.
type RedisContainer struct {
	container testcontainers.Container
	Addr      string // host:port
}

// Stop terminates the Redis container.
func (c *RedisContainer) Stop(ctx context.Context) error {
	if c.container == nil {
		return nil
	}
	return c.container.Terminate(ctx)
}

// ============================================================================
// Shared Redis for the package: one container, isolated DB index per suite.
// ============================================================================

var (
	sharedRedisOnce sync.Once
	sharedRedis     *RedisContainer
	sharedRedisErr  error

	redisDBCounterMu sync.Mutex
	redisDBCounter   int
)

// redisDatabases — количество логических БД в тестовом Redis. По одной на
// suite; дефолтных 16 не хватает при больших прогонах.
const redisDatabases = 2048

// GetSharedRedis возвращает process-wide Redis контейнер (по аналогии с
// GetSharedPostgres). Каждый suite получает собственный DB index через
// NextRedisURL — нативная изоляция Redis без shared state между сьютами.
func GetSharedRedis(ctx context.Context) (*RedisContainer, error) {
	sharedRedisOnce.Do(func() {
		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "redis:7-alpine",
				ExposedPorts: []string{"6379/tcp"},
				Cmd:          []string{"redis-server", "--databases", fmt.Sprintf("%d", redisDatabases)},
				WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
			},
			Started: true,
		})
		if err != nil {
			sharedRedisErr = fmt.Errorf("start redis container: %w", err)
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			sharedRedisErr = fmt.Errorf("get redis host: %w", err)
			return
		}
		port, err := container.MappedPort(ctx, "6379/tcp")
		if err != nil {
			sharedRedisErr = fmt.Errorf("get redis port: %w", err)
			return
		}

		sharedRedis = &RedisContainer{
			container: container,
			Addr:      fmt.Sprintf("%s:%s", host, port.Port()),
		}
	})
	return sharedRedis, sharedRedisErr
}

// StopSharedRedis terminates the shared container. Call from TestMain after
// all suites have finished.
func StopSharedRedis(ctx context.Context) error {
	if sharedRedis == nil {
		return nil
	}
	return sharedRedis.Stop(ctx)
}

// NextRedisURL выделяет следующий свободный DB index в shared Redis и
// возвращает URL вида redis://host:port/<n>.
func NextRedisURL() (string, error) {
	if sharedRedis == nil {
		return "", fmt.Errorf("shared redis not started; call GetSharedRedis first")
	}

	redisDBCounterMu.Lock()
	defer redisDBCounterMu.Unlock()
	if redisDBCounter >= redisDatabases {
		return "", fmt.Errorf("redis db index pool exhausted (%d databases)", redisDatabases)
	}
	db := redisDBCounter
	redisDBCounter++

	return fmt.Sprintf("redis://%s/%d", sharedRedis.Addr, db), nil
}
