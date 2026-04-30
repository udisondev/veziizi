package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"

	frApp "github.com/udisondev/veziizi/backend/internal/application/freightrequest"
	orgApp "github.com/udisondev/veziizi/backend/internal/application/organization"
	adminApp "github.com/udisondev/veziizi/backend/internal/application/admin"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/adapters"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/sequence"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
	"github.com/google/uuid"
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

	orgService := orgApp.NewService(txManager, evtStore, publisher, invitations, members, organizations)
	adminService := adminApp.NewService(txManager, evtStore, publisher, pendingOrgs)
	seqGen := sequence.NewGenerator(txManager)
	memberChecker := adapters.NewMemberCheckerAdapter(orgService)
	frService := frApp.NewService(txManager, evtStore, publisher, seqGen, memberChecker)

	_ = adminService

	fmt.Println("=== Создание сценарных данных ===")
	fmt.Println()

	// ─── Загружаем тестовых пользователей ──────────────────────────────────────

	ui1Owner, err := members.GetByEmail(ctx, "ui1.owner@test.local")
	if err != nil {
		log.Fatalf("ui1.owner не найден (запустите seed:ui-users сначала): %v", err)
	}
	alphaOwner, err := members.GetByEmail(ctx, "alpha.owner@mail.ru")
	if err != nil {
		log.Fatalf("alpha.owner не найден (запустите seed:orgs сначала): %v", err)
	}
	betaOwner, err := members.GetByEmail(ctx, "beta.owner@mail.ru")
	if err != nil {
		log.Fatalf("beta.owner не найден (запустите seed:orgs сначала): %v", err)
	}
	gammaOwner, err := members.GetByEmail(ctx, "gamma.owner@mail.ru")
	if err != nil {
		log.Fatalf("gamma.owner не найден (запустите seed:orgs сначала): %v", err)
	}

	ui1OrgID := ui1Owner.OrganizationID
	alphaOrgID := alphaOwner.OrganizationID
	betaOrgID := betaOwner.OrganizationID
	gammaOrgID := gammaOwner.OrganizationID

	fmt.Printf("ui1 org:    %s\n", ui1OrgID)
	fmt.Printf("alpha org:  %s\n", alphaOrgID)
	fmt.Printf("beta org:   %s\n", betaOrgID)
	fmt.Printf("gamma org:  %s\n", gammaOrgID)
	fmt.Println()

	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	dayAfter := time.Now().UTC().AddDate(0, 0, 3).Format("2006-01-02")

	// ─── Сценарий 1: ui1 как заказчик, 3 завершённые сделки с отзывами ────────
	// alpha, beta, gamma — перевозчики поочерёдно

	fmt.Println("--- Сценарий 1: завершённые сделки (ui1 → alpha, beta, gamma) ---")

	type deal struct {
		carrierOrgID    uuid.UUID
		carrierMemberID uuid.UUID
		carrierName     string
		rating          int
		comment         string
	}

	completedDeals := []deal{
		{alphaOrgID, alphaOwner.ID, "Альфа Логистика", 5, "Отличная работа, всё в срок!"},
		{betaOrgID, betaOwner.ID, "Бета Транспорт", 4, "Хорошо, небольшая задержка"},
		{gammaOrgID, gammaOwner.ID, "Гамма Перевозки", 5, "Профессионально, рекомендую"},
	}

	routes := []struct {
		from, to string
	}{
		{"Москва, склад А", "Санкт-Петербург, склад Б"},
		{"Казань, ТЦ Мега", "Нижний Новгород, склад В"},
		{"Екатеринбург, завод", "Челябинск, терминал"},
	}

	for i, d := range completedDeals {
		route, err := values.NewRoute([]values.RoutePoint{
			{
				IsLoading: true,
				Address:   routes[i].from,
				DateFrom:  tomorrow,
			},
			{
				IsUnloading: true,
				Address:     routes[i].to,
				DateFrom:    dayAfter,
			},
		})
		if err != nil {
			log.Fatalf("failed to build route %d: %v", i, err)
		}

		frID, err := frService.Create(ctx, frApp.CreateInput{
			CustomerOrgID:    ui1OrgID,
			CustomerMemberID: ui1Owner.ID,
			Route:            route,
			Cargo: values.CargoInfo{
				Description: fmt.Sprintf("Груз %d — паллеты", i+1),
				Weight:      float64(5000 + i*2000),
				Quantity:    10 + i,
			},
			VehicleRequirements: values.VehicleRequirements{
				VehicleType:    values.VehicleTypeHeavyTruck,
				VehicleSubType: values.VehicleSubTypeBoxTruck,
			},
			Payment: values.Payment{
				Price:   &values.Money{Amount: int64(150000+i*50000) * 100, Currency: "RUB"},
				VatType: values.VatTypeNone,
				Method:  values.PaymentMethodBankTransfer,
				Terms:   values.PaymentTermsOnUnloading,
			},
		})
		if err != nil {
			log.Fatalf("deal %d: failed to create: %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)

		offerID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
			FreightRequestID: frID,
			CarrierOrgID:     d.carrierOrgID,
			CarrierMemberID:  d.carrierMemberID,
			Price:            values.Money{Amount: int64(140000+i*50000) * 100, Currency: "RUB"},
			VatType:          values.VatTypeNone,
			PaymentMethod:    values.PaymentMethodBankTransfer,
		})
		if err != nil {
			log.Fatalf("deal %d: failed to make offer: %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)

		if err := frService.SelectOffer(ctx, frApp.SelectOfferInput{
			FreightRequestID: frID,
			OfferID:          offerID,
			ActorID:          ui1Owner.ID,
			ActorOrgID:       ui1OrgID,
		}); err != nil {
			log.Fatalf("deal %d: failed to select offer: %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)

		if err := frService.ConfirmOffer(ctx, frApp.ConfirmOfferInput{
			FreightRequestID: frID,
			OfferID:          offerID,
			ActorMemberID:    d.carrierMemberID,
			ActorOrgID:       d.carrierOrgID,
			ActorRole:        orgValues.MemberRoleOwner,
		}); err != nil {
			log.Fatalf("deal %d: failed to confirm offer: %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)

		// Завершают обе стороны
		if err := frService.Complete(ctx, frApp.CompleteInput{
			FreightRequestID: frID,
			OrgID:            d.carrierOrgID,
			MemberID:         d.carrierMemberID,
		}); err != nil {
			log.Fatalf("deal %d: failed to complete (carrier): %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)

		if err := frService.Complete(ctx, frApp.CompleteInput{
			FreightRequestID: frID,
			OrgID:            ui1OrgID,
			MemberID:         ui1Owner.ID,
		}); err != nil {
			log.Fatalf("deal %d: failed to complete (customer): %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)

		// ui1 оставляет отзыв перевозчику
		if _, err := frService.LeaveReview(ctx, frApp.LeaveReviewInput{
			FreightRequestID: frID,
			ReviewerOrgID:    ui1OrgID,
			ReviewerMemberID: ui1Owner.ID,
			Rating:           d.rating,
			Comment:          d.comment,
		}); err != nil {
			log.Fatalf("deal %d: failed to leave review by customer: %v", i, err)
		}
		time.Sleep(200 * time.Millisecond)

		// Перевозчик оставляет отзыв ui1
		if _, err := frService.LeaveReview(ctx, frApp.LeaveReviewInput{
			FreightRequestID: frID,
			ReviewerOrgID:    d.carrierOrgID,
			ReviewerMemberID: d.carrierMemberID,
			Rating:           5,
			Comment:          "Хороший заказчик, всё чётко",
		}); err != nil {
			log.Fatalf("deal %d: failed to leave review by carrier: %v", i, err)
		}

		fmt.Printf("  ✓ Сделка с %s — завершена, отзывы оставлены\n", d.carrierName)
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println()

	// ─── Сценарий 2: активные заявки ui1 в разных статусах ───────────────────

	fmt.Println("--- Сценарий 2: активные заявки ui1 (published, selected, confirmed) ---")

	// 2а) Просто опубликована — ждёт перевозчика
	frPublished, err := frService.Create(ctx, frApp.CreateInput{
		CustomerOrgID:    ui1OrgID,
		CustomerMemberID: ui1Owner.ID,
		Route: mustRoute(values.RoutePoint{IsLoading: true, Address: "Москва, ул. Склад, 5", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Воронеж, ул. Терминал, 1", DateFrom: dayAfter}),
		Cargo: values.CargoInfo{Description: "Оборудование", Weight: 3000, Quantity: 5},
		VehicleRequirements: values.VehicleRequirements{
			VehicleType:    values.VehicleTypeMediumTruck,
			VehicleSubType: values.VehicleSubTypeBoxTruck,
		},
		Payment: values.Payment{
			VatType: values.VatTypeNone,
			Method:  values.PaymentMethodBankTransfer,
			Terms:   values.PaymentTermsOnUnloading,
		},
	})
	if err != nil {
		log.Fatalf("published: failed to create: %v", err)
	}
	fmt.Printf("  ✓ Опубликована (ждёт перевозчика): %s\n", frPublished)
	time.Sleep(200 * time.Millisecond)

	// Три pending-оффера от разных перевозчиков — видны в фиде дашборда
	pendingOffers := []struct {
		orgID    uuid.UUID
		memberID uuid.UUID
		name     string
		price    int64
	}{
		{alphaOrgID, alphaOwner.ID, "Альфа Логистика", 8500000},
		{betaOrgID, betaOwner.ID, "Бета Транспорт", 9200000},
		{gammaOrgID, gammaOwner.ID, "Гамма Перевозки", 7800000},
	}
	for _, po := range pendingOffers {
		_, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
			FreightRequestID: frPublished,
			CarrierOrgID:     po.orgID,
			CarrierMemberID:  po.memberID,
			Price:            values.Money{Amount: po.price, Currency: "RUB"},
			VatType:          values.VatTypeNone,
			PaymentMethod:    values.PaymentMethodBankTransfer,
		})
		if err != nil {
			log.Fatalf("pending offer from %s: %v", po.name, err)
		}
		fmt.Printf("    → оффер от %s (%.0f₽)\n", po.name, float64(po.price)/100)
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)

	// 2б) Выбран перевозчик — ждёт подтверждения
	frSelected, err := frService.Create(ctx, frApp.CreateInput{
		CustomerOrgID:    ui1OrgID,
		CustomerMemberID: ui1Owner.ID,
		Route: mustRoute(values.RoutePoint{IsLoading: true, Address: "Тула, завод", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Рязань, склад", DateFrom: dayAfter}),
		Cargo: values.CargoInfo{Description: "Металлопрокат", Weight: 10000, Quantity: 1},
		VehicleRequirements: values.VehicleRequirements{
			VehicleType:    values.VehicleTypeHeavyTruck,
			VehicleSubType: values.VehicleSubTypeBoxTruck,
		},
		Payment: values.Payment{
			Price:   &values.Money{Amount: 8000000, Currency: "RUB"},
			VatType: values.VatTypeNone,
			Method:  values.PaymentMethodBankTransfer,
			Terms:   values.PaymentTermsOnUnloading,
		},
	})
	if err != nil {
		log.Fatalf("selected: failed to create: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	offerSelectedID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
		FreightRequestID: frSelected,
		CarrierOrgID:     alphaOrgID,
		CarrierMemberID:  alphaOwner.ID,
		Price:            values.Money{Amount: 7500000, Currency: "RUB"},
		VatType:          values.VatTypeNone,
		PaymentMethod:    values.PaymentMethodBankTransfer,
	})
	if err != nil {
		log.Fatalf("selected: failed to make offer: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	if err := frService.SelectOffer(ctx, frApp.SelectOfferInput{
		FreightRequestID: frSelected,
		OfferID:          offerSelectedID,
		ActorID:          ui1Owner.ID,
		ActorOrgID:       ui1OrgID,
	}); err != nil {
		log.Fatalf("selected: failed to select offer: %v", err)
	}
	fmt.Printf("  ✓ Перевозчик выбран (ждёт подтверждения): %s\n", frSelected)
	time.Sleep(300 * time.Millisecond)

	// 2в) Подтверждена — в пути
	frConfirmed, err := frService.Create(ctx, frApp.CreateInput{
		CustomerOrgID:    ui1OrgID,
		CustomerMemberID: ui1Owner.ID,
		Route: mustRoute(values.RoutePoint{IsLoading: true, Address: "Самара, порт", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Уфа, склад", DateFrom: dayAfter}),
		Cargo: values.CargoInfo{Description: "Продукты питания", Weight: 7000, Quantity: 20},
		VehicleRequirements: values.VehicleRequirements{
			VehicleType:    values.VehicleTypeHeavyTruck,
			VehicleSubType: values.VehicleSubTypeBoxTruck,
		},
		Payment: values.Payment{
			Price:   &values.Money{Amount: 12000000, Currency: "RUB"},
			VatType: values.VatTypeNone,
			Method:  values.PaymentMethodBankTransfer,
			Terms:   values.PaymentTermsOnUnloading,
		},
	})
	if err != nil {
		log.Fatalf("confirmed: failed to create: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	offerConfirmedID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
		FreightRequestID: frConfirmed,
		CarrierOrgID:     betaOrgID,
		CarrierMemberID:  betaOwner.ID,
		Price:            values.Money{Amount: 11000000, Currency: "RUB"},
		VatType:          values.VatTypeNone,
		PaymentMethod:    values.PaymentMethodBankTransfer,
	})
	if err != nil {
		log.Fatalf("confirmed: failed to make offer: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	if err := frService.SelectOffer(ctx, frApp.SelectOfferInput{
		FreightRequestID: frConfirmed,
		OfferID:          offerConfirmedID,
		ActorID:          ui1Owner.ID,
		ActorOrgID:       ui1OrgID,
	}); err != nil {
		log.Fatalf("confirmed: failed to select offer: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	if err := frService.ConfirmOffer(ctx, frApp.ConfirmOfferInput{
		FreightRequestID: frConfirmed,
		OfferID:          offerConfirmedID,
		ActorMemberID:    betaOwner.ID,
		ActorOrgID:       betaOrgID,
		ActorRole:        orgValues.MemberRoleOwner,
	}); err != nil {
		log.Fatalf("confirmed: failed to confirm offer: %v", err)
	}
	fmt.Printf("  ✓ В пути (подтверждена): %s\n", frConfirmed)
	time.Sleep(300 * time.Millisecond)

	// 2г) Ещё одна опубликованная заявка — 2 оффера
	frPublished2, err := frService.Create(ctx, frApp.CreateInput{
		CustomerOrgID:    ui1OrgID,
		CustomerMemberID: ui1Owner.ID,
		Route: mustRoute(values.RoutePoint{IsLoading: true, Address: "Краснодар, порт", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Ростов-на-Дону, склад", DateFrom: dayAfter}),
		Cargo: values.CargoInfo{Description: "Строительные материалы", Weight: 8000, Quantity: 15},
		VehicleRequirements: values.VehicleRequirements{
			VehicleType:    values.VehicleTypeHeavyTruck,
			VehicleSubType: values.VehicleSubTypeBoxTruck,
		},
		Payment: values.Payment{
			Price:   &values.Money{Amount: 5500000, Currency: "RUB"},
			VatType: values.VatTypeNone,
			Method:  values.PaymentMethodBankTransfer,
			Terms:   values.PaymentTermsOnUnloading,
		},
	})
	if err != nil {
		log.Fatalf("published2: failed to create: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	for _, po := range []struct {
		orgID    uuid.UUID
		memberID uuid.UUID
		name     string
		price    int64
	}{
		{betaOrgID, betaOwner.ID, "Бета Транспорт", 5200000},
		{gammaOrgID, gammaOwner.ID, "Гамма Перевозки", 5000000},
	} {
		_, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
			FreightRequestID: frPublished2,
			CarrierOrgID:     po.orgID,
			CarrierMemberID:  po.memberID,
			Price:            values.Money{Amount: po.price, Currency: "RUB"},
			VatType:          values.VatTypeNone,
			PaymentMethod:    values.PaymentMethodBankTransfer,
		})
		if err != nil {
			log.Fatalf("pending offer2 from %s: %v", po.name, err)
		}
		fmt.Printf("    → оффер от %s (%.0f₽)\n", po.name, float64(po.price)/100)
		time.Sleep(150 * time.Millisecond)
	}
	fmt.Printf("  ✓ Вторая опубликованная заявка с 2 офферами: %s\n", frPublished2)
	fmt.Println()

	// ─── Сценарий 3: alpha как заказчик → ui1 как перевозчик ──────────────────

	fmt.Println()
	fmt.Println("--- Сценарий 3: ui1 как перевозчик (alpha создаёт, ui1 везёт) ---")

	frAlphaCustomer, err := frService.Create(ctx, frApp.CreateInput{
		CustomerOrgID:    alphaOrgID,
		CustomerMemberID: alphaOwner.ID,
		Route: mustRoute(values.RoutePoint{IsLoading: true, Address: "Пермь, завод", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Тюмень, склад", DateFrom: dayAfter}),
		Cargo: values.CargoInfo{Description: "Химическое сырьё", Weight: 15000, Quantity: 3},
		VehicleRequirements: values.VehicleRequirements{
			VehicleType:    values.VehicleTypeHeavyTruck,
			VehicleSubType: values.VehicleSubTypeBoxTruck,
		},
		Payment: values.Payment{
			Price:   &values.Money{Amount: 25000000, Currency: "RUB"},
			VatType: values.VatTypeNone,
			Method:  values.PaymentMethodBankTransfer,
			Terms:   values.PaymentTermsOnUnloading,
		},
	})
	if err != nil {
		log.Fatalf("carrier scenario: failed to create: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	offerUI1ID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
		FreightRequestID: frAlphaCustomer,
		CarrierOrgID:     ui1OrgID,
		CarrierMemberID:  ui1Owner.ID,
		Price:            values.Money{Amount: 24000000, Currency: "RUB"},
		VatType:          values.VatTypeNone,
		PaymentMethod:    values.PaymentMethodBankTransfer,
	})
	if err != nil {
		log.Fatalf("carrier scenario: failed to make offer: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	if err := frService.SelectOffer(ctx, frApp.SelectOfferInput{
		FreightRequestID: frAlphaCustomer,
		OfferID:          offerUI1ID,
		ActorID:          alphaOwner.ID,
		ActorOrgID:       alphaOrgID,
	}); err != nil {
		log.Fatalf("carrier scenario: failed to select offer: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	if err := frService.ConfirmOffer(ctx, frApp.ConfirmOfferInput{
		FreightRequestID: frAlphaCustomer,
		OfferID:          offerUI1ID,
		ActorMemberID:    ui1Owner.ID,
		ActorOrgID:       ui1OrgID,
		ActorRole:        orgValues.MemberRoleOwner,
	}); err != nil {
		log.Fatalf("carrier scenario: failed to confirm offer: %v", err)
	}
	fmt.Printf("  ✓ ui1 везёт для alpha — заявка в пути: %s\n", frAlphaCustomer)

	// ─── Сценарий 4: заявки, которые скоро истекают ──────────────────────────

	fmt.Println()
	fmt.Println("--- Сценарий 4: заявки, истекающие через 1 и 2 дня ---")

	expiresIn1Day := time.Now().UTC().Add(22 * time.Hour)
	expiresIn2Days := time.Now().UTC().Add(46 * time.Hour)

	for i, exp := range []struct {
		expiry  time.Time
		from    string
		to      string
		comment string
	}{
		{expiresIn1Day, "Новосибирск, склад", "Омск, терминал", "истекает через ~1 день"},
		{expiresIn2Days, "Красноярск, порт", "Иркутск, склад", "истекает через ~2 дня"},
	} {
		exp := exp
		frID, err := frService.Create(ctx, frApp.CreateInput{
			CustomerOrgID:    ui1OrgID,
			CustomerMemberID: ui1Owner.ID,
			ExpiresAt:        &exp.expiry,
			Route: mustRoute(
				values.RoutePoint{IsLoading: true, Address: exp.from, DateFrom: tomorrow},
				values.RoutePoint{IsUnloading: true, Address: exp.to, DateFrom: dayAfter},
			),
			Cargo: values.CargoInfo{Description: fmt.Sprintf("Груз %d", i+10), Weight: 4000, Quantity: 8},
			VehicleRequirements: values.VehicleRequirements{
				VehicleType:    values.VehicleTypeMediumTruck,
				VehicleSubType: values.VehicleSubTypeBoxTruck,
			},
			Payment: values.Payment{
				Price:   &values.Money{Amount: 6000000, Currency: "RUB"},
				VatType: values.VatTypeNone,
				Method:  values.PaymentMethodBankTransfer,
				Terms:   values.PaymentTermsOnUnloading,
			},
		})
		if err != nil {
			log.Fatalf("expiring soon %d: %v", i, err)
		}
		fmt.Printf("  ✓ %s → %s (%s): %s\n", exp.from, exp.to, exp.comment, frID)
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println()
	fmt.Println("=== Готово! ===")
	fmt.Println()
	fmt.Println("Подождите ~10 секунд, пока воркеры обработают события.")
	fmt.Println()
	fmt.Println("Дашборд ui1.owner@test.local должен показывать:")
	fmt.Println("  Мои активные заявки:     4 ждут перевозчика · 1 перевозчик выбран · 1 в пути")
	fmt.Println("  Офферы на мои заявки:    5 (от alpha, beta, gamma на 2 заявки)")
	fmt.Println("  Истекают скоро:          2 (через 1 и 2 дня)")
	fmt.Println("  Я перевозчик:            1 везу сейчас")
	fmt.Println("  Завершённых:             3 сделки")
	fmt.Println("  Рейтинг:                 ~4.7 (3 отзыва)")
}

func mustRoute(loading, unloading values.RoutePoint) values.Route {
	r, err := values.NewRoute([]values.RoutePoint{loading, unloading})
	if err != nil {
		log.Fatalf("failed to build route: %v", err)
	}
	return r
}
