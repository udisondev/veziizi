package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"

	"github.com/google/uuid"
	frApp "github.com/udisondev/veziizi/backend/internal/application/freightrequest"
	orgApp "github.com/udisondev/veziizi/backend/internal/application/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/adapters"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/sequence"
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
	members := projections.NewMembersProjection(
		txManager,
		cfg.Security.MaxFailedLoginAttempts,
		int(cfg.Security.AccountLockoutDuration.Minutes()),
	)
	organizations := projections.NewOrganizationsProjection(txManager)

	orgService := orgApp.NewService(txManager, evtStore, publisher, invitations, members, organizations)
	seqGen := sequence.NewGenerator(txManager)
	memberChecker := adapters.NewMemberCheckerAdapter(orgService)
	frService := frApp.NewService(txManager, evtStore, publisher, seqGen, memberChecker)

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

	if err := txManager.InTx(ctx, func(ctx context.Context) error {
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
				return fmt.Errorf("build route %d: %w", i, err)
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
				return fmt.Errorf("deal %d: create: %w", i, err)
			}

			offerID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
				FreightRequestID: frID,
				CarrierOrgID:     d.carrierOrgID,
				CarrierMemberID:  d.carrierMemberID,
				Price:            values.Money{Amount: int64(140000+i*50000) * 100, Currency: "RUB"},
				VatType:          values.VatTypeNone,
				PaymentMethod:    values.PaymentMethodBankTransfer,
			})
			if err != nil {
				return fmt.Errorf("deal %d: make offer: %w", i, err)
			}

			if err := frService.SelectOffer(ctx, frApp.SelectOfferInput{
				FreightRequestID: frID,
				OfferID:          offerID,
				ActorID:          ui1Owner.ID,
				ActorOrgID:       ui1OrgID,
			}); err != nil {
				return fmt.Errorf("deal %d: select offer: %w", i, err)
			}

			if err := frService.ConfirmOffer(ctx, frApp.ConfirmOfferInput{
				FreightRequestID: frID,
				OfferID:          offerID,
				ActorMemberID:    d.carrierMemberID,
				ActorOrgID:       d.carrierOrgID,
				ActorRole:        orgValues.MemberRoleOwner,
			}); err != nil {
				return fmt.Errorf("deal %d: confirm offer: %w", i, err)
			}

			// Завершают обе стороны
			if err := frService.Complete(ctx, frApp.CompleteInput{
				FreightRequestID: frID,
				OrgID:            d.carrierOrgID,
				MemberID:         d.carrierMemberID,
			}); err != nil {
				return fmt.Errorf("deal %d: complete (carrier): %w", i, err)
			}

			if err := frService.Complete(ctx, frApp.CompleteInput{
				FreightRequestID: frID,
				OrgID:            ui1OrgID,
				MemberID:         ui1Owner.ID,
			}); err != nil {
				return fmt.Errorf("deal %d: complete (customer): %w", i, err)
			}

			// ui1 оставляет отзыв перевозчику
			if _, err := frService.LeaveReview(ctx, frApp.LeaveReviewInput{
				FreightRequestID: frID,
				ReviewerOrgID:    ui1OrgID,
				ReviewerMemberID: ui1Owner.ID,
				Rating:           d.rating,
				Comment:          d.comment,
			}); err != nil {
				return fmt.Errorf("deal %d: leave review by customer: %w", i, err)
			}

			// Перевозчик оставляет отзыв ui1
			if _, err := frService.LeaveReview(ctx, frApp.LeaveReviewInput{
				FreightRequestID: frID,
				ReviewerOrgID:    d.carrierOrgID,
				ReviewerMemberID: d.carrierMemberID,
				Rating:           5,
				Comment:          "Хороший заказчик, всё чётко",
			}); err != nil {
				return fmt.Errorf("deal %d: leave review by carrier: %w", i, err)
			}

			fmt.Printf("  ✓ Сделка с %s — завершена, отзывы оставлены\n", d.carrierName)
		}
		fmt.Println()

		// ─── Сценарий 2: активные заявки ui1 в разных статусах ───────────────────

		fmt.Println("--- Сценарий 2: активные заявки ui1 (published, selected, confirmed) ---")

		// 2а) Просто опубликована — ждёт перевозчика
		routePublished, err := buildRoute(
			values.RoutePoint{IsLoading: true, Address: "Москва, ул. Склад, 5", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Воронеж, ул. Терминал, 1", DateFrom: dayAfter},
		)
		if err != nil {
			return fmt.Errorf("published: route: %w", err)
		}
		frPublished, err := frService.Create(ctx, frApp.CreateInput{
			CustomerOrgID:    ui1OrgID,
			CustomerMemberID: ui1Owner.ID,
			Route:            routePublished,
			Cargo:            values.CargoInfo{Description: "Оборудование", Weight: 3000, Quantity: 5},
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
			return fmt.Errorf("published: create: %w", err)
		}
		fmt.Printf("  ✓ Опубликована (ждёт перевозчика): %s\n", frPublished)

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
			_, err = frService.MakeOffer(ctx, frApp.MakeOfferInput{
				FreightRequestID: frPublished,
				CarrierOrgID:     po.orgID,
				CarrierMemberID:  po.memberID,
				Price:            values.Money{Amount: po.price, Currency: "RUB"},
				VatType:          values.VatTypeNone,
				PaymentMethod:    values.PaymentMethodBankTransfer,
			})
			if err != nil {
				return fmt.Errorf("pending offer from %s: %w", po.name, err)
			}
			fmt.Printf("    → оффер от %s (%.0f₽)\n", po.name, float64(po.price)/100)
		}

		// 2б) Выбран перевозчик — ждёт подтверждения
		routeSelected, err := buildRoute(
			values.RoutePoint{IsLoading: true, Address: "Тула, завод", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Рязань, склад", DateFrom: dayAfter},
		)
		if err != nil {
			return fmt.Errorf("selected: route: %w", err)
		}
		frSelected, err := frService.Create(ctx, frApp.CreateInput{
			CustomerOrgID:    ui1OrgID,
			CustomerMemberID: ui1Owner.ID,
			Route:            routeSelected,
			Cargo:            values.CargoInfo{Description: "Металлопрокат", Weight: 10000, Quantity: 1},
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
			return fmt.Errorf("selected: create: %w", err)
		}

		offerSelectedID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
			FreightRequestID: frSelected,
			CarrierOrgID:     alphaOrgID,
			CarrierMemberID:  alphaOwner.ID,
			Price:            values.Money{Amount: 7500000, Currency: "RUB"},
			VatType:          values.VatTypeNone,
			PaymentMethod:    values.PaymentMethodBankTransfer,
		})
		if err != nil {
			return fmt.Errorf("selected: make offer: %w", err)
		}

		if err = frService.SelectOffer(ctx, frApp.SelectOfferInput{
			FreightRequestID: frSelected,
			OfferID:          offerSelectedID,
			ActorID:          ui1Owner.ID,
			ActorOrgID:       ui1OrgID,
		}); err != nil {
			return fmt.Errorf("selected: select offer: %w", err)
		}
		fmt.Printf("  ✓ Перевозчик выбран (ждёт подтверждения): %s\n", frSelected)

		// 2в) Подтверждена — в пути
		routeConfirmed, err := buildRoute(
			values.RoutePoint{IsLoading: true, Address: "Самара, порт", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Уфа, склад", DateFrom: dayAfter},
		)
		if err != nil {
			return fmt.Errorf("confirmed: route: %w", err)
		}
		frConfirmed, err := frService.Create(ctx, frApp.CreateInput{
			CustomerOrgID:    ui1OrgID,
			CustomerMemberID: ui1Owner.ID,
			Route:            routeConfirmed,
			Cargo:            values.CargoInfo{Description: "Продукты питания", Weight: 7000, Quantity: 20},
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
			return fmt.Errorf("confirmed: create: %w", err)
		}

		offerConfirmedID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
			FreightRequestID: frConfirmed,
			CarrierOrgID:     betaOrgID,
			CarrierMemberID:  betaOwner.ID,
			Price:            values.Money{Amount: 11000000, Currency: "RUB"},
			VatType:          values.VatTypeNone,
			PaymentMethod:    values.PaymentMethodBankTransfer,
		})
		if err != nil {
			return fmt.Errorf("confirmed: make offer: %w", err)
		}

		if err = frService.SelectOffer(ctx, frApp.SelectOfferInput{
			FreightRequestID: frConfirmed,
			OfferID:          offerConfirmedID,
			ActorID:          ui1Owner.ID,
			ActorOrgID:       ui1OrgID,
		}); err != nil {
			return fmt.Errorf("confirmed: select offer: %w", err)
		}

		if err = frService.ConfirmOffer(ctx, frApp.ConfirmOfferInput{
			FreightRequestID: frConfirmed,
			OfferID:          offerConfirmedID,
			ActorMemberID:    betaOwner.ID,
			ActorOrgID:       betaOrgID,
			ActorRole:        orgValues.MemberRoleOwner,
		}); err != nil {
			return fmt.Errorf("confirmed: confirm offer: %w", err)
		}
		fmt.Printf("  ✓ В пути (подтверждена): %s\n", frConfirmed)

		// 2г) Ещё одна опубликованная заявка — 2 оффера
		routePublished2, err := buildRoute(
			values.RoutePoint{IsLoading: true, Address: "Краснодар, порт", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Ростов-на-Дону, склад", DateFrom: dayAfter},
		)
		if err != nil {
			return fmt.Errorf("published2: route: %w", err)
		}
		frPublished2, err := frService.Create(ctx, frApp.CreateInput{
			CustomerOrgID:    ui1OrgID,
			CustomerMemberID: ui1Owner.ID,
			Route:            routePublished2,
			Cargo:            values.CargoInfo{Description: "Строительные материалы", Weight: 8000, Quantity: 15},
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
			return fmt.Errorf("published2: create: %w", err)
		}

		for _, po := range []struct {
			orgID    uuid.UUID
			memberID uuid.UUID
			name     string
			price    int64
		}{
			{betaOrgID, betaOwner.ID, "Бета Транспорт", 5200000},
			{gammaOrgID, gammaOwner.ID, "Гамма Перевозки", 5000000},
		} {
			_, err = frService.MakeOffer(ctx, frApp.MakeOfferInput{
				FreightRequestID: frPublished2,
				CarrierOrgID:     po.orgID,
				CarrierMemberID:  po.memberID,
				Price:            values.Money{Amount: po.price, Currency: "RUB"},
				VatType:          values.VatTypeNone,
				PaymentMethod:    values.PaymentMethodBankTransfer,
			})
			if err != nil {
				return fmt.Errorf("pending offer2 from %s: %w", po.name, err)
			}
			fmt.Printf("    → оффер от %s (%.0f₽)\n", po.name, float64(po.price)/100)
		}
		fmt.Printf("  ✓ Вторая опубликованная заявка с 2 офферами: %s\n", frPublished2)
		fmt.Println()

		// ─── Сценарий 3: alpha как заказчик → ui1 как перевозчик ──────────────────

		fmt.Println()
		fmt.Println("--- Сценарий 3: ui1 как перевозчик (alpha создаёт, ui1 везёт) ---")

		routeAlpha, err := buildRoute(
			values.RoutePoint{IsLoading: true, Address: "Пермь, завод", DateFrom: tomorrow},
			values.RoutePoint{IsUnloading: true, Address: "Тюмень, склад", DateFrom: dayAfter},
		)
		if err != nil {
			return fmt.Errorf("carrier scenario: route: %w", err)
		}
		frAlphaCustomer, err := frService.Create(ctx, frApp.CreateInput{
			CustomerOrgID:    alphaOrgID,
			CustomerMemberID: alphaOwner.ID,
			Route:            routeAlpha,
			Cargo:            values.CargoInfo{Description: "Химическое сырьё", Weight: 15000, Quantity: 3},
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
			return fmt.Errorf("carrier scenario: create: %w", err)
		}

		offerUI1ID, err := frService.MakeOffer(ctx, frApp.MakeOfferInput{
			FreightRequestID: frAlphaCustomer,
			CarrierOrgID:     ui1OrgID,
			CarrierMemberID:  ui1Owner.ID,
			Price:            values.Money{Amount: 24000000, Currency: "RUB"},
			VatType:          values.VatTypeNone,
			PaymentMethod:    values.PaymentMethodBankTransfer,
		})
		if err != nil {
			return fmt.Errorf("carrier scenario: make offer: %w", err)
		}

		if err := frService.SelectOffer(ctx, frApp.SelectOfferInput{
			FreightRequestID: frAlphaCustomer,
			OfferID:          offerUI1ID,
			ActorID:          alphaOwner.ID,
			ActorOrgID:       alphaOrgID,
		}); err != nil {
			return fmt.Errorf("carrier scenario: select offer: %w", err)
		}

		if err := frService.ConfirmOffer(ctx, frApp.ConfirmOfferInput{
			FreightRequestID: frAlphaCustomer,
			OfferID:          offerUI1ID,
			ActorMemberID:    ui1Owner.ID,
			ActorOrgID:       ui1OrgID,
			ActorRole:        orgValues.MemberRoleOwner,
		}); err != nil {
			return fmt.Errorf("carrier scenario: confirm offer: %w", err)
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
			expRoute, err := buildRoute(
				values.RoutePoint{IsLoading: true, Address: exp.from, DateFrom: tomorrow},
				values.RoutePoint{IsUnloading: true, Address: exp.to, DateFrom: dayAfter},
			)
			if err != nil {
				return fmt.Errorf("expiring soon %d: route: %w", i, err)
			}
			frID, err := frService.Create(ctx, frApp.CreateInput{
				CustomerOrgID:    ui1OrgID,
				CustomerMemberID: ui1Owner.ID,
				ExpiresAt:        &exp.expiry,
				Route:            expRoute,
				Cargo:            values.CargoInfo{Description: fmt.Sprintf("Груз %d", i+10), Weight: 4000, Quantity: 8},
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
				return fmt.Errorf("expiring soon %d: create: %w", i, err)
			}
			fmt.Printf("  ✓ %s → %s (%s): %s\n", exp.from, exp.to, exp.comment, frID)
		}

		return nil
	}); err != nil {
		log.Fatalf("seed scenario failed (rolled back): %v", err)
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

func buildRoute(loading, unloading values.RoutePoint) (values.Route, error) {
	r, err := values.NewRoute([]values.RoutePoint{loading, unloading})
	if err != nil {
		return values.Route{}, fmt.Errorf("build route: %w", err)
	}
	return r, nil
}
