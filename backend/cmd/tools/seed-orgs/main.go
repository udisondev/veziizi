package main

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"

	"github.com/udisondev/veziizi/backend/internal/application/admin"
	"github.com/udisondev/veziizi/backend/internal/application/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
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

// Имена владельцев для каждой организации
var ownerNames = []string{
	"Иванов Сергей Петрович",
	"Петрова Анна Михайловна",
	"Сидоров Дмитрий Александрович",
}

// Дополнительные участники для каждой организации (по индексу)
var additionalMembersByOrg = [][]memberData{
	// Альфа Логистика
	{
		{emailPrefix: "admin", name: "Козлова Елена Викторовна", role: values.MemberRoleAdministrator},
		{emailPrefix: "emp1", name: "Смирнов Алексей Игоревич", role: values.MemberRoleEmployee},
		{emailPrefix: "emp2", name: "Новикова Мария Андреевна", role: values.MemberRoleEmployee},
	},
	// Бета Транспорт
	{
		{emailPrefix: "admin", name: "Морозов Виктор Николаевич", role: values.MemberRoleAdministrator},
		{emailPrefix: "emp1", name: "Волкова Ольга Сергеевна", role: values.MemberRoleEmployee},
		{emailPrefix: "emp2", name: "Лебедев Артём Павлович", role: values.MemberRoleEmployee},
	},
	// Гамма Перевозки
	{
		{emailPrefix: "admin", name: "Соколова Наталья Дмитриевна", role: values.MemberRoleAdministrator},
		{emailPrefix: "emp1", name: "Кузнецов Максим Олегович", role: values.MemberRoleEmployee},
		{emailPrefix: "emp2", name: "Попова Татьяна Владимировна", role: values.MemberRoleEmployee},
	},
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
	organizations := projections.NewOrganizationsProjection(txManager)
	pendingOrgs := projections.NewPendingOrganizationsProjection(txManager)

	orgService := organization.NewService(txManager, evtStore, publisher, invitations, members, organizations)
	adminService := admin.NewService(txManager, evtStore, publisher, pendingOrgs)

	fmt.Println("Creating seed organizations...")
	fmt.Println()

	for i, org := range orgs {
		ownerEmail := fmt.Sprintf("%s.owner@mail.ru", org.prefix)
		ownerPassword := fmt.Sprintf("%s.owner12345", org.prefix)
		ownerName := ownerNames[i]

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
		for _, m := range additionalMembersByOrg[i] {
			email := fmt.Sprintf("%s.%s@mail.ru", org.prefix, m.emailPrefix)
			password := fmt.Sprintf("%s.%s12345", org.prefix, m.emailPrefix)
			name := m.name

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
