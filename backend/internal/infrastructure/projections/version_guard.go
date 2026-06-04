package projections

import (
	"fmt"
	"strings"
)

// VersionGuardUpsertSuffix строит ON CONFLICT-суффикс для полного UPSERT'а
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
func VersionGuardUpsertSuffix(table string, cols ...string) string {
	sets := make([]string, 0, len(cols))
	for _, c := range cols {
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
	}
	return fmt.Sprintf(
		"ON CONFLICT (id) DO UPDATE SET %s WHERE %s.version <= EXCLUDED.version",
		strings.Join(sets, ", "), table,
	)
}
