package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type MembersHandler struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewMembersHandler(db dbtx.TxManager) *MembersHandler {
	return &MembersHandler{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (h *MembersHandler) OnMemberAdded(ctx context.Context, e *events.MemberAdded) error {
	// Преобразуем пустые строки в nil для INET колонок
	var regIP, regFingerprint, regUserAgent any
	if e.RegistrationIP != "" {
		regIP = e.RegistrationIP
	}
	if e.RegistrationFingerprint != "" {
		regFingerprint = e.RegistrationFingerprint
	}
	if e.RegistrationUserAgent != "" {
		regUserAgent = e.RegistrationUserAgent
	}

	query, args, err := h.psql.
		Insert("members_lookup").
		Columns(
			"id", "organization_id", "email", "password_hash", "name", "phone",
			"telegram_id", "role", "status", "created_at",
			"registration_ip", "registration_fingerprint", "registration_user_agent",
		).
		Values(
			e.MemberID, e.AggregateID(), e.Email, e.PasswordHash, e.Name, e.Phone,
			nil, e.Role.String(), "active", e.OccurredAt(),
			regIP, regFingerprint, regUserAgent,
		).
		Suffix("ON CONFLICT (id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	result, err := h.db.Exec(ctx, query, args...)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
		slog.Error("member email already exists in lookup, domain invariant violated",
			slog.String("member_id", e.MemberID.String()),
			slog.String("email", e.Email),
			slog.String("constraint", pgErr.ConstraintName),
		)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to insert member: %w", err)
	}

	if result.RowsAffected() == 0 {
		slog.Debug("member already exists in lookup, idempotent replay", slog.String("member_id", e.MemberID.String()))
		return nil
	}

	slog.Debug("member added to lookup", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) OnMemberRemoved(ctx context.Context, e *events.MemberRemoved) error {
	query, args, err := h.psql.
		Delete("members_lookup").
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}

	slog.Debug("member removed from lookup", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) OnMemberRoleChanged(ctx context.Context, e *events.MemberRoleChanged) error {
	query, args, err := h.psql.
		Update("members_lookup").
		Set("role", e.NewRole.String()).
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	slog.Debug("member role updated", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) OnMemberBlocked(ctx context.Context, e *events.MemberBlocked) error {
	query, args, err := h.psql.
		Update("members_lookup").
		Set("status", "blocked").
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to block member: %w", err)
	}

	slog.Debug("member blocked", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) OnMemberUnblocked(ctx context.Context, e *events.MemberUnblocked) error {
	query, args, err := h.psql.
		Update("members_lookup").
		Set("status", "active").
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to unblock member: %w", err)
	}

	slog.Debug("member unblocked", slog.String("member_id", e.MemberID.String()))
	return nil
}
