package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// MembersHandler обновляет members_lookup. Rebuild-from-aggregate здесь
// невозможен: строка хранит password_hash, которого нет в домене (SEC-007),
// поэтому используется per-event модель с защитами от at-least-once и
// out-of-order доставки:
//   - role и status guard'ятся РАЗДЕЛЬНЫМИ version-колонками (role_version,
//     status_version) — ортогональные поля, общий version терял бы обновление
//     одного поля после свежего события другого;
//   - удаление через tombstone (members_removed): повторно доставленный
//     MemberAdded не воскрешает удалённого участника, а статусное событие
//     после удаления ack'ается, а не уходит в вечный retry;
//   - Added/Removed сериализуются advisory xact-lock'ом по member id, чтобы
//     конкурентные инстансы не проскочили между проверкой tombstone и INSERT.
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

// memberRemoved проверяет tombstone удалённого участника.
func (h *MembersHandler) memberRemoved(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	if err := h.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM members_removed WHERE id = $1)`, id,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check members_removed: %w", err)
	}
	return exists, nil
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

	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := lockProjectionRow(ctx, h.db, e.MemberID); err != nil {
			return err
		}

		// INSERT ... SELECT WHERE NOT EXISTS(tombstone): повторно доставленный
		// Added после Removed не воскрешает строку. role_version/status_version
		// инициализируются версией события Added — baseline guard'а, а не
		// DEFAULT 0.
		query := `
			INSERT INTO members_lookup (
				id, organization_id, email, password_hash, name, phone,
				telegram_id, role, status, created_at,
				registration_ip, registration_fingerprint, registration_user_agent,
				role_version, status_version
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14
			WHERE NOT EXISTS (SELECT 1 FROM members_removed WHERE id = $1)
			ON CONFLICT (id) DO NOTHING
		`
		result, err := h.db.Exec(ctx, query,
			e.MemberID, e.AggregateID(), e.Email, e.PasswordHash, e.Name, e.Phone,
			nil, e.Role.String(), "active", e.OccurredAt(),
			regIP, regFingerprint, regUserAgent,
			e.Version(),
		)
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
			slog.Debug("member already exists or was removed, idempotent replay",
				slog.String("member_id", e.MemberID.String()))
			return nil
		}

		slog.Debug("member added to lookup", slog.String("member_id", e.MemberID.String()))
		return nil
	})
}

// OnMemberRemoved пишет tombstone и удаляет строку одной tx (под advisory
// lock'ом — см. lockProjectionRow). Повтор идемпотентен: tombstone ON CONFLICT
// DO NOTHING, DELETE — no-op.
func (h *MembersHandler) OnMemberRemoved(ctx context.Context, e *events.MemberRemoved) error {
	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := lockProjectionRow(ctx, h.db, e.MemberID); err != nil {
			return err
		}

		if _, err := h.db.Exec(ctx,
			`INSERT INTO members_removed (id) VALUES ($1) ON CONFLICT DO NOTHING`, e.MemberID,
		); err != nil {
			return fmt.Errorf("failed to insert member tombstone: %w", err)
		}

		if _, err := h.db.Exec(ctx,
			`DELETE FROM members_lookup WHERE id = $1`, e.MemberID,
		); err != nil {
			return fmt.Errorf("failed to delete member: %w", err)
		}

		slog.Debug("member removed from lookup", slog.String("member_id", e.MemberID.String()))
		return nil
	})
}

// Статусные апдейты идут через versionGuardedUpdate по СВОЕЙ version-колонке:
// устаревшее событие того же поля (out-of-order при N инстансах) не перетирает
// свежее, событие раньше Created уходит в retry до появления строки, событие
// после Removed ack'ается по tombstone'у.

// guardedFieldUpdate — общий путь role/status апдейтов: version guard +
// tombstone-aware обработка отсутствующей строки.
func (h *MembersHandler) guardedFieldUpdate(ctx context.Context, memberID uuid.UUID, versionCol string, version int64, sets map[string]any) error {
	err := versionGuardedUpdate(ctx, h.db, h.psql, "members_lookup", versionCol, memberID, version, sets)
	if err == nil {
		return nil
	}
	if errors.Is(err, errProjectionRowMissing) {
		removed, checkErr := h.memberRemoved(ctx, memberID)
		if checkErr != nil {
			return checkErr
		}
		if removed {
			slog.Debug("member already removed, acking stale status event",
				slog.String("member_id", memberID.String()))
			return nil
		}
	}
	return err
}

func (h *MembersHandler) OnMemberRoleChanged(ctx context.Context, e *events.MemberRoleChanged) error {
	if err := h.guardedFieldUpdate(ctx, e.MemberID, "role_version", e.Version(), map[string]any{
		"role": e.NewRole.String(),
	}); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	slog.Debug("member role updated", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) OnMemberBlocked(ctx context.Context, e *events.MemberBlocked) error {
	if err := h.guardedFieldUpdate(ctx, e.MemberID, "status_version", e.Version(), map[string]any{
		"status": "blocked",
	}); err != nil {
		return fmt.Errorf("failed to block member: %w", err)
	}

	slog.Debug("member blocked", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) OnMemberUnblocked(ctx context.Context, e *events.MemberUnblocked) error {
	if err := h.guardedFieldUpdate(ctx, e.MemberID, "status_version", e.Version(), map[string]any{
		"status": "active",
	}); err != nil {
		return fmt.Errorf("failed to unblock member: %w", err)
	}

	slog.Debug("member unblocked", slog.String("member_id", e.MemberID.String()))
	return nil
}
