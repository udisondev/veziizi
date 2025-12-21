package display

import (
	"context"
	"fmt"
	"strings"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// FreightRequestFormatter форматирует события заявок на перевозку
type FreightRequestFormatter struct{}

// NewFreightRequestFormatter создаёт новый FreightRequestFormatter
func NewFreightRequestFormatter() *FreightRequestFormatter {
	return &FreightRequestFormatter{}
}

// Supports проверяет, поддерживает ли форматтер данный тип события
func (f *FreightRequestFormatter) Supports(eventType string) bool {
	return strings.HasPrefix(eventType, "freight_request.") ||
		strings.HasPrefix(eventType, "offer.")
}

// Format форматирует событие в DisplayView
func (f *FreightRequestFormatter) Format(ctx context.Context, event eventstore.Event, resolver EntityResolver) (DisplayView, error) {
	switch e := event.(type) {
	case events.FreightRequestCreated:
		return f.formatCreated(ctx, e, resolver), nil
	case events.FreightRequestUpdated:
		return f.formatUpdated(ctx, e, resolver), nil
	case events.FreightRequestReassigned:
		return f.formatReassigned(ctx, e, resolver), nil
	case events.FreightRequestCancelled:
		return f.formatCancelled(ctx, e, resolver), nil
	case events.FreightRequestExpired:
		return f.formatExpired(), nil
	case events.OfferMade:
		return f.formatOfferMade(ctx, e, resolver), nil
	case events.OfferWithdrawn:
		return f.formatOfferWithdrawn(ctx, e, resolver), nil
	case events.OfferSelected:
		return f.formatOfferSelected(ctx, e, resolver), nil
	case events.OfferRejected:
		return f.formatOfferRejected(ctx, e, resolver), nil
	case events.OfferConfirmed:
		return f.formatOfferConfirmed(ctx, e, resolver), nil
	case events.OfferDeclined:
		return f.formatOfferDeclined(ctx, e, resolver), nil
	default:
		return DisplayView{
			Title:       "Событие заявки",
			Description: event.EventType(),
			Severity:    "info",
		}, nil
	}
}

func (f *FreightRequestFormatter) formatCreated(ctx context.Context, e events.FreightRequestCreated, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Заявка создана", fmt.Sprintf("Создана заявка №%d", e.RequestNumber)).
		WithIcon("file-plus").
		WithSeverity("success")

	view.AddField("Номер заявки", fmt.Sprintf("№%d", e.RequestNumber))
	view.AddField("Маршрут", formatRoute(e.Route))

	customerOrg := resolver.ResolveOrganization(ctx, e.CustomerOrgID)
	if customerOrg != "" {
		view.AddField("Заказчик", customerOrg)
	}

	return view
}

func (f *FreightRequestFormatter) formatUpdated(_ context.Context, e events.FreightRequestUpdated, _ EntityResolver) DisplayView {
	view := NewDisplayView("Заявка обновлена", "Данные заявки изменены").
		WithIcon("edit").
		WithSeverity("info")

	// Показываем что изменилось
	var changes []string
	if e.Route != nil {
		changes = append(changes, "маршрут")
	}
	if e.Cargo != nil {
		changes = append(changes, "груз")
	}
	if e.VehicleRequirements != nil {
		changes = append(changes, "требования к ТС")
	}
	if e.Payment != nil {
		changes = append(changes, "оплата")
	}
	if e.Comment != nil {
		changes = append(changes, "комментарий")
	}

	if len(changes) > 0 {
		view.AddField("Изменено", strings.Join(changes, ", "))
	}

	return view
}

func (f *FreightRequestFormatter) formatReassigned(ctx context.Context, e events.FreightRequestReassigned, resolver EntityResolver) DisplayView {
	oldMember := resolver.ResolveMember(ctx, e.OldMemberID)
	newMember := resolver.ResolveMember(ctx, e.NewMemberID)

	view := NewDisplayView("Заявка переназначена", "Заявка передана другому сотруднику").
		WithIcon("user-switch").
		WithSeverity("info")

	if oldMember != "" && newMember != "" {
		view.AddDiff("Ответственный", oldMember, newMember)
	} else {
		if oldMember != "" {
			view.AddField("Был", oldMember)
		}
		if newMember != "" {
			view.AddField("Стал", newMember)
		}
	}

	return view
}

func (f *FreightRequestFormatter) formatCancelled(_ context.Context, e events.FreightRequestCancelled, _ EntityResolver) DisplayView {
	view := NewDisplayView("Заявка отменена", "Заявка на перевозку отменена").
		WithIcon("x-circle").
		WithSeverity("warning")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *FreightRequestFormatter) formatExpired() DisplayView {
	return NewDisplayView("Заявка истекла", "Срок действия заявки истёк").
		WithIcon("clock").
		WithSeverity("warning")
}

func (f *FreightRequestFormatter) formatOfferMade(ctx context.Context, e events.OfferMade, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Оффер сделан", "Перевозчик сделал предложение").
		WithIcon("truck").
		WithSeverity("info")

	carrierOrg := resolver.ResolveOrganization(ctx, e.CarrierOrgID)
	if carrierOrg != "" {
		view.AddField("Перевозчик", carrierOrg)
	}

	view.AddFieldWithType("Цена", formatMoney(e.Price), "money")
	view.AddField("НДС", translateVatType(e.VatType.String()))
	view.AddField("Способ оплаты", translatePaymentMethod(e.PaymentMethod.String()))

	if e.Comment != "" {
		view.AddField("Комментарий", e.Comment)
	}

	return view
}

func (f *FreightRequestFormatter) formatOfferWithdrawn(_ context.Context, e events.OfferWithdrawn, _ EntityResolver) DisplayView {
	view := NewDisplayView("Оффер отозван", "Перевозчик отозвал своё предложение").
		WithIcon("undo").
		WithSeverity("warning")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *FreightRequestFormatter) formatOfferSelected(_ context.Context, _ events.OfferSelected, _ EntityResolver) DisplayView {
	return NewDisplayView("Оффер выбран", "Заказчик выбрал предложение перевозчика").
		WithIcon("check").
		WithSeverity("success")
}

func (f *FreightRequestFormatter) formatOfferRejected(_ context.Context, e events.OfferRejected, _ EntityResolver) DisplayView {
	view := NewDisplayView("Оффер отклонён", "Заказчик отклонил предложение").
		WithIcon("x").
		WithSeverity("warning")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *FreightRequestFormatter) formatOfferConfirmed(_ context.Context, _ events.OfferConfirmed, _ EntityResolver) DisplayView {
	return NewDisplayView("Оффер подтверждён", "Перевозчик подтвердил заказ").
		WithIcon("check-circle").
		WithSeverity("success")
}

func (f *FreightRequestFormatter) formatOfferDeclined(_ context.Context, e events.OfferDeclined, _ EntityResolver) DisplayView {
	view := NewDisplayView("Оффер отклонён перевозчиком", "Перевозчик отказался от заказа").
		WithIcon("x-circle").
		WithSeverity("error")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

// formatRoute форматирует маршрут в читаемую строку
func formatRoute(route values.Route) string {
	if len(route.Points) == 0 {
		return ""
	}

	var loading, unloading string
	for _, p := range route.Points {
		if p.IsLoading && loading == "" {
			loading = p.Address
		}
		if p.IsUnloading {
			unloading = p.Address
		}
	}

	if loading != "" && unloading != "" {
		return loading + " → " + unloading
	}
	if loading != "" {
		return loading
	}
	return unloading
}

// formatMoney форматирует деньги
func formatMoney(m values.Money) string {
	// Конвертируем копейки в рубли
	rubles := float64(m.Amount) / 100
	return fmt.Sprintf("%.2f %s", rubles, m.Currency.String())
}

// translateVatType переводит тип НДС
func translateVatType(vatType string) string {
	types := map[string]string{
		"with_vat":    "С НДС",
		"without_vat": "Без НДС",
	}
	if translated, ok := types[vatType]; ok {
		return translated
	}
	return vatType
}

// translatePaymentMethod переводит способ оплаты
func translatePaymentMethod(method string) string {
	methods := map[string]string{
		"cash":          "Наличные",
		"bank_transfer": "Безналичный расчёт",
		"card":          "Карта",
	}
	if translated, ok := methods[method]; ok {
		return translated
	}
	return method
}
