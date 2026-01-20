/**
 * Mock Notifications Store
 * Mock данные для уведомлений в sandbox режиме
 */

import type { Notification } from '@/types/notification'
import { generateId } from './generators'

// Контрагенты для симуляции (синхронизировано с offers.ts)
const CARRIERS = [
  { id: 'carrier-1', name: 'ТрансЛогистик' },
  { id: 'carrier-2', name: 'СпецГруз' },
  { id: 'carrier-3', name: 'МегаФура' },
  { id: 'carrier-4', name: 'ЭкспрессДоставка' },
]

class MockNotificationsStore {
  private items: Map<string, Notification> = new Map()

  /**
   * Получить список уведомлений
   */
  list(limit: number = 50, offset: number = 0): Notification[] {
    const all = Array.from(this.items.values()).sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    )
    return all.slice(offset, offset + limit)
  }

  /**
   * Получить количество непрочитанных
   */
  getUnreadCount(): number {
    let count = 0
    for (const notification of this.items.values()) {
      if (!notification.is_read) count++
    }
    return count
  }

  /**
   * Создать уведомление о новом предложении
   */
  createNewOfferNotification(frId: string, carrierName: string, offerIndex: number): Notification {
    const id = generateId('notification')

    const notification: Notification = {
      id,
      member_id: 'sandbox-member-1',
      organization_id: 'sandbox-org-1',
      notification_type: 'new_offer',
      title: `Новое предложение от ${carrierName}`,
      body: `Перевозчик ${carrierName} сделал предложение на вашу заявку`,
      link: `/freight-requests/${frId}`,
      entity_type: 'freight_request',
      entity_id: frId,
      is_read: false,
      created_at: new Date(Date.now() - offerIndex * 60000).toISOString(),
    }

    this.items.set(id, notification)

    return notification
  }

  /**
   * Пометить как прочитанное
   */
  markAsRead(ids: string[]): void {
    for (const id of ids) {
      const notification = this.items.get(id)
      if (notification && !notification.is_read) {
        notification.is_read = true
        notification.read_at = new Date().toISOString()
      }
    }
  }

  /**
   * Пометить все как прочитанные
   */
  markAllAsRead(): void {
    for (const notification of this.items.values()) {
      if (!notification.is_read) {
        notification.is_read = true
        notification.read_at = new Date().toISOString()
      }
    }
  }

  /**
   * Генерация уведомлений для офферов
   * Вызывается из freightRequests.seedWithOffers()
   */
  seedOffersNotifications(frId: string, offersCount: number): void {
    for (let i = 0; i < offersCount && i < CARRIERS.length; i++) {
      this.createNewOfferNotification(frId, CARRIERS[i]!.name, i)
    }
  }

  /**
   * Очистить store
   */
  clear(): void {
    this.items.clear()
  }
}

/**
 * Mock Notification Preferences Store
 */
class MockNotificationPreferencesStore {
  private preferences = {
    member_id: 'sandbox-member-1',
    enabled_categories: {
      freight_requests: { in_app: true, telegram: false, email: true },
      offers: { in_app: true, telegram: false, email: true },
      reviews: { in_app: true, telegram: false, email: false },
      organization: { in_app: true, telegram: false, email: true },
    },
    telegram: {
      connected: false,
      username: undefined as string | undefined,
      connected_at: undefined as string | undefined,
    },
    email: {
      connected: true,
      email: 'demo@veziizi.local' as string | undefined,
      verified: true,
      verified_at: '2026-01-15T10:00:00Z' as string | undefined,
      marketing_consent: false,
    },
  }

  get() {
    return { ...this.preferences }
  }

  updateCategories(categories: Record<string, unknown>) {
    this.preferences.enabled_categories = {
      ...this.preferences.enabled_categories,
      ...categories,
    } as typeof this.preferences.enabled_categories
  }

  setEmail(email: string) {
    this.preferences.email = {
      connected: true,
      email,
      verified: false,
      verified_at: undefined,
      marketing_consent: false,
    }
  }

  disconnectEmail() {
    this.preferences.email = {
      connected: false,
      email: undefined,
      verified: false,
      verified_at: undefined,
      marketing_consent: false,
    }
  }

  setMarketingConsent(consent: boolean) {
    this.preferences.email.marketing_consent = consent
  }

  verifyEmail() {
    if (this.preferences.email.connected) {
      this.preferences.email.verified = true
      this.preferences.email.verified_at = new Date().toISOString()
    }
  }

  connectTelegram(username: string) {
    this.preferences.telegram = {
      connected: true,
      username,
      connected_at: new Date().toISOString(),
    }
  }

  disconnectTelegram() {
    this.preferences.telegram = {
      connected: false,
      username: undefined,
      connected_at: undefined,
    }
  }
}

// Глобальный singleton для корректной работы при HMR
declare global {
  interface Window {
    __mockNotifications?: MockNotificationsStore
    __mockNotificationPreferences?: MockNotificationPreferencesStore
  }
}

if (!window.__mockNotifications) {
  window.__mockNotifications = new MockNotificationsStore()
}

if (!window.__mockNotificationPreferences) {
  window.__mockNotificationPreferences = new MockNotificationPreferencesStore()
}

export const mockNotifications = window.__mockNotifications
export const mockNotificationPreferences = window.__mockNotificationPreferences
