package setup

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
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
//
// Deprecated: prefer GetSharedPostgres + CreateTestDatabase so we don't pay a
// container startup cost (and docker daemon serialization) per suite.
func StartPostgres(ctx context.Context) (*PostgresContainer, error) {
	return startPostgres(ctx, "veziizi_test")
}

func startPostgres(ctx context.Context, dbName string) (*PostgresContainer, error) {
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername("veziizi"),
		postgres.WithPassword("veziizi"),
		// Multi-suite e2e shares a single container; each suite holds its own
		// pgxpool + watermill subscribers, so the default max_connections=100
		// is exhausted quickly. Bump it.
		testcontainers.WithCmd(
			"postgres",
			"-c", "max_connections=1000",
			"-c", "shared_buffers=256MB",
		),
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

// ============================================================================
// Shared PostgreSQL pool for the package: one container, many isolated DBs.
// ============================================================================

var (
	sharedPgOnce     sync.Once
	sharedPg         *PostgresContainer
	sharedPgErr      error
	sharedPgAdminDSN string // DSN to the bootstrap "postgres" database
)

// GetSharedPostgres returns a process-wide Postgres container. The first call
// starts the container; subsequent calls return the same instance. Each suite
// then creates its own database inside this container via CreateTestDatabase.
func GetSharedPostgres(ctx context.Context) (*PostgresContainer, error) {
	sharedPgOnce.Do(func() {
		// Use the default "postgres" database for the bootstrap DSN so we can
		// CREATE DATABASE for individual suites afterwards.
		sharedPg, sharedPgErr = startPostgres(ctx, "postgres")
		if sharedPgErr == nil {
			sharedPgAdminDSN = sharedPg.DSN
		}
	})
	return sharedPg, sharedPgErr
}

// StopSharedPostgres terminates the shared container. Call from TestMain after
// all suites have finished.
func StopSharedPostgres(ctx context.Context) error {
	if sharedPg == nil {
		return nil
	}
	return sharedPg.Stop(ctx)
}

// CreateTestDatabase creates a fresh database on the shared Postgres container
// and returns a DSN pointing at it. Database name is generated from `prefix`
// and a monotonically increasing counter so it stays under Postgres' 63-char
// limit while remaining readable in logs.
func CreateTestDatabase(ctx context.Context, prefix string) (string, error) {
	if sharedPg == nil {
		return "", fmt.Errorf("shared postgres not started; call GetSharedPostgres first")
	}

	dbName := nextTestDBName(prefix)

	adminDB, err := sql.Open("postgres", sharedPgAdminDSN)
	if err != nil {
		return "", fmt.Errorf("open admin db: %w", err)
	}
	defer func() { _ = adminDB.Close() }()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %q`, dbName)); err != nil {
		return "", fmt.Errorf("create database %s: %w", dbName, err)
	}

	// Build the DSN by swapping the database segment in the shared DSN. The
	// shared DSN looks like "postgres://user:pass@host:port/postgres?sslmode=disable".
	return rewriteDSNDatabase(sharedPgAdminDSN, dbName), nil
}

var (
	testDBCounterMu sync.Mutex
	testDBCounter   int
)

func nextTestDBName(prefix string) string {
	testDBCounterMu.Lock()
	defer testDBCounterMu.Unlock()
	testDBCounter++
	// Sanitize prefix for SQL identifier safety: only lowercase letters/digits.
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '_'
		}
	}, prefix)
	if clean == "" {
		clean = "e2e"
	}
	return fmt.Sprintf("e2e_%s_%d", clean, testDBCounter)
}

// rewriteDSNDatabase replaces the path component of a postgres:// URL with the
// given database name. Avoids pulling in net/url just to swap a single segment.
func rewriteDSNDatabase(dsn, newDB string) string {
	// "postgres://user:pass@host:port/<db>?..." -> swap <db>
	schemeIdx := strings.Index(dsn, "://")
	if schemeIdx < 0 {
		return dsn
	}
	afterScheme := dsn[schemeIdx+3:]
	slash := strings.Index(afterScheme, "/")
	if slash < 0 {
		return dsn
	}
	rest := afterScheme[slash+1:]
	query := ""
	if q := strings.Index(rest, "?"); q >= 0 {
		query = rest[q:]
	}
	return dsn[:schemeIdx+3] + afterScheme[:slash+1] + newDB + query
}
