package main

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"

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

type orgData struct {
	name      string
	prefix    string
	inn       string
	legalName string
}

type memberData struct {
	emailPrefix string
	name        string
	role        values.MemberRole
}

var orgs = []orgData{
	{name: "Альфа Логистика", prefix: "alpha", inn: "1111111111", legalName: "ООО Альфа Логистика"},
	{name: "Бета Транспорт", prefix: "beta", inn: "2222222222", legalName: "ООО Бета Транспорт"},
	{name: "Гамма Перевозки", prefix: "gamma", inn: "3333333333", legalName: "ООО Гамма Перевозки"},
}

var additionalMembers = []memberData{
	{emailPrefix: "admin", name: "Администратор", role: values.MemberRoleAdministrator},
	{emailPrefix: "emp1", name: "Сотрудник 1", role: values.MemberRoleEmployee},
	{emailPrefix: "emp2", name: "Сотрудник 2", role: values.MemberRoleEmployee},
}

func main() {
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
	members := projections.NewMembersProjection(txManager)
	pendingOrgs := projections.NewPendingOrganizationsProjection(txManager)

	orgService := organization.NewService(txManager, evtStore, publisher, invitations, members)
	adminService := admin.NewService(txManager, evtStore, publisher, pendingOrgs)

	fmt.Println("Creating seed organizations...")
	fmt.Println()

	for _, org := range orgs {
		ownerEmail := fmt.Sprintf("%s.owner@mail.ru", org.prefix)
		ownerPassword := fmt.Sprintf("%s.owner12345", org.prefix)
		ownerName := fmt.Sprintf("%s Owner", org.name)

		// Register organization with owner
		input := organization.RegisterInput{
			Name:          org.name,
			INN:           org.inn,
			LegalName:     org.legalName,
			Country:       values.CountryRU,
			Phone:         "+79001234567",
			Email:         fmt.Sprintf("%s@company.ru", org.prefix),
			Address:       values.Address("Москва, ул. Тестовая, 1"),
			OwnerEmail:    ownerEmail,
			OwnerPassword: ownerPassword,
			OwnerName:     ownerName,
			OwnerPhone:    "+79001234567",
		}

		output, err := orgService.Register(ctx, input)
		if err != nil {
			log.Printf("failed to register %s: %v", org.name, err)
			continue
		}

		fmt.Printf("=== %s ===\n", org.name)
		fmt.Printf("  Org ID: %s\n", output.OrganizationID)
		fmt.Printf("  Owner: %s / %s\n", ownerEmail, ownerPassword)

		// Small delay for events
		time.Sleep(300 * time.Millisecond)

		// Approve organization
		approveInput := admin.ApproveInput{
			OrganizationID: output.OrganizationID,
			AdminID:        uuid.Nil,
		}
		if err := adminService.Approve(ctx, approveInput); err != nil {
			log.Printf("failed to approve %s: %v", org.name, err)
			continue
		}
		fmt.Printf("  Status: ACTIVE\n")

		// Add additional members
		for _, m := range additionalMembers {
			email := fmt.Sprintf("%s.%s@mail.ru", org.prefix, m.emailPrefix)
			password := fmt.Sprintf("%s.%s12345", org.prefix, m.emailPrefix)
			name := fmt.Sprintf("%s %s", org.name, m.name)

			addInput := organization.AddMemberInput{
				OrganizationID: output.OrganizationID,
				Email:          email,
				Password:       password,
				Name:           name,
				Phone:          "+79001234567",
				Role:           m.role,
			}

			memberID, err := orgService.AddMemberDirect(ctx, addInput)
			if err != nil {
				log.Printf("failed to add member %s: %v", email, err)
				continue
			}
			fmt.Printf("  %s: %s / %s (ID: %s)\n", m.role, email, password, memberID)
		}

		fmt.Println()
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println("Seed completed!")
	fmt.Println("Note: Wait for workers to process events before logging in.")
}
