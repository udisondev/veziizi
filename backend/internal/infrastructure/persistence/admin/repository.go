package admin

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
)

type Repository struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewRepository(db dbtx.TxManager) *Repository {
	return &Repository{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type Admin struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Name         string    `db:"name"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*Admin, error) {
	query, args, err := r.psql.
		Select("id", "email", "password_hash", "name", "is_active", "created_at").
		From("platform_admins").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var a Admin
	if err := pgxscan.Get(ctx, r.db, &a, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get admin by email: %w", err)
	}

	return &a, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Admin, error) {
	query, args, err := r.psql.
		Select("id", "email", "password_hash", "name", "is_active", "created_at").
		From("platform_admins").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var a Admin
	if err := pgxscan.Get(ctx, r.db, &a, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get admin by id: %w", err)
	}

	return &a, nil
}
