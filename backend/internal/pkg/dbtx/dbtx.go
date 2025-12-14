package dbtx

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

var TxKey txKey

type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type txOpt struct {
	pgx.TxOptions
}

type TxOpt func(*txOpt)

func WithIsolationLevel(l pgx.TxIsoLevel) TxOpt {
	return func(o *txOpt) { o.IsoLevel = l }
}

func WithAccessMode(m pgx.TxAccessMode) TxOpt {
	return func(o *txOpt) { o.AccessMode = m }
}

type TxManager interface {
	Conn
	InTx(ctx context.Context, fn func(ctx context.Context) error, opts ...TxOpt) error
}

type TxExecutor struct {
	pool *pgxpool.Pool
}

func NewTxExecutor(pool *pgxpool.Pool) *TxExecutor {
	return &TxExecutor{pool: pool}
}

func FromCtx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(TxKey).(pgx.Tx)
	return tx, ok
}

func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

func (e *TxExecutor) InTx(ctx context.Context, fn func(ctx context.Context) error, opts ...TxOpt) error {
	// If already in transaction, create savepoint
	if tx, ok := FromCtx(ctx); ok {
		return inSavepoint(ctx, tx, fn)
	}

	opt := txOpt{
		TxOptions: pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		},
	}
	for apply := range slices.Values(opts) {
		apply(&opt)
	}

	tx, err := e.pool.BeginTx(ctx, opt.TxOptions)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := fn(WithTx(ctx, tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func inSavepoint(ctx context.Context, tx pgx.Tx, fn func(ctx context.Context) error) error {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin savepoint: %w", err)
	}
	defer sp.Rollback(ctx) //nolint:errcheck

	if err := fn(WithTx(ctx, sp)); err != nil {
		return err
	}

	return sp.Commit(ctx)
}

// Exec executes query using tx from context or pool
func (e *TxExecutor) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := FromCtx(ctx); ok {
		return tx.Exec(ctx, sql, args...)
	}
	return e.pool.Exec(ctx, sql, args...)
}

// Query executes query using tx from context or pool
func (e *TxExecutor) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx, ok := FromCtx(ctx); ok {
		return tx.Query(ctx, sql, args...)
	}
	return e.pool.Query(ctx, sql, args...)
}

// QueryRow executes query using tx from context or pool
func (e *TxExecutor) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx, ok := FromCtx(ctx); ok {
		return tx.QueryRow(ctx, sql, args...)
	}
	return e.pool.QueryRow(ctx, sql, args...)
}
