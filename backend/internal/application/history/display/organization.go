package display

import (
	"context"
	"strings"

	"codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OrganizationFormatter форматирует события организации
type OrganizationFormatter struct{}

// NewOrganizationFormatter создаёт новый OrganizationFormatter
func NewOrganizationFormatter() *OrganizationFormatter {
	return &OrganizationFormatter{}
}

// Supports проверяет, поддерживает ли форматтер данный тип события
func (f *OrganizationFormatter) Supports(eventType string) bool {
	return strings.HasPrefix(eventType, "organization.") ||
		strings.HasPrefix(eventType, "member.") ||
		strings.HasPrefix(eventType, "invitation.") ||
		strings.HasPrefix(eventType, "fraudster.")
}

// Format форматирует событие в DisplayView
func (f *OrganizationFormatter) Format(ctx context.Context, event eventstore.Event, resolver EntityResolver) (DisplayView, error) {
	switch e := event.(type) {
	case events.OrganizationCreated:
		return f.formatOrganizationCreated(e), nil
	case events.OrganizationApproved:
		return f.formatOrganizationApproved(), nil
	case events.OrganizationRejected:
		return f.formatOrganizationRejected(e), nil
	case events.OrganizationSuspended:
		return f.formatOrganizationSuspended(e), nil
	case events.OrganizationUpdated:
		return f.formatOrganizationUpdated(ctx, e, resolver), nil
	case events.MemberAdded:
		return f.formatMemberAdded(ctx, e, resolver), nil
	case events.MemberRemoved:
		return f.formatMemberRemoved(ctx, e, resolver), nil
	case events.MemberRoleChanged:
		return f.formatMemberRoleChanged(ctx, e, resolver), nil
	case events.MemberBlocked:
		return f.formatMemberBlocked(ctx, e, resolver), nil
	case events.MemberUnblocked:
		return f.formatMemberUnblocked(ctx, e, resolver), nil
	case events.InvitationCreated:
		return f.formatInvitationCreated(ctx, e, resolver), nil
	case events.InvitationAccepted:
		return f.formatInvitationAccepted(ctx, e, resolver), nil
	case events.InvitationExpired:
		return f.formatInvitationExpired(), nil
	case events.InvitationCancelled:
		return f.formatInvitationCancelled(ctx, e, resolver), nil
	case events.FraudsterMarked:
		return f.formatFraudsterMarked(e), nil
	case events.FraudsterUnmarked:
		return f.formatFraudsterUnmarked(e), nil
	default:
		return DisplayView{
			Title:       "Событие организации",
			Description: event.EventType(),
			Severity:    "info",
		}, nil
	}
}

func (f *OrganizationFormatter) formatOrganizationCreated(e events.OrganizationCreated) DisplayView {
	view := NewDisplayView("Организация создана", "Организация зарегистрирована в системе").
		WithIcon("building").
		WithSeverity("success")

	view.AddField("Название", e.Name)
	view.AddField("Юридическое название", e.LegalName)
	view.AddField("ИНН", e.INN)
	view.AddField("Email", e.Email)
	if e.Phone != "" {
		view.AddField("Телефон", e.Phone)
	}
	view.AddFieldWithType("Страна", e.Country.String(), "text")

	return view
}

func (f *OrganizationFormatter) formatOrganizationApproved() DisplayView {
	return NewDisplayView("Организация одобрена", "Организация прошла модерацию и может работать в системе").
		WithIcon("check-circle").
		WithSeverity("success")
}

func (f *OrganizationFormatter) formatOrganizationRejected(e events.OrganizationRejected) DisplayView {
	view := NewDisplayView("Организация отклонена", "Организация не прошла модерацию").
		WithIcon("x-circle").
		WithSeverity("error")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *OrganizationFormatter) formatOrganizationSuspended(e events.OrganizationSuspended) DisplayView {
	view := NewDisplayView("Организация приостановлена", "Деятельность организации временно приостановлена").
		WithIcon("pause-circle").
		WithSeverity("warning")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *OrganizationFormatter) formatOrganizationUpdated(_ context.Context, e events.OrganizationUpdated, _ EntityResolver) DisplayView {
	view := NewDisplayView("Организация обновлена", "Данные организации изменены").
		WithIcon("edit").
		WithSeverity("info")

	// Показываем что изменилось
	if e.Name != nil {
		view.AddField("Новое название", *e.Name)
	}
	if e.Phone != nil {
		view.AddField("Новый телефон", *e.Phone)
	}
	if e.Email != nil {
		view.AddField("Новый email", *e.Email)
	}
	if e.Address != nil {
		view.AddField("Новый адрес", e.Address.String())
	}

	return view
}

func (f *OrganizationFormatter) formatMemberAdded(_ context.Context, e events.MemberAdded, _ EntityResolver) DisplayView {
	view := NewDisplayView("Сотрудник добавлен", "В организацию добавлен новый сотрудник").
		WithIcon("user-plus").
		WithSeverity("success")

	view.AddField("Имя", e.Name)
	view.AddField("Email", e.Email)
	view.AddField("Роль", translateRole(e.Role.String()))

	return view
}

func (f *OrganizationFormatter) formatMemberRemoved(ctx context.Context, e events.MemberRemoved, resolver EntityResolver) DisplayView {
	memberName := resolver.ResolveMember(ctx, e.MemberID)
	description := "Сотрудник удалён из организации"
	if memberName != "" {
		description = memberName + " удалён из организации"
	}

	view := NewDisplayView("Сотрудник удалён", description).
		WithIcon("user-minus").
		WithSeverity("warning")

	if memberName != "" {
		view.AddField("Сотрудник", memberName)
	}

	return view
}

func (f *OrganizationFormatter) formatMemberRoleChanged(ctx context.Context, e events.MemberRoleChanged, resolver EntityResolver) DisplayView {
	memberName := resolver.ResolveMember(ctx, e.MemberID)
	description := "Роль сотрудника изменена"
	if memberName != "" {
		description = "Роль " + memberName + " изменена"
	}

	view := NewDisplayView("Роль изменена", description).
		WithIcon("user-cog").
		WithSeverity("info")

	if memberName != "" {
		view.AddField("Сотрудник", memberName)
	}

	view.AddDiff("Роль", translateRole(e.OldRole.String()), translateRole(e.NewRole.String()))

	return view
}

func (f *OrganizationFormatter) formatMemberBlocked(ctx context.Context, e events.MemberBlocked, resolver EntityResolver) DisplayView {
	memberName := resolver.ResolveMember(ctx, e.MemberID)
	description := "Сотрудник заблокирован"
	if memberName != "" {
		description = memberName + " заблокирован"
	}

	view := NewDisplayView("Сотрудник заблокирован", description).
		WithIcon("user-x").
		WithSeverity("warning")

	if memberName != "" {
		view.AddField("Сотрудник", memberName)
	}

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *OrganizationFormatter) formatMemberUnblocked(ctx context.Context, e events.MemberUnblocked, resolver EntityResolver) DisplayView {
	memberName := resolver.ResolveMember(ctx, e.MemberID)
	description := "Сотрудник разблокирован"
	if memberName != "" {
		description = memberName + " разблокирован"
	}

	view := NewDisplayView("Сотрудник разблокирован", description).
		WithIcon("user-check").
		WithSeverity("success")

	if memberName != "" {
		view.AddField("Сотрудник", memberName)
	}

	return view
}

func (f *OrganizationFormatter) formatInvitationCreated(_ context.Context, e events.InvitationCreated, _ EntityResolver) DisplayView {
	view := NewDisplayView("Приглашение создано", "Отправлено приглашение для нового сотрудника").
		WithIcon("mail").
		WithSeverity("info")

	view.AddField("Email", e.Email)
	view.AddField("Роль", translateRole(e.Role.String()))

	if e.Name != nil && *e.Name != "" {
		view.AddField("Имя", *e.Name)
	}

	return view
}

func (f *OrganizationFormatter) formatInvitationAccepted(ctx context.Context, e events.InvitationAccepted, resolver EntityResolver) DisplayView {
	memberName := resolver.ResolveMember(ctx, e.MemberID)
	description := "Приглашение принято"
	if memberName != "" {
		description = memberName + " принял приглашение"
	}

	view := NewDisplayView("Приглашение принято", description).
		WithIcon("check").
		WithSeverity("success")

	if memberName != "" {
		view.AddField("Новый сотрудник", memberName)
	}

	return view
}

func (f *OrganizationFormatter) formatInvitationExpired() DisplayView {
	return NewDisplayView("Приглашение истекло", "Срок действия приглашения истёк").
		WithIcon("clock").
		WithSeverity("warning")
}

func (f *OrganizationFormatter) formatInvitationCancelled(_ context.Context, _ events.InvitationCancelled, _ EntityResolver) DisplayView {
	return NewDisplayView("Приглашение отменено", "Приглашение было отменено").
		WithIcon("x").
		WithSeverity("warning")
}

func (f *OrganizationFormatter) formatFraudsterMarked(e events.FraudsterMarked) DisplayView {
	title := "Отмечен как мошенник"
	description := "Организация отмечена как подозреваемая в мошенничестве"
	severity := "warning"

	if e.IsConfirmed {
		description = "Организация подтверждена как мошенник"
		severity = "error"
	}

	view := NewDisplayView(title, description).
		WithIcon("alert-triangle").
		WithSeverity(severity)

	if e.IsConfirmed {
		view.AddField("Статус", "Подтверждено")
	} else {
		view.AddField("Статус", "Подозрение")
	}

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

func (f *OrganizationFormatter) formatFraudsterUnmarked(e events.FraudsterUnmarked) DisplayView {
	view := NewDisplayView("Снята метка мошенника", "С организации снят статус мошенника").
		WithIcon("shield-check").
		WithSeverity("success")

	if e.Reason != "" {
		view.AddField("Причина", e.Reason)
	}

	return view
}

// translateRole переводит роль на русский
func translateRole(role string) string {
	roles := map[string]string{
		"owner":         "Владелец",
		"administrator": "Администратор",
		"employee":      "Сотрудник",
	}
	if translated, ok := roles[role]; ok {
		return translated
	}
	return role
}
