package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// versionGuardUpsertSuffix строит ON CONFLICT-суффикс для полного UPSERT'а
// строки проекции с version-guard'ом:
//
//	ON CONFLICT (id) DO UPDATE SET col = EXCLUDED.col, ...
//	WHERE <table>.version <= EXCLUDED.version
//
// Используется проекциями, перешедшими на rebuild-from-aggregate: хендлер
// перечитывает агрегат из event store и пишет полное состояние. Guard
// гарантирует, что устаревший rebuild (конкурентный инстанс, обработавший
// более раннее событие) не перетрёт свежие данные, а повторная доставка
// того же события (равная версия, те же данные) останется идемпотентной.
//
// cols — все обновляемые колонки, ДОЛЖНЫ включать "version".
func versionGuardUpsertSuffix(table string, cols ...string) string {
	sets := make([]string, 0, len(cols))
	for _, c := range cols {
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
	}
	return fmt.Sprintf(
		"ON CONFLICT (id) DO UPDATE SET %s WHERE %s.version <= EXCLUDED.version",
		strings.Join(sets, ", "), table,
	)
}

// versionGuardedUpdate выполняет статусный UPDATE строки проекции с
// version-guard'ом. Используется sub-entity проекциями (members, invitations,
// support tickets), где полный rebuild из агрегата невозможен или избыточен
// (например, members_lookup хранит password_hash, которого нет в домене —
// SEC-007).
//
// Протокол при 0 затронутых строк:
//   - строки нет вообще → событие опередило Created (конкурентный инстанс ещё
//     не вставил строку) → возвращаем ошибку, Retry middleware повторит, к
//     тому моменту Created догонит;
//   - строка есть, но её version новее → устаревшее событие (out-of-order при
//     N инстансах) → молчаливый Ack: состояние уже актуальнее.
//
// version — версия агрегата на момент события (Event.Version()), монотонно
// растёт по всем событиям агрегата, поэтому годится как guard для строк его
// sub-entities.
func versionGuardedUpdate(
	ctx context.Context,
	db dbtx.TxManager,
	psql squirrel.StatementBuilderType,
	table string,
	id uuid.UUID,
	version int64,
	sets map[string]any,
) error {
	sets["version"] = version
	query, args, err := psql.
		Update(table).
		SetMap(sets).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.LtOrEq{"version": version}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build guarded update for %s: %w", table, err)
	}

	res, err := db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec guarded update for %s: %w", table, err)
	}
	if res.RowsAffected() > 0 {
		return nil
	}

	var existing int64
	checkQuery, checkArgs, err := psql.
		Select("version").
		From(table).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build version check for %s: %w", table, err)
	}
	if err := db.QueryRow(ctx, checkQuery, checkArgs...).Scan(&existing); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%s row %s not found yet, event v%d before created — retry", table, id, version)
		}
		return fmt.Errorf("check %s row version: %w", table, err)
	}

	// Строка есть, но её версия новее события — устаревший апдейт, Ack.
	return nil
}
