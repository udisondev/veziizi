package projections

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type VehiclesProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewVehiclesProjection(db dbtx.TxManager) *VehiclesProjection {
	return &VehiclesProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// VehicleListItem is the denormalized data stored in vehicles_lookup.
type VehicleListItem struct {
	ID                 uuid.UUID `json:"id"`
	OrgID              uuid.UUID `json:"org_id"`
	RegistrationNumber string    `json:"registration_number"`
	Brand              *string   `json:"brand,omitempty"`
	Model              *string   `json:"model,omitempty"`
	VehicleType        string    `json:"vehicle_type"`
	VehicleSubType     string    `json:"vehicle_subtype"`
	LoadingTypes       []string  `json:"loading_types,omitempty"`
	Capacity           *float64  `json:"capacity,omitempty"`
	Volume             *float64  `json:"volume,omitempty"`
	Length             *float64  `json:"length,omitempty"`
	Width              *float64  `json:"width,omitempty"`
	Height             *float64  `json:"height,omitempty"`
	RequiresADR        bool      `json:"requires_adr"`
	HasTemperature     bool      `json:"has_temperature"`
	TempMin            *float64  `json:"temp_min,omitempty"`
	TempMax            *float64  `json:"temp_max,omitempty"`
	Thermograph        bool      `json:"thermograph"`
	Status             string    `json:"status"`
	RejectionReason    *string   `json:"rejection_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type VehicleFilterOption func(squirrel.SelectBuilder) squirrel.SelectBuilder

func WithVehicleOrgID(id uuid.UUID) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"org_id": id})
	}
}

func WithVehicleStatus(status string) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"status": status})
	}
}

func WithVehicleStatuses(statuses []string) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(statuses) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"status": statuses})
	}
}

func WithFleetVehicleType(vt string) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"vehicle_type": vt})
	}
}

func WithFleetVehicleSubType(sub string) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"vehicle_subtype": sub})
	}
}

func WithMinCapacity(value float64) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.GtOrEq{"capacity": value})
	}
}

func WithMinFleetVolume(value float64) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.GtOrEq{"volume": value})
	}
}

func WithRequiresADR(required bool) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"requires_adr": required})
	}
}

func WithLoadingType(loadingType string) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		// Pass the slice through as a parameter — pgx marshals []string to text[]
		// natively, so we never build a Postgres array literal from user input
		// (previous "{"+v+"}" form silently matched everything for v == "").
		return b.Where("loading_types @> ?", []string{loadingType})
	}
}

func WithVehicleLimit(limit int) VehicleFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Limit(uint64(limit))
	}
}

var vehicleColumns = []string{
	"id", "org_id", "registration_number", "brand", "model",
	"vehicle_type", "vehicle_subtype", "loading_types",
	"capacity", "volume", "length", "width", "height",
	"requires_adr", "has_temperature", "temp_min", "temp_max", "thermograph",
	"status", "rejection_reason", "created_at", "updated_at",
}

func scanVehicle(row pgx.Row) (*VehicleListItem, error) {
	var item VehicleListItem
	if err := row.Scan(
		&item.ID, &item.OrgID, &item.RegistrationNumber, &item.Brand, &item.Model,
		&item.VehicleType, &item.VehicleSubType, &item.LoadingTypes,
		&item.Capacity, &item.Volume, &item.Length, &item.Width, &item.Height,
		&item.RequiresADR, &item.HasTemperature, &item.TempMin, &item.TempMax, &item.Thermograph,
		&item.Status, &item.RejectionReason, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

// VehicleUpsertInput contains the full row to insert/update.
type VehicleUpsertInput struct {
	ID                 uuid.UUID
	OrgID              uuid.UUID
	RegistrationNumber string
	Brand              string
	Model              string
	VehicleType        string
	VehicleSubType     string
	LoadingTypes       []string
	Capacity           float64
	Volume             float64
	Length             float64
	Width              float64
	Height             float64
	RequiresADR        bool
	HasTemperature     bool
	TempMin            *float64
	TempMax            *float64
	Thermograph        bool
	Status             string
	RejectionReason    string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const vehicleUpsertConflict = `ON CONFLICT (id) DO UPDATE SET
    registration_number = EXCLUDED.registration_number,
    brand = EXCLUDED.brand,
    model = EXCLUDED.model,
    vehicle_type = EXCLUDED.vehicle_type,
    vehicle_subtype = EXCLUDED.vehicle_subtype,
    loading_types = EXCLUDED.loading_types,
    capacity = EXCLUDED.capacity,
    volume = EXCLUDED.volume,
    length = EXCLUDED.length,
    width = EXCLUDED.width,
    height = EXCLUDED.height,
    requires_adr = EXCLUDED.requires_adr,
    has_temperature = EXCLUDED.has_temperature,
    temp_min = EXCLUDED.temp_min,
    temp_max = EXCLUDED.temp_max,
    thermograph = EXCLUDED.thermograph,
    status = EXCLUDED.status,
    rejection_reason = EXCLUDED.rejection_reason,
    updated_at = EXCLUDED.updated_at`

func (p *VehiclesProjection) Upsert(ctx context.Context, in VehicleUpsertInput) error {
	loadingTypes := in.LoadingTypes
	if loadingTypes == nil {
		loadingTypes = []string{}
	}
	query, args, err := p.psql.
		Insert("vehicles_lookup").
		Columns(vehicleColumns...).
		Values(
			in.ID, in.OrgID, in.RegistrationNumber, nullableString(in.Brand), nullableString(in.Model),
			in.VehicleType, in.VehicleSubType, loadingTypes,
			nullableFloat(in.Capacity), nullableFloat(in.Volume), nullableFloat(in.Length), nullableFloat(in.Width), nullableFloat(in.Height),
			in.RequiresADR, in.HasTemperature, in.TempMin, in.TempMax, in.Thermograph,
			in.Status, nullableString(in.RejectionReason), in.CreatedAt, in.UpdatedAt,
		).
		Suffix(vehicleUpsertConflict).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert query: %w", err)
	}
	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert vehicle: %w", err)
	}
	return nil
}

func (p *VehiclesProjection) UpdateStatus(ctx context.Context, id uuid.UUID, status, rejectionReason string, updatedAt time.Time) error {
	builder := p.psql.Update("vehicles_lookup").
		Set("status", status).
		Set("updated_at", updatedAt).
		Where(squirrel.Eq{"id": id})
	if rejectionReason == "" {
		builder = builder.Set("rejection_reason", nil)
	} else {
		builder = builder.Set("rejection_reason", rejectionReason)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}
	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update vehicle status: %w", err)
	}
	return nil
}

func (p *VehiclesProjection) GetByID(ctx context.Context, id uuid.UUID) (*VehicleListItem, error) {
	query, args, err := p.psql.
		Select(vehicleColumns...).
		From("vehicles_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}
	item, err := scanVehicle(p.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get vehicle: %w", err)
	}
	return item, nil
}

func (p *VehiclesProjection) List(ctx context.Context, opts ...VehicleFilterOption) ([]VehicleListItem, error) {
	builder := p.psql.
		Select(vehicleColumns...).
		From("vehicles_lookup").
		OrderBy("created_at DESC", "id DESC")
	for _, opt := range opts {
		builder = opt(builder)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query vehicles: %w", err)
	}
	defer rows.Close()

	var result []VehicleListItem
	for rows.Next() {
		item, err := scanVehicle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		result = append(result, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return result, nil
}

func nullableFloat(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}
