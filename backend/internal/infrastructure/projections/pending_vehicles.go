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

type PendingVehiclesProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewPendingVehiclesProjection(db dbtx.TxManager) *PendingVehiclesProjection {
	return &PendingVehiclesProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type PendingVehicle struct {
	ID                 uuid.UUID `db:"id"`
	OrgID              uuid.UUID `db:"org_id"`
	RegistrationNumber string    `db:"registration_number"`
	Brand              *string   `db:"brand"`
	Model              *string   `db:"model"`
	VehicleType        string    `db:"vehicle_type"`
	VehicleSubType     string    `db:"vehicle_subtype"`
	SubmittedAt        time.Time `db:"submitted_at"`
}

func (p *PendingVehiclesProjection) Upsert(ctx context.Context, v PendingVehicle) error {
	query, args, err := p.psql.
		Insert("pending_vehicles").
		Columns("id", "org_id", "registration_number", "brand", "model", "vehicle_type", "vehicle_subtype", "submitted_at").
		Values(v.ID, v.OrgID, v.RegistrationNumber, v.Brand, v.Model, v.VehicleType, v.VehicleSubType, v.SubmittedAt).
		Suffix(`ON CONFLICT (id) DO UPDATE SET
            registration_number = EXCLUDED.registration_number,
            brand = EXCLUDED.brand,
            model = EXCLUDED.model,
            vehicle_type = EXCLUDED.vehicle_type,
            vehicle_subtype = EXCLUDED.vehicle_subtype,
            submitted_at = EXCLUDED.submitted_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build pending_vehicles upsert: %w", err)
	}
	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert pending vehicle: %w", err)
	}
	return nil
}

func (p *PendingVehiclesProjection) Remove(ctx context.Context, id uuid.UUID) error {
	query, args, err := p.psql.Delete("pending_vehicles").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build pending_vehicles delete: %w", err)
	}
	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete pending vehicle: %w", err)
	}
	return nil
}

// List returns all vehicles awaiting moderation, newest first.
func (p *PendingVehiclesProjection) List(ctx context.Context) ([]PendingVehicle, error) {
	query, args, err := p.psql.
		Select("id", "org_id", "registration_number", "brand", "model", "vehicle_type", "vehicle_subtype", "submitted_at").
		From("pending_vehicles").
		OrderBy("submitted_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build pending_vehicles select: %w", err)
	}
	var rows []PendingVehicle
	if err := pgxscan.Select(ctx, p.db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list pending vehicles: %w", err)
	}
	return rows, nil
}
