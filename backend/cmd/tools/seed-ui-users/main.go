package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

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
		if err = publisher.Close(); err != nil {
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

	if err := txManager.InTx(ctx, func(ctx context.Context) error {
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
			return fmt.Errorf("profile 1: register org: %w", err)
		}
		fmt.Printf("  Org ID: %s\n", out1.OrganizationID)
		fmt.Printf("  Owner:    ui1.owner@test.local / test123 (ID: %s)\n", out1.MemberID)

		if err = adminService.Approve(ctx, admin.ApproveInput{
			OrganizationID: out1.OrganizationID,
			AdminID:        uuid.Nil,
		}); err != nil {
			return fmt.Errorf("profile 1: approve: %w", err)
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
			return fmt.Errorf("profile 1: add admin: %w", err)
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
			return fmt.Errorf("profile 1: add employee: %w", err)
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
			return fmt.Errorf("profile 1: add blocked employee: %w", err)
		}

		if err = orgService.BlockMember(ctx, organization.BlockMemberInput{
			OrganizationID: out1.OrganizationID,
			ActorID:        out1.MemberID,
			MemberID:       blockedID,
			Reason:         "UI test — заблокированный пользователь",
		}); err != nil {
			return fmt.Errorf("profile 1: block employee: %w", err)
		}
		fmt.Printf("  Blocked:  ui1.blocked@test.local / test123 (ID: %s) [BLOCKED]\n", blockedID)
		fmt.Println()

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
			return fmt.Errorf("profile 2: register org: %w", err)
		}
		fmt.Printf("  Org ID: %s\n", out2.OrganizationID)
		fmt.Printf("  Owner:  ui2.owner@test.local / test123\n")
		fmt.Printf("  Status: PENDING (на модерации)\n")
		fmt.Println()

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
			return fmt.Errorf("profile 3: register org: %w", err)
		}
		fmt.Printf("  Org ID: %s\n", out3.OrganizationID)

		if err = adminService.Approve(ctx, admin.ApproveInput{
			OrganizationID: out3.OrganizationID,
			AdminID:        uuid.Nil,
		}); err != nil {
			return fmt.Errorf("profile 3: approve: %w", err)
		}

		org3, err := adminService.GetOrganization(ctx, out3.OrganizationID)
		if err != nil {
			return fmt.Errorf("profile 3: load org: %w", err)
		}
		if err := org3.Suspend(uuid.Nil, "UI test — приостановленная организация"); err != nil {
			return fmt.Errorf("profile 3: suspend org: %w", err)
		}

		changes := org3.Changes()
		if err := evtStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("profile 3: save suspension: %w", err)
		}
		if err := publisher.Publish(ctx, "organization.events", changes...); err != nil {
			return fmt.Errorf("profile 3: publish suspension: %w", err)
		}
		org3.ClearChanges()

		fmt.Printf("  Owner:  ui3.owner@test.local / test123\n")
		fmt.Printf("  Status: SUSPENDED (приостановлена)\n")
		fmt.Println()

		return nil
	}); err != nil {
		log.Fatalf("seed ui-users failed (rolled back): %v", err)
	}

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
