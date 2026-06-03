package projections

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type OrganizationsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewOrganizationsProjection(db dbtx.TxManager) *OrganizationsProjection {
	return &OrganizationsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type OrganizationLookup struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	LegalName string    `db:"legal_name"`
	INN       string    `db:"inn"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	// Version — версия агрегата на момент rebuild'а проекции. Guard в Upsert
	// отбрасывает запись устаревшего состояния (см. rebuild-from-aggregate).
	Version int64 `db:"version"`
}

// GetByID retrieves organization by ID
func (p *OrganizationsProjection) GetByID(ctx context.Context, id uuid.UUID) (*OrganizationLookup, error) {
	query, args, err := p.psql.
		Select("id", "name", "legal_name", "inn", "status", "created_at", "updated_at").
		From("organizations_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var org OrganizationLookup
	if err := pgxscan.Get(ctx, p.db, &org, query, args...); err != nil {
		return nil, fmt.Errorf("get organization by id: %w", err)
	}

	return &org, nil
}

// GetNames возвращает названия организаций по их ID
func (p *OrganizationsProjection) GetNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]string), nil
	}

	query, args, err := p.psql.
		Select("id", "name").
		From("organizations_lookup").
		Where(squirrel.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	type idName struct {
		ID   uuid.UUID `db:"id"`
		Name string    `db:"name"`
	}

	var rows []idName
	if err := pgxscan.Select(ctx, p.db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get organization names: %w", err)
	}

	result := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.Name
	}

	return result, nil
}

// Upsert пишет полное состояние организации с version-guard'ом: устаревший
// rebuild (меньшая версия агрегата) не перетирает свежие данные, повторная
// запись той же версии идемпотентна.
func (p *OrganizationsProjection) Upsert(ctx context.Context, org OrganizationLookup) error {
	query := `
		INSERT INTO organizations_lookup (id, name, legal_name, inn, status, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			legal_name = EXCLUDED.legal_name,
			inn = EXCLUDED.inn,
			status = EXCLUDED.status,
			updated_at = NOW(),
			version = EXCLUDED.version
		WHERE organizations_lookup.version <= EXCLUDED.version
	`

	if _, err := p.db.Exec(ctx, query, org.ID, org.Name, org.LegalName, org.INN, org.Status, org.CreatedAt, org.Version); err != nil {
		return fmt.Errorf("upsert organization: %w", err)
	}

	return nil
}
