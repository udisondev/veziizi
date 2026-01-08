/**
 * Mock Handlers for Notifications
 */

import { registerHandler } from './index'
import { mockNotifications } from '@/sandbox/mockData/notifications'

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

  // Get preferences (заглушка для sandbox)
  registerHandler('GET', '/notifications/preferences', () => {
    return {
      data: {
        member_id: 'sandbox-member-1',
        enabled_categories: {
          freight_requests: { in_app: true, telegram: false },
          offers: { in_app: true, telegram: false },
          reviews: { in_app: true, telegram: false },
          organization: { in_app: true, telegram: false },
        },
        telegram: { connected: false },
      },
    }
  })

  // Update preferences (заглушка для sandbox)
  registerHandler('PATCH', '/notifications/preferences', () => {
    return { status: 204 }
  })
}
