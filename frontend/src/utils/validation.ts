/**
 * Утилиты для валидации полей форм.
 */

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** Возвращает сообщение об ошибке или null если email валиден */
export function validateEmail(value: string): string | null {
  if (!value) return 'Email обязателен'
  if (!EMAIL_REGEX.test(value)) return 'Введите корректный email'
  return null
}

/** Возвращает сообщение об ошибке или null если поле не пустое */
export function validateRequired(value: string, label = 'Поле'): string | null {
  if (!value?.trim()) return `${label} обязательно`
  return null
}

/** Возвращает сообщение об ошибке или null если число положительное */
export function validatePositiveNumber(value: number | undefined, label = 'Значение'): string | null {
  if (!value || value <= 0) return `${label} должно быть больше 0`
  return null
}
