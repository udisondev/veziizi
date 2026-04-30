package main

import (
	"context"
	"fmt"
	"log"
	"time"

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
	members := projections.NewMembersProjection(txManager, cfg.Security.MaxFailedLoginAttempts, int(cfg.Security.AccountLockoutDuration.Minutes()))
	organizations := projections.NewOrganizationsProjection(txManager)
	pendingOrgs := projections.NewPendingOrganizationsProjection(txManager)

	orgService := organization.NewService(txManager, evtStore, publisher, invitations, members, organizations)
	adminService := admin.NewService(txManager, evtStore, publisher, pendingOrgs)

	fmt.Println("=== Создание тестовых пользователей для UI тестирования ===")
	fmt.Println()

	// ─── Профиль 1: Активная организация ───────────────────────────────────────
	// Owner + Admin + Employee + Заблокированный сотрудник

	fmt.Println("--- Профиль 1: Активная организация (UI Test 1 Active) ---")

	out1, err := orgService.Register(ctx, organization.RegisterInput{
		Name:          "UI Test 1 Active",
		INN:           "9900000001",
		LegalName:     "ООО UI Test 1",
		Country:       values.CountryRU,
		Phone:         "+79001110001",
		Email:         "ui1@test.local",
		Address:       values.Address("Москва, ул. Тестовая, 1"),
		OwnerEmail:    "ui1.owner@test.local",
		OwnerPassword: "test123",
		OwnerName:     "Владелец Первый",
		OwnerPhone:    "+79001110001",
	})
	if err != nil {
		log.Fatalf("Profile 1: failed to register org: %v", err)
	}
	fmt.Printf("  Org ID: %s\n", out1.OrganizationID)
	fmt.Printf("  Owner:    ui1.owner@test.local / test123 (ID: %s)\n", out1.MemberID)

	time.Sleep(300 * time.Millisecond)

	if err := adminService.Approve(ctx, admin.ApproveInput{
		OrganizationID: out1.OrganizationID,
		AdminID:        uuid.Nil,
	}); err != nil {
		log.Fatalf("Profile 1: failed to approve: %v", err)
	}
	fmt.Println("  Status: ACTIVE")

	// Administrator
	adminID, err := orgService.AddMemberDirect(ctx, organization.AddMemberInput{
		OrganizationID: out1.OrganizationID,
		Email:          "ui1.admin@test.local",
		Password:       "test123",
		Name:           "Администратор Первый",
		Phone:          "+79001110002",
		Role:           values.MemberRoleAdministrator,
	})
	if err != nil {
		log.Fatalf("Profile 1: failed to add admin: %v", err)
	}
	fmt.Printf("  Admin:    ui1.admin@test.local / test123 (ID: %s)\n", adminID)

	// Employee
	empID, err := orgService.AddMemberDirect(ctx, organization.AddMemberInput{
		OrganizationID: out1.OrganizationID,
		Email:          "ui1.employee@test.local",
		Password:       "test123",
		Name:           "Сотрудник Первый",
		Phone:          "+79001110003",
		Role:           values.MemberRoleEmployee,
	})
	if err != nil {
		log.Fatalf("Profile 1: failed to add employee: %v", err)
	}
	fmt.Printf("  Employee: ui1.employee@test.local / test123 (ID: %s)\n", empID)

	// Blocked employee
	blockedID, err := orgService.AddMemberDirect(ctx, organization.AddMemberInput{
		OrganizationID: out1.OrganizationID,
		Email:          "ui1.blocked@test.local",
		Password:       "test123",
		Name:           "Заблокированный Сотрудник",
		Phone:          "+79001110004",
		Role:           values.MemberRoleEmployee,
	})
	if err != nil {
		log.Fatalf("Profile 1: failed to add blocked employee: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	if err := orgService.BlockMember(ctx, organization.BlockMemberInput{
		OrganizationID: out1.OrganizationID,
		ActorID:        out1.MemberID,
		MemberID:       blockedID,
		Reason:         "UI test — заблокированный пользователь",
	}); err != nil {
		log.Fatalf("Profile 1: failed to block employee: %v", err)
	}
	fmt.Printf("  Blocked:  ui1.blocked@test.local / test123 (ID: %s) [BLOCKED]\n", blockedID)
	fmt.Println()

	time.Sleep(300 * time.Millisecond)

	// ─── Профиль 2: Организация на модерации ───────────────────────────────────

	fmt.Println("--- Профиль 2: Организация на модерации (UI Test 2 Pending) ---")

	out2, err := orgService.Register(ctx, organization.RegisterInput{
		Name:          "UI Test 2 Pending",
		INN:           "9900000002",
		LegalName:     "ООО UI Test 2",
		Country:       values.CountryRU,
		Phone:         "+79001110010",
		Email:         "ui2@test.local",
		Address:       values.Address("Москва, ул. Тестовая, 2"),
		OwnerEmail:    "ui2.owner@test.local",
		OwnerPassword: "test123",
		OwnerName:     "Владелец Второй",
		OwnerPhone:    "+79001110010",
	})
	if err != nil {
		log.Fatalf("Profile 2: failed to register org: %v", err)
	}
	fmt.Printf("  Org ID: %s\n", out2.OrganizationID)
	fmt.Printf("  Owner:  ui2.owner@test.local / test123\n")
	fmt.Printf("  Status: PENDING (на модерации)\n")
	fmt.Println()

	time.Sleep(300 * time.Millisecond)

	// ─── Профиль 3: Приостановленная организация ───────────────────────────────

	fmt.Println("--- Профиль 3: Приостановленная организация (UI Test 3 Suspended) ---")

	out3, err := orgService.Register(ctx, organization.RegisterInput{
		Name:          "UI Test 3 Suspended",
		INN:           "9900000003",
		LegalName:     "ООО UI Test 3",
		Country:       values.CountryRU,
		Phone:         "+79001110020",
		Email:         "ui3@test.local",
		Address:       values.Address("Москва, ул. Тестовая, 3"),
		OwnerEmail:    "ui3.owner@test.local",
		OwnerPassword: "test123",
		OwnerName:     "Владелец Третий",
		OwnerPhone:    "+79001110020",
	})
	if err != nil {
		log.Fatalf("Profile 3: failed to register org: %v", err)
	}
	fmt.Printf("  Org ID: %s\n", out3.OrganizationID)

	time.Sleep(300 * time.Millisecond)

	// Approve first
	if err := adminService.Approve(ctx, admin.ApproveInput{
		OrganizationID: out3.OrganizationID,
		AdminID:        uuid.Nil,
	}); err != nil {
		log.Fatalf("Profile 3: failed to approve: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Load org and suspend
	org3, err := adminService.GetOrganization(ctx, out3.OrganizationID)
	if err != nil {
		log.Fatalf("Profile 3: failed to load org: %v", err)
	}
	if err := org3.Suspend(uuid.Nil, "UI test — приостановленная организация"); err != nil {
		log.Fatalf("Profile 3: failed to suspend org: %v", err)
	}

	changes := org3.Changes()
	if err := txManager.InTx(ctx, func(ctx context.Context) error {
		if err := evtStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		if err := publisher.Publish(ctx, "organization.events", changes...); err != nil {
			return fmt.Errorf("failed to publish: %w", err)
		}
		org3.ClearChanges()
		return nil
	}); err != nil {
		log.Fatalf("Profile 3: failed to save suspension: %v", err)
	}

	fmt.Printf("  Owner:  ui3.owner@test.local / test123\n")
	fmt.Printf("  Status: SUSPENDED (приостановлена)\n")
	fmt.Println()

	fmt.Println("=== Готово! ===")
	fmt.Println()
	fmt.Println("Подождите несколько секунд, пока воркеры обработают события, затем войдите:")
	fmt.Println()
	fmt.Println("Профиль 1 — Активная организация:")
	fmt.Println("  Владелец:          ui1.owner@test.local    / test123  (все права, видит 'Новая заявка')")
	fmt.Println("  Администратор:     ui1.admin@test.local    / test123  (может создавать заявки)")
	fmt.Println("  Сотрудник:         ui1.employee@test.local / test123  (только просмотр)")
	fmt.Println("  Заблокированный:   ui1.blocked@test.local  / test123  (не может войти)")
	fmt.Println()
	fmt.Println("Профиль 2 — Организация на модерации:")
	fmt.Println("  Владелец:          ui2.owner@test.local    / test123  (страница 'На модерации')")
	fmt.Println()
	fmt.Println("Профиль 3 — Приостановленная организация:")
	fmt.Println("  Владелец:          ui3.owner@test.local    / test123  (страница 'Приостановлена')")
}
