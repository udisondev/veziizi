import { api } from './client'
import type {
  Notification,
  NotificationPreferences,
  NotificationFilters,
  EnabledCategories,
  TelegramLinkCodeResponse,
} from '@/types/notification'

export const notificationsApi = {
  // ===============================
  // In-app уведомления
  // ===============================

  // Получить список уведомлений
  async list(filters?: NotificationFilters): Promise<Notification[]> {
    const params = new URLSearchParams()

    if (filters?.category) {
      params.set('category', filters.category)
    }
    if (filters?.is_read !== undefined) {
      params.set('is_read', String(filters.is_read))
    }
    if (filters?.limit) {
      params.set('limit', String(filters.limit))
    }
    if (filters?.offset) {
      params.set('offset', String(filters.offset))
    }

    const query = params.toString()
    const result = await api.get<Notification[] | null>(`/notifications${query ? `?${query}` : ''}`)
    return result ?? []
  },

  // Получить количество непрочитанных
  async getUnreadCount(): Promise<number> {
    const result = await api.get<{ unread: number }>('/notifications/unread-count')
    return result.unread
  },

  // Пометить как прочитанные
  async markAsRead(ids: string[]): Promise<void> {
    await api.post('/notifications/read', { notification_ids: ids })
  },

  // Пометить все как прочитанные
  async markAllAsRead(): Promise<void> {
    await api.post('/notifications/read-all')
  },

  // ===============================
  // Настройки
  // ===============================

  // Получить настройки
  async getPreferences(): Promise<NotificationPreferences> {
    return api.get('/notifications/preferences')
  },

  // Обновить настройки категорий
  async updatePreferences(categories: Partial<EnabledCategories>): Promise<void> {
    await api.patch('/notifications/preferences', { categories })
  },

  // ===============================
  // Telegram (привязка через бота)
  // ===============================

  // Сгенерировать код привязки
  async generateLinkCode(): Promise<TelegramLinkCodeResponse> {
    return api.post('/notifications/telegram/link-code')
  },

  // Отключить Telegram
  async disconnectTelegram(): Promise<void> {
    await api.delete('/notifications/telegram')
  },

  // ===============================
  // Email
  // ===============================

  // Установить email для уведомлений (требует верификации)
  async setEmail(email: string): Promise<void> {
    await api.post('/notifications/email', { email })
  },

  // Отключить email уведомления
  async disconnectEmail(): Promise<void> {
    await api.delete('/notifications/email')
  },

  // Установить согласие на маркетинговые рассылки
  async setMarketingConsent(consent: boolean): Promise<void> {
    await api.patch('/notifications/email/marketing', { consent })
  },

  // Повторно отправить письмо с подтверждением
  async resendVerification(): Promise<void> {
    await api.post('/notifications/email/resend-verification')
  },

  // Подтвердить email по токену (публичный endpoint)
  async verifyEmail(token: string): Promise<void> {
    await api.post('/notifications/email/verify', { token })
  },
}
