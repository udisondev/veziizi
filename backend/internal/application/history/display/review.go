package display

import (
	"context"
	"fmt"
	"strings"

	"codeberg.org/udison/veziizi/backend/internal/domain/review/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// ReviewFormatter форматирует события отзывов
type ReviewFormatter struct{}

// NewReviewFormatter создаёт новый ReviewFormatter
func NewReviewFormatter() *ReviewFormatter {
	return &ReviewFormatter{}
}

// Supports проверяет, поддерживает ли форматтер данный тип события
func (f *ReviewFormatter) Supports(eventType string) bool {
	return strings.HasPrefix(eventType, "review.")
}

// Format форматирует событие в DisplayView
func (f *ReviewFormatter) Format(ctx context.Context, event eventstore.Event, resolver EntityResolver) (DisplayView, error) {
	switch e := event.(type) {
	case events.ReviewReceived:
		return f.formatReceived(ctx, e, resolver), nil
	case events.ReviewAnalyzed:
		return f.formatAnalyzed(e), nil
	case events.ReviewApproved:
		return f.formatApproved(e), nil
	case events.ReviewRejected:
		return f.formatRejected(e), nil
	case events.ReviewActivated:
		return f.formatActivated(e), nil
	case events.ReviewDeactivated:
		return f.formatDeactivated(e), nil
	default:
		return DisplayView{
			Title:       "Событие отзыва",
			Description: event.EventType(),
			Severity:    "info",
		}, nil
	}
}

func (f *ReviewFormatter) formatReceived(ctx context.Context, e events.ReviewReceived, resolver EntityResolver) DisplayView {
	view := NewDisplayView("Отзыв получен", "Получен новый отзыв для обработки").
		WithIcon("inbox").
		WithSeverity("info")

	reviewerOrg := resolver.ResolveOrganization(ctx, e.ReviewerOrgID)
	if reviewerOrg != "" {
		view.AddField("Автор отзыва", reviewerOrg)
	}

	reviewedOrg := resolver.ResolveOrganization(ctx, e.ReviewedOrgID)
	if reviewedOrg != "" {
		view.AddField("Получатель", reviewedOrg)
	}

	view.AddField("Оценка", formatReviewRating(e.Rating))

	if e.Comment != "" {
		view.AddField("Комментарий", e.Comment)
	}

	// Сумма заказа
	if e.OrderAmount > 0 {
		amount := float64(e.OrderAmount) / 100
		view.AddFieldWithType("Сумма заказа", fmt.Sprintf("%.2f %s", amount, e.OrderCurrency), "money")
	}

	return view
}

func (f *ReviewFormatter) formatAnalyzed(e events.ReviewAnalyzed) DisplayView {
	title := "Отзыв проанализирован"
	description := "Проведён автоматический анализ отзыва"
	severity := "info"

	if e.RequiresModeration {
		description = "Отзыв требует модерации"
		severity = "warning"
	}

	view := NewDisplayView(title, description).
		WithIcon("search").
		WithSeverity(severity)

	view.AddField("Вес отзыва", fmt.Sprintf("%.2f", e.RawWeight))
	view.AddField("Оценка фрода", fmt.Sprintf("%.1f%%", e.FraudScore*100))

	if e.RequiresModeration {
		view.AddFieldWithType("Требует модерации", "Да", "status")
	} else {
		view.AddFieldWithType("Требует модерации", "Нет", "status")
	}

	// Показываем сигналы фрода
	if len(e.FraudSignals) > 0 {
		var signals []string
		for _, s := range e.FraudSignals {
			signals = append(signals, s.Description)
		}
		view.AddField("Обнаружено", strings.Join(signals, "; "))
	}

	view.AddFieldWithType("Дата активации", e.ActivationDate.Format("02.01.2006"), "date")

	return view
}

func (f *ReviewFormatter) formatApproved(e events.ReviewApproved) DisplayView {
	description := "Отзыв одобрен модератором"
	if e.ApprovedBy == nil {
		description = "Отзыв одобрен автоматически"
	}

	view := NewDisplayView("Отзыв одобрен", description).
		WithIcon("check-circle").
		WithSeverity("success")

	view.AddField("Финальный вес", fmt.Sprintf("%.2f", e.FinalWeight))

	if e.Note != "" {
		view.AddField("Примечание", e.Note)
	}

	return view
}

func (f *ReviewFormatter) formatRejected(e events.ReviewRejected) DisplayView {
	view := NewDisplayView("Отзыв отклонён", "Отзыв отклонён модератором").
		WithIcon("x-circle").
		WithSeverity("error")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *ReviewFormatter) formatActivated(e events.ReviewActivated) DisplayView {
	view := NewDisplayView("Отзыв активирован", "Отзыв начал влиять на рейтинг организации").
		WithIcon("zap").
		WithSeverity("success")

	view.AddField("Финальный вес", fmt.Sprintf("%.2f", e.FinalWeight))

	return view
}

func (f *ReviewFormatter) formatDeactivated(e events.ReviewDeactivated) DisplayView {
	view := NewDisplayView("Отзыв деактивирован", "Отзыв больше не влияет на рейтинг").
		WithIcon("power-off").
		WithSeverity("warning")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

// formatReviewRating форматирует рейтинг звёздами
func formatReviewRating(rating int) string {
	if rating < 1 || rating > 5 {
		return fmt.Sprintf("%d", rating)
	}
	return strings.Repeat("★", rating) + strings.Repeat("☆", 5-rating)
}
