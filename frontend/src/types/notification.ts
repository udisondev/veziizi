// Типы уведомлений (синхронизируются с бэкендом)
export type NotificationType =
  | 'new_freight_request'
  | 'new_offer'
  | 'offer_selected'
  | 'offer_rejected'
  | 'offer_confirmed'
  | 'offer_declined'
  | 'offer_withdrawn'
  | 'freight_completed'
  | 'freight_cancelled'
  | 'review_received'
  | 'member_invited'
  | 'member_joined'
  | 'org_status_changed'

// Категории для фильтрации и настроек
export type NotificationCategory = 'freight_requests' | 'offers' | 'reviews' | 'organization'

// Каналы доставки
export type NotificationChannel = 'in_app' | 'telegram'

// Уведомление
export interface Notification {
  id: string
  member_id: string
  organization_id: string
  notification_type: NotificationType
  title: string
  body?: string
  link?: string
  entity_type?: 'freight_request' | 'organization' | 'member'
  entity_id?: string
  is_read: boolean
  read_at?: string
  created_at: string
}

// Настройки категории
export interface CategorySettings {
  in_app: boolean
  telegram: boolean
}

// Настройки всех категорий
export type EnabledCategories = {
  [key in NotificationCategory]: CategorySettings
}

// Статус Telegram подключения
export interface TelegramStatus {
  connected: boolean
  username?: string
  connected_at?: string
}

// Настройки уведомлений пользователя
export interface NotificationPreferences {
  member_id: string
  enabled_categories: EnabledCategories
  telegram: TelegramStatus
}

// Фильтры для списка
export interface NotificationFilters {
  category?: NotificationCategory
  is_read?: boolean
  limit?: number
  offset?: number
}

// Ответ на генерацию кода привязки Telegram
export interface TelegramLinkCodeResponse {
  code: string
  expires_in: number // секунды
  bot_url: string
}

// Labels для UI
export const notificationTypeLabels: Record<NotificationType, string> = {
  new_freight_request: 'Новая заявка',
  new_offer: 'Новое предложение',
  offer_selected: 'Предложение выбрано',
  offer_rejected: 'Предложение отклонено',
  offer_confirmed: 'Перевозка подтверждена',
  offer_declined: 'Предложение отклонено перевозчиком',
  offer_withdrawn: 'Предложение отозвано',
  freight_completed: 'Перевозка завершена',
  freight_cancelled: 'Перевозка отменена',
  review_received: 'Новый отзыв',
  member_invited: 'Приглашение',
  member_joined: 'Новый сотрудник',
  org_status_changed: 'Статус организации',
}

export const categoryLabels: Record<NotificationCategory, string> = {
  freight_requests: 'Заявки',
  offers: 'Предложения',
  reviews: 'Отзывы',
  organization: 'Организация',
}

export const categoryDescriptions: Record<NotificationCategory, string> = {
  freight_requests: 'Новые заявки на перевозку грузов',
  offers: 'Предложения на ваши заявки и статусы ваших предложений',
  reviews: 'Новые отзывы о вашей организации',
  organization: 'Изменения в организации: сотрудники, статус',
}

export const allCategories: NotificationCategory[] = ['freight_requests', 'offers', 'reviews', 'organization']

// Получить категорию по типу уведомления
export function getCategoryByType(type: NotificationType): NotificationCategory {
  switch (type) {
    case 'new_freight_request':
    case 'freight_completed':
    case 'freight_cancelled':
      return 'freight_requests'
    case 'new_offer':
    case 'offer_selected':
    case 'offer_rejected':
    case 'offer_confirmed':
    case 'offer_declined':
    case 'offer_withdrawn':
      return 'offers'
    case 'review_received':
      return 'reviews'
    case 'member_invited':
    case 'member_joined':
    case 'org_status_changed':
      return 'organization'
    default:
      return 'organization'
  }
}
