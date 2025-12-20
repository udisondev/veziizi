package projections

import "strings"

// SEC-014: Экранирование спецсимволов PostgreSQL ILIKE
// Символы %, _, \ имеют специальное значение в ILIKE и должны быть экранированы

// EscapeLikePattern экранирует спецсимволы для безопасного использования в ILIKE
// Также ограничивает длину строки для защиты от DoS
func EscapeLikePattern(s string) string {
	// Ограничиваем длину поискового запроса
	const maxSearchLen = 100
	if len(s) > maxSearchLen {
		s = s[:maxSearchLen]
	}

	// Экранируем спецсимволы PostgreSQL ILIKE
	// Порядок важен: сначала \, потом % и _
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)

	return s
}

// WrapLikePattern добавляет % вокруг экранированного паттерна для поиска "contains"
func WrapLikePattern(s string) string {
	return "%" + EscapeLikePattern(s) + "%"
}
