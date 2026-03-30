/**
 * Mock Data Generators
 * Утилиты для генерации mock данных
 */

let idCounter = 0

/**
 * Генерация уникального ID
 */
export function generateId(prefix = 'sandbox'): string {
  return `${prefix}-${++idCounter}-${Date.now().toString(36)}`
}

/**
 * Сброс счётчика ID (для тестов)
 */
export function resetIdCounter(): void {
  idCounter = 0
}

/**
 * Генерация номера заявки
 */
let requestNumberCounter = 1000
export function generateRequestNumber(): number {
  return ++requestNumberCounter
}

/**
 * Случайное число в диапазоне
 */
export function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

/**
 * Случайный элемент массива
 */
export function randomItem<T>(items: T[]): T {
  return items[Math.floor(Math.random() * items.length)]!
}

/**
 * Случайная дата в будущем (от 1 до maxDays дней)
 */
export function randomFutureDate(maxDays = 14): string {
  const date = new Date()
  date.setDate(date.getDate() + randomInt(1, maxDays))
  return date.toISOString().split('T')[0]!
}

/**
 * Случайное время
 */
export function randomTime(): string {
  const hours = randomInt(8, 18).toString().padStart(2, '0')
  const minutes = randomItem(['00', '30'])
  return `${hours}:${minutes}`
}

/**
 * Форматирование цены (в копейках -> рубли)
 */
export function formatPrice(amount: number): string {
  return new Intl.NumberFormat('ru-RU').format(amount / 100)
}

// Данные для генерации реалистичных заявок
export const CITIES = [
  { id: 1, name: 'Москва', countryId: 1 },
  { id: 2, name: 'Санкт-Петербург', countryId: 1 },
  { id: 3, name: 'Казань', countryId: 1 },
  { id: 4, name: 'Нижний Новгород', countryId: 1 },
  { id: 5, name: 'Екатеринбург', countryId: 1 },
  { id: 6, name: 'Новосибирск', countryId: 1 },
  { id: 7, name: 'Ростов-на-Дону', countryId: 1 },
  { id: 8, name: 'Краснодар', countryId: 1 },
]

export const CARGO_DESCRIPTIONS = [
  'Мебель в разборе',
  'Бытовая техника',
  'Строительные материалы',
  'Продукты питания',
  'Оборудование',
  'Запчасти',
  'Одежда и текстиль',
  'Электроника',
]

export const ADDRESSES = [
  'ул. Ленина, д. 1',
  'пр. Мира, д. 25',
  'ул. Советская, д. 10',
  'пр. Победы, д. 50',
  'ул. Гагарина, д. 15',
  'ул. Кирова, д. 30',
]

export const CONTACT_NAMES = [
  'Иван Петров',
  'Алексей Смирнов',
  'Дмитрий Козлов',
  'Мария Иванова',
  'Елена Сидорова',
  'Андрей Волков',
]

export const PHONE_NUMBERS = [
  '+7 (999) 123-45-67',
  '+7 (999) 234-56-78',
  '+7 (999) 345-67-89',
  '+7 (999) 456-78-90',
  '+7 (999) 567-89-01',
]
