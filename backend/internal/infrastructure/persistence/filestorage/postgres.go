package filestorage

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type FileStorage interface {
	Save(ctx context.Context, data []byte, mimeType string) (uuid.UUID, error)
	Get(ctx context.Context, id uuid.UUID) (data []byte, mimeType string, err error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PostgresStorage struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewPostgresStorage(db dbtx.TxManager) *PostgresStorage {
	return &PostgresStorage{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresStorage) Save(ctx context.Context, data []byte, mimeType string) (uuid.UUID, error) {
	id := uuid.New()

	query, args, err := s.psql.
		Insert("files").
		Columns("id", "data", "mime_type").
		Values(id, data, mimeType).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build insert query: %w", err)
	}

	if _, err := s.db.Exec(ctx, query, args...); err != nil {
		return uuid.Nil, fmt.Errorf("insert file: %w", err)
	}

	return id, nil
}

func (s *PostgresStorage) Get(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	query, args, err := s.psql.
		Select("data", "mime_type").
		From("files").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, "", fmt.Errorf("build select query: %w", err)
	}

	var data []byte
	var mimeType string
	if err := s.db.QueryRow(ctx, query, args...).Scan(&data, &mimeType); err != nil {
		return nil, "", fmt.Errorf("get file: %w", err)
	}

	return data, mimeType, nil
}

func (s *PostgresStorage) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := s.psql.
		Delete("files").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	if _, err := s.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}
