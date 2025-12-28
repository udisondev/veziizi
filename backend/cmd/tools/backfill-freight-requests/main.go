package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest"
	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization"
	orgEvents "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	txManager := dbtx.NewTxExecutor(pool)
	es := eventstore.NewPostgresStore(txManager)
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Get all freight request IDs
	rows, err := pool.Query(ctx, "SELECT id FROM freight_requests_lookup")
	if err != nil {
		slog.Error("failed to query freight requests", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			slog.Error("failed to scan id", slog.String("error", err.Error()))
			os.Exit(1)
		}
		ids = append(ids, id)
	}
	rows.Close()

	slog.Info("found freight requests to backfill", slog.Int("count", len(ids)))

	for _, id := range ids {
		events, err := es.Load(ctx, id, frEvents.AggregateType)
		if err != nil {
			slog.Error("failed to load events", slog.String("id", id.String()), slog.String("error", err.Error()))
			continue
		}

		fr := freightrequest.NewFromEvents(id, events)
		if fr.Version() == 0 {
			slog.Warn("no events for freight request", slog.String("id", id.String()))
			continue
		}

		// Extract display data
		route := fr.Route()
		var originAddr, destAddr string
		if len(route.Points) > 0 {
			originAddr = route.Points[0].Address
			destAddr = route.Points[len(route.Points)-1].Address
		}

		vehicleType := fr.VehicleRequirements().VehicleType.String()
		vehicleSubType := fr.VehicleRequirements().VehicleSubType.String()

		var priceAmount *int64
		var priceCurrency *string
		if fr.Payment().Price != nil {
			priceAmount = &fr.Payment().Price.Amount
			curr := fr.Payment().Price.Currency.String()
			priceCurrency = &curr
		}

		// Load organization data
		var orgName, orgINN, orgCountry *string
		orgEvts, err := es.Load(ctx, fr.CustomerOrgID(), orgEvents.AggregateType)
		if err != nil {
			slog.Warn("failed to load organization",
				slog.String("id", id.String()),
				slog.String("org_id", fr.CustomerOrgID().String()),
				slog.String("error", err.Error()))
		} else if len(orgEvts) > 0 {
			org := organization.NewFromEvents(fr.CustomerOrgID(), orgEvts)
			name := org.Name()
			inn := org.INN()
			country := org.Country().String()
			orgName = &name
			orgINN = &inn
			orgCountry = &country
		}

		query, args, err := psql.
			Update("freight_requests_lookup").
			Set("origin_address", originAddr).
			Set("destination_address", destAddr).
			Set("cargo_weight", fr.Cargo().Weight).
			Set("price_amount", priceAmount).
			Set("price_currency", priceCurrency).
			Set("vehicle_type", vehicleType).
			Set("vehicle_subtype", vehicleSubType).
			Set("customer_org_name", orgName).
			Set("customer_org_inn", orgINN).
			Set("customer_org_country", orgCountry).
			Set("customer_member_id", fr.CustomerMemberID()).
			Where(squirrel.Eq{"id": id}).
			ToSql()
		if err != nil {
			slog.Error("failed to build query", slog.String("id", id.String()), slog.String("error", err.Error()))
			continue
		}

		if _, err := pool.Exec(ctx, query, args...); err != nil {
			slog.Error("failed to update", slog.String("id", id.String()), slog.String("error", err.Error()))
			continue
		}

		slog.Info("updated freight request",
			slog.String("id", id.String()),
			slog.String("origin", originAddr),
			slog.String("destination", destAddr),
			slog.Any("org_name", orgName),
		)
	}

	fmt.Println("Backfill completed!")
}
