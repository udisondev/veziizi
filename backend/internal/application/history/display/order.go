package display

import (
	"context"
	"fmt"
	"strings"

	"codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OrderFormatter форматирует события заказов
type OrderFormatter struct{}

// NewOrderFormatter создаёт новый OrderFormatter
func NewOrderFormatter() *OrderFormatter {
	return &OrderFormatter{}
}

// Supports проверяет, поддерживает ли форматтер данный тип события
func (f *OrderFormatter) Supports(eventType string) bool {
	return strings.HasPrefix(eventType, "order.")
}

// Format форматирует событие в DisplayView
func (f *OrderFormatter) Format(ctx context.Context, event eventstore.Event, resolver EntityResolver) (DisplayView, error) {
	switch e := event.(type) {
	case events.OrderCreated:
		return f.formatCreated(ctx, e, resolver), nil
	case events.OrderCancelled:
		return f.formatCancelled(ctx, e, resolver), nil
	case events.CustomerCompleted:
		return f.formatCustomerCompleted(ctx, e, resolver), nil
	case events.CarrierCompleted:
		return f.formatCarrierCompleted(ctx, e, resolver), nil
	case events.OrderCompleted:
		return f.formatCompleted(), nil
	case events.MessageSent:
		return f.formatMessageSent(ctx, e, resolver), nil
	case events.DocumentAttached:
		return f.formatDocumentAttached(ctx, e, resolver), nil
	case events.DocumentRemoved:
		return f.formatDocumentRemoved(ctx, e, resolver), nil
	case events.ReviewLeft:
		return f.formatReviewLeft(ctx, e, resolver), nil
	default:
		return DisplayView{
			Title:       "Событие заказа",
			Description: event.EventType(),
			Severity:    "info",
		}, nil
	}
}

func (f *OrderFormatter) formatCreated(ctx context.Context, e events.OrderCreated, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Заказ создан", fmt.Sprintf("Создан заказ №%d", e.OrderNumber)).
		WithIcon("package").
		WithSeverity("success")

	view.AddField("Номер заказа", fmt.Sprintf("№%d", e.OrderNumber))

	customerOrg := resolver.ResolveOrganization(ctx, e.CustomerOrgID)
	if customerOrg != "" {
		view.AddField("Заказчик", customerOrg)
	}

	carrierOrg := resolver.ResolveOrganization(ctx, e.CarrierOrgID)
	if carrierOrg != "" {
		view.AddField("Перевозчик", carrierOrg)
	}

	return view
}

func (f *OrderFormatter) formatCancelled(ctx context.Context, e events.OrderCancelled, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Заказ отменён", "Заказ был отменён").
		WithIcon("x-circle").
		WithSeverity("error")

	cancelledByOrg := resolver.ResolveOrganization(ctx, e.CancelledByOrgID)
	if cancelledByOrg != "" {
		view.AddField("Отменил (организация)", cancelledByOrg)
	}

	cancelledBy := resolver.ResolveMember(ctx, e.CancelledByMemberID)
	if cancelledBy != "" {
		view.AddField("Отменил (сотрудник)", cancelledBy)
	}

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *OrderFormatter) formatCustomerCompleted(ctx context.Context, e events.CustomerCompleted, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Заказчик завершил", "Заказчик подтвердил выполнение заказа").
		WithIcon("check").
		WithSeverity("success")

	completedBy := resolver.ResolveMember(ctx, e.MemberID)
	if completedBy != "" {
		view.AddField("Подтвердил", completedBy)
	}

	return view
}

func (f *OrderFormatter) formatCarrierCompleted(ctx context.Context, e events.CarrierCompleted, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Перевозчик завершил", "Перевозчик подтвердил выполнение заказа").
		WithIcon("check").
		WithSeverity("success")

	completedBy := resolver.ResolveMember(ctx, e.MemberID)
	if completedBy != "" {
		view.AddField("Подтвердил", completedBy)
	}

	return view
}

func (f *OrderFormatter) formatCompleted() DisplayView {
	return NewDisplayView("Заказ завершён", "Обе стороны подтвердили выполнение заказа").
		WithIcon("check-circle").
		WithSeverity("success")
}

func (f *OrderFormatter) formatMessageSent(ctx context.Context, e events.MessageSent, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Сообщение отправлено", "Новое сообщение в чате заказа").
		WithIcon("message-circle").
		WithSeverity("info")

	senderOrg := resolver.ResolveOrganization(ctx, e.SenderOrgID)
	if senderOrg != "" {
		view.AddField("Организация", senderOrg)
	}

	sender := resolver.ResolveMember(ctx, e.SenderMemberID)
	if sender != "" {
		view.AddField("Отправитель", sender)
	}

	// Показываем полный текст сообщения
	if e.Content != "" {
		view.AddField("Сообщение", e.Content)
	}

	return view
}

func (f *OrderFormatter) formatDocumentAttached(ctx context.Context, e events.DocumentAttached, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Документ прикреплён", "К заказу прикреплён новый документ").
		WithIcon("file-plus").
		WithSeverity("info")

	view.AddField("Имя файла", e.Name)
	view.AddField("Размер", formatFileSize(e.Size))

	uploadedBy := resolver.ResolveMember(ctx, e.UploadedBy)
	if uploadedBy != "" {
		view.AddField("Загрузил", uploadedBy)
	}

	return view
}

func (f *OrderFormatter) formatDocumentRemoved(ctx context.Context, e events.DocumentRemoved, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Документ удалён", "Документ удалён из заказа").
		WithIcon("file-minus").
		WithSeverity("warning")

	removedBy := resolver.ResolveMember(ctx, e.RemovedBy)
	if removedBy != "" {
		view.AddField("Удалил", removedBy)
	}

	return view
}

func (f *OrderFormatter) formatReviewLeft(ctx context.Context, e events.ReviewLeft, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Отзыв оставлен", "Оставлен отзыв о работе").
		WithIcon("star").
		WithSeverity("info")

	reviewerOrg := resolver.ResolveOrganization(ctx, e.ReviewerOrgID)
	if reviewerOrg != "" {
		view.AddField("Автор отзыва", reviewerOrg)
	}

	view.AddField("Оценка", formatRating(e.Rating))

	if e.Comment != "" {
		view.AddField("Комментарий", e.Comment)
	}

	return view
}

// formatFileSize форматирует размер файла
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f ГБ", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f МБ", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f КБ", float64(size)/KB)
	default:
		return fmt.Sprintf("%d Б", size)
	}
}

// formatRating форматирует рейтинг звёздами
func formatRating(rating int) string {
	if rating < 1 || rating > 5 {
		return fmt.Sprintf("%d", rating)
	}
	return strings.Repeat("★", rating) + strings.Repeat("☆", 5-rating)
}
