/**
 * Mock Handlers for Notifications
 */

import { registerHandler } from './index'
import { mockNotifications, mockNotificationPreferences } from '@/sandbox/mockData/notifications'

export function notificationsHandlers(): void {
  // Get notifications list
  registerHandler('GET', '/notifications', (_params, _body, query) => {
    const limit = parseInt(query?.get('limit') || '50')
    const offset = parseInt(query?.get('offset') || '0')

    const notifications = mockNotifications.list(limit, offset)
    return { data: notifications }
  })

  // Get unread count
  registerHandler('GET', '/notifications/unread-count', () => {
    const unread = mockNotifications.getUnreadCount()
    return { data: { unread } }
  })

  // Mark notifications as read
  registerHandler('POST', '/notifications/read', (_params, body) => {
    const { notification_ids } = body as { notification_ids: string[] }
    mockNotifications.markAsRead(notification_ids)
    return { status: 204 }
  })

  // Mark all as read
  registerHandler('POST', '/notifications/read-all', () => {
    mockNotifications.markAllAsRead()
    return { status: 204 }
  })

  // Get preferences
  registerHandler('GET', '/notifications/preferences', () => {
    return { data: mockNotificationPreferences.get() }
  })

  // Update preferences
  registerHandler('PATCH', '/notifications/preferences', (_params, body) => {
    const { categories } = body as { categories: Record<string, unknown> }
    mockNotificationPreferences.updateCategories(categories)
    return { status: 204 }
  })

  // ===============================
  // Email handlers
  // ===============================

  // Set email for notifications
  registerHandler('POST', '/notifications/email', (_params, body) => {
    const { email } = body as { email: string }
    mockNotificationPreferences.setEmail(email)
    return { status: 204 }
  })

  // Disconnect email
  registerHandler('DELETE', '/notifications/email', () => {
    mockNotificationPreferences.disconnectEmail()
    return { status: 204 }
  })

  // Set marketing consent
  registerHandler('PATCH', '/notifications/email/marketing', (_params, body) => {
    const { consent } = body as { consent: boolean }
    mockNotificationPreferences.setMarketingConsent(consent)
    return { status: 204 }
  })

  // Resend verification email
  registerHandler('POST', '/notifications/email/resend-verification', () => {
    // В sandbox просто возвращаем успех
    return { status: 204 }
  })

  // Verify email by token
  registerHandler('POST', '/notifications/email/verify', (_params, body) => {
    const { token } = body as { token: string }

    // Тестовые токены для демонстрации разных состояний
    if (token === 'invalid-token') {
      return {
        status: 400,
        data: { error: 'invalid or expired token' },
      }
    }

    if (token === 'rate-limit-token') {
      return {
        status: 429,
        data: { error: 'too many requests' },
      }
    }

    // Для любого другого токена — успех + помечаем email как verified
    mockNotificationPreferences.verifyEmail()
    console.log('[Sandbox] Email verified with token:', token)

    return { status: 204 }
  })

  // ===============================
  // Telegram handlers
  // ===============================

  // Generate link code
  registerHandler('POST', '/notifications/telegram/link-code', () => {
    return {
      data: {
        code: 'DEMO123',
        expires_in: 300,
        bot_url: 'https://t.me/veziizi_bot?start=DEMO123',
      },
    }
  })

  // Disconnect Telegram
  registerHandler('DELETE', '/notifications/telegram', () => {
    mockNotificationPreferences.disconnectTelegram()
    return { status: 204 }
  })
}
