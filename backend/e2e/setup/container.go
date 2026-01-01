package setup

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers PostgreSQL container.
type PostgresContainer struct {
	container *postgres.PostgresContainer
	DSN       string
}

// StartPostgres starts a PostgreSQL container for E2E tests.
func StartPostgres(ctx context.Context) (*PostgresContainer, error) {
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("veziizi_test"),
		postgres.WithUsername("veziizi"),
		postgres.WithPassword("veziizi"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		if termErr := container.Terminate(ctx); termErr != nil {
			return nil, fmt.Errorf("failed to get connection string: %w (terminate error: %v)", err, termErr)
		}
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &PostgresContainer{
		container: container,
		DSN:       dsn,
	}, nil
}

// Stop terminates the PostgreSQL container.
func (c *PostgresContainer) Stop(ctx context.Context) error {
	if c.container == nil {
		return nil
	}
	return c.container.Terminate(ctx)
}
