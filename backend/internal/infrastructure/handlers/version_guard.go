package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// errProjectionRowMissing — строка проекции ещё не существует: событие
// опередило Created (out-of-order при N инстансах). По умолчанию хендлер
// возвращает ошибку как есть → Retry middleware повторит доставку, когда
// Created догонит. Хендлеры с tombstone-семантикой (members) проверяют по
// errors.Is, не относится ли «отсутствие строки» к удалённой сущности —
// тогда событие ack'ается вместо вечного retry.
var errProjectionRowMissing = errors.New("projection row missing")

// versionGuardedUpdate выполняет статусный UPDATE строки проекции с
// version-guard'ом. Используется sub-entity проекциями (members, invitations,
// support tickets), где полный rebuild из агрегата невозможен или избыточен
// (например, members_lookup хранит password_hash, которого нет в домене —
// SEC-007).
//
// versionCol — колонка-guard. Должна сравнивать версии ТОЛЬКО событий, которые
// меняют один и тот же набор полей: ортогональные поля одной строки (role и
// status участника) обязаны иметь раздельные version-колонки, иначе свежее
// событие одного поля заставит guard молча выбросить более раннее (но
// единственное актуальное) событие другого поля.
//
// Протокол при 0 затронутых строк:
//   - строки нет вообще → возвращаем errProjectionRowMissing (см. выше);
//   - строка есть, но её versionCol новее → устаревшее событие (out-of-order
//     при N инстансах) → молчаливый Ack: состояние уже актуальнее.
//
// version — версия агрегата на момент события (Event.Version()), монотонно
// растёт по всем событиям агрегата, поэтому годится как guard для строк его
// sub-entities.
func versionGuardedUpdate(
	ctx context.Context,
	db dbtx.TxManager,
	psql squirrel.StatementBuilderType,
	table string,
	versionCol string,
	id uuid.UUID,
	version int64,
	sets map[string]any,
) error {
	sets[versionCol] = version
	query, args, err := psql.
		Update(table).
		SetMap(sets).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.LtOrEq{versionCol: version}).
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
		Select(versionCol).
		From(table).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build version check for %s: %w", table, err)
	}
	if err := db.QueryRow(ctx, checkQuery, checkArgs...).Scan(&existing); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%s row %s not found yet, event v%d before created: %w", table, id, version, errProjectionRowMissing)
		}
		return fmt.Errorf("check %s row version: %w", table, err)
	}

	// Строка есть, но её версия новее события — устаревший апдейт, Ack.
	return nil
}
