package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/application/admin"
	"codeberg.org/udison/veziizi/backend/internal/application/organization"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
	defer publisher.Close()

	invitations := projections.NewInvitationsProjection(txManager)
	pendingOrgs := projections.NewPendingOrganizationsProjection(txManager)

	// Create services
	orgService := organization.NewService(txManager, evtStore, publisher, invitations)
	adminService := admin.NewService(txManager, evtStore, publisher, pendingOrgs)

	// Register organization
	input := organization.RegisterInput{
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
	}

	output, err := orgService.Register(ctx, input)
	if err != nil {
		log.Fatalf("failed to register organization: %v", err)
	}

	fmt.Printf("Organization created:\n")
	fmt.Printf("  ID: %s\n", output.OrganizationID)
	fmt.Printf("  Name: %s\n", *orgName)
	fmt.Printf("  Owner ID: %s\n", output.MemberID)
	fmt.Printf("  Owner Email: %s\n", *email)

	// Auto-approve if requested
	if *approve {
		// Small delay to let events propagate
		time.Sleep(500 * time.Millisecond)

		approveInput := admin.ApproveInput{
			OrganizationID: output.OrganizationID,
			AdminID:        uuid.Nil, // system approval for dev
		}
		if err := adminService.Approve(ctx, approveInput); err != nil {
			log.Fatalf("failed to approve organization: %v", err)
		}
		fmt.Printf("  Status: ACTIVE (auto-approved)\n")
	} else {
		fmt.Printf("  Status: PENDING\n")
	}

	fmt.Printf("\nYou can now login with:\n")
	fmt.Printf("  Email: %s\n", *email)
	fmt.Printf("  Password: %s\n", *password)
	fmt.Printf("\nNote: Wait for workers to process events before logging in.\n")
}
