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
      this.createNewOfferNotification(frId, CARRIERS[i].name, i)
    }
  }

  /**
   * Очистить store
   */
  clear(): void {
    this.items.clear()
  }
}

// Глобальный singleton для корректной работы при HMR
declare global {
  interface Window {
    __mockNotifications?: MockNotificationsStore
  }
}

if (!window.__mockNotifications) {
  window.__mockNotifications = new MockNotificationsStore()
}

export const mockNotifications = window.__mockNotifications
