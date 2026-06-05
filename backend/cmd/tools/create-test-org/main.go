package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/udisondev/veziizi/backend/internal/application/admin"
	"github.com/udisondev/veziizi/backend/internal/application/organization"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

func main() {
	// Flags for owner credentials
	email := flag.String("email", "owner@test.local", "Owner email")
	password := flag.String("password", "test123", "Owner password")
	name := flag.String("name", "Test Owner", "Owner name")
	orgName := flag.String("org", "Test Organization", "Organization name")
	approve := flag.Bool("approve", true, "Auto-approve organization")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Setup dependencies
	txManager := dbtx.NewTxExecutor(pool)
	evtStore := eventstore.NewPostgresStore(txManager)
	wmLogger := watermill.NewSlogLogger(nil)
	publisher, err := messaging.NewEventPublisher(pool, wmLogger)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Printf("failed to close publisher: %v", err)
		}
	}()

	invitations := projections.NewInvitationsProjection(txManager)
	members := projections.NewMembersProjection(txManager, cfg.Security.MaxFailedLoginAttempts, int(cfg.Security.AccountLockoutDuration.Minutes()))
	organizations := projections.NewOrganizationsProjection(txManager)
	pendingOrgs := projections.NewPendingOrganizationsProjection(txManager)

	// Create services
	orgService := organization.NewService(txManager, evtStore, publisher, invitations, members, organizations)
	adminService := admin.NewService(txManager, evtStore, publisher, pendingOrgs)

	var orgID, memberID uuid.UUID
	if err := txManager.InTx(ctx, func(ctx context.Context) error {
		output, err := orgService.Register(ctx, organization.RegisterInput{
			Name:          *orgName,
			INN:           "1234567890",
			LegalName:     *orgName + " LLC",
			Country:       values.CountryRU,
			Phone:         "+79001234567",
			Email:         "org@test.local",
			Address:       values.Address("Moscow, Test St, 1"),
			OwnerEmail:    *email,
			OwnerPassword: *password,
			OwnerName:     *name,
			OwnerPhone:    "+79001234567",
		})
		if err != nil {
			return fmt.Errorf("register: %w", err)
		}
		orgID = output.OrganizationID
		memberID = output.MemberID

		if *approve {
			if err := adminService.Approve(ctx, admin.ApproveInput{
				OrganizationID: orgID,
				AdminID:        uuid.Nil,
			}); err != nil {
				return fmt.Errorf("approve: %w", err)
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("failed to create test org (rolled back): %v", err)
	}

	fmt.Printf("Organization created:\n")
	fmt.Printf("  ID: %s\n", orgID)
	fmt.Printf("  Name: %s\n", *orgName)
	fmt.Printf("  Owner ID: %s\n", memberID)
	fmt.Printf("  Owner Email: %s\n", *email)
	if *approve {
		fmt.Printf("  Status: ACTIVE (auto-approved)\n")
	} else {
		fmt.Printf("  Status: PENDING\n")
	}

	fmt.Printf("\nYou can now login with:\n")
	fmt.Printf("  Email: %s\n", *email)
	fmt.Printf("  Password: %s\n", *password)
	fmt.Printf("\nNote: Wait for workers to process events before logging in.\n")
}
