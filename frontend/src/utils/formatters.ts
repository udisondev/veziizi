/**
 * Утилиты форматирования для фронтенда
 * Централизованные функции для форматирования дат, денег, веса и других значений
 */

import type { Currency, Money } from '@/types/freightRequest'

// ============================================================================
// Форматирование дат
// ============================================================================

/**
 * Форматирует дату со временем
 * @example formatDateTime('2024-01-15T10:30:00Z') -> '15.01.2024, 10:30'
 */
export function formatDateTime(dateStr: string | Date): string {
  const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr
  return date.toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Форматирует только дату (без времени)
 * @example formatDate('2024-01-15T10:30:00Z') -> '15.01.2024'
 */
export function formatDate(dateStr: string | Date): string {
  const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr
  return date.toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

/**
 * Форматирует дату в коротком формате
 * @example formatDateShort('2024-01-15T10:30:00Z') -> '15 янв. 2024'
 */
export function formatDateShort(dateStr: string | Date): string {
  const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr
  return date.toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

/**
 * Форматирует относительное время
 * @example formatRelativeTime('2024-01-15T10:30:00Z') -> 'вчера' или '2 дня назад'
 */
export function formatRelativeTime(dateStr: string | Date): string {
  const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays === 0) {
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
    if (diffHours === 0) {
      const diffMinutes = Math.floor(diffMs / (1000 * 60))
      if (diffMinutes < 1) return 'только что'
      return `${diffMinutes} мин. назад`
    }
    return `${diffHours} ч. назад`
  }
  if (diffDays === 1) return 'вчера'
  if (diffDays < 7) return `${diffDays} дн. назад`
  return formatDate(date)
}

// ============================================================================
// Форматирование денег
// ============================================================================

const currencySymbols: Record<Currency, string> = {
  RUB: '₽',
  KZT: '₸',
  BYN: 'Br',
  EUR: '€',
  USD: '$',
}

/**
 * Форматирует деньги (из копеек в рубли с символом валюты)
 * @example formatMoney({ amount: 150000, currency: 'RUB' }) -> '1 500 ₽'
 */
export function formatMoney(money: Money): string {
  const amount = money.amount / 100 // Конвертация из копеек
  const formatted = new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(amount)
  return `${formatted} ${currencySymbols[money.currency]}`
}

/**
 * Форматирует сумму без символа валюты
 * @example formatAmount(150000) -> '1 500'
 */
export function formatAmount(amount: number): string {
  return new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(amount / 100)
}

// ============================================================================
// Форматирование веса и размеров
// ============================================================================

/**
 * Форматирует вес в тоннах
 * @example formatWeight(1500) -> '1.5 т'
 */
export function formatWeight(weightKg: number): string {
  if (weightKg >= 1000) {
    const tons = weightKg / 1000
    return `${tons.toLocaleString('ru-RU', { maximumFractionDigits: 1 })} т`
  }
  return `${weightKg.toLocaleString('ru-RU')} кг`
}

/**
 * Форматирует объём
 * @example formatVolume(15.5) -> '15,5 м³'
 */
export function formatVolume(volumeM3: number): string {
  return `${volumeM3.toLocaleString('ru-RU', { maximumFractionDigits: 1 })} м³`
}

/**
 * Форматирует размеры файла
 * @example formatFileSize(1536000) -> '1.5 МБ'
 */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} Б`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`
  return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`
}

// ============================================================================
// Форматирование телефонов
// ============================================================================

/**
 * Форматирует телефон в международном формате
 * @example formatPhone('79991234567') -> '+7 (999) 123-45-67'
 */
export function formatPhone(phone: string): string {
  const cleaned = phone.replace(/\D/g, '')
  if (cleaned.length === 11 && cleaned.startsWith('7')) {
    return `+7 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7, 9)}-${cleaned.slice(9, 11)}`
  }
  return phone
}
