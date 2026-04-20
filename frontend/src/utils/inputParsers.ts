/**
 * Утилиты для парсинга значений из нативных input-событий.
 * Используются в обработчиках @input числовых полей форм.
 */

/** Парсит float из input-события. Возвращает 0 если поле пустое/невалидное */
export function parseInputFloat(event: Event): number {
  return parseFloat((event.target as HTMLInputElement).value) || 0
}

/** Парсит float из input-события. Возвращает undefined если поле пустое */
export function parseInputFloatOrUndefined(event: Event): number | undefined {
  const val = parseFloat((event.target as HTMLInputElement).value)
  return isNaN(val) ? undefined : val
}

/** Парсит int из input-события. Возвращает undefined если поле пустое */
export function parseInputInt(event: Event): number | undefined {
  const val = parseInt((event.target as HTMLInputElement).value)
  return isNaN(val) ? undefined : val
}

/** Читает строковое значение из input-события */
export function parseInputString(event: Event): string {
  return (event.target as HTMLInputElement).value
}
