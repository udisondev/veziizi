import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { notificationsApi } from '@/api/notifications'
import type {
  Notification,
  NotificationPreferences,
  NotificationFilters,
  NotificationCategory,
} from '@/types/notification'
import { logger } from '@/utils/logger'

export const useNotificationsStore = defineStore('notifications', () => {
  // ===============================
  // State
  // ===============================
  const notifications = ref<Notification[]>([])
  const unreadCount = ref(0)
  const preferences = ref<NotificationPreferences | null>(null)
  const isLoading = ref(false)
  const isLoadingPreferences = ref(false)
  const error = ref<string | null>(null)

  // Polling
  const pollingInterval = ref<number | null>(null)
  const POLLING_INTERVAL_MS = 30000 // 30 секунд

  // ===============================
  // Computed
  // ===============================
  const hasUnread = computed(() => unreadCount.value > 0)

  const isTelegramConnected = computed(
    () => preferences.value?.telegram.connected ?? false
  )

  const isEmailConnected = computed(
    () => preferences.value?.email.connected ?? false
  )

  const isEmailVerified = computed(
    () => preferences.value?.email.verified ?? false
  )

  const recentNotifications = computed(() =>
    notifications.value.slice(0, 5)
  )

  // ===============================
  // Actions: Notifications
  // ===============================
  async function fetchNotifications(filters?: NotificationFilters): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      notifications.value = await notificationsApi.list({
        ...filters,
        limit: filters?.limit ?? 50,
      })
    } catch (e) {
      error.value = 'Не удалось загрузить уведомления'
      logger.error('Failed to fetch notifications', e)
    } finally {
      isLoading.value = false
    }
  }

  async function fetchRecentNotifications(): Promise<void> {
    try {
      const recent = await notificationsApi.list({ limit: 5 })
      // Merge с существующими, сохраняя уникальность
      const existingIds = new Set(notifications.value.map(n => n.id))
      const newNotifications = recent.filter(n => !existingIds.has(n.id))
      if (newNotifications.length > 0) {
        notifications.value = [...newNotifications, ...notifications.value]
      }
    } catch (e) {
      logger.error('Failed to fetch recent notifications', e)
    }
  }

  async function fetchUnreadCount(): Promise<void> {
    try {
      unreadCount.value = await notificationsApi.getUnreadCount()
    } catch (e) {
      logger.error('Failed to fetch unread count', e)
    }
  }

  async function markAsRead(id: string): Promise<void> {
    try {
      await notificationsApi.markAsRead([id])
      // Обновляем локально
      const notification = notifications.value.find(n => n.id === id)
      if (notification && !notification.is_read) {
        notification.is_read = true
        notification.read_at = new Date().toISOString()
        unreadCount.value = Math.max(0, unreadCount.value - 1)
      }
    } catch (e) {
      logger.error('Failed to mark notification as read', e)
    }
  }

  async function markAllAsRead(): Promise<void> {
    try {
      await notificationsApi.markAllAsRead()
      notifications.value.forEach(n => {
        if (!n.is_read) {
          n.is_read = true
          n.read_at = new Date().toISOString()
        }
      })
      unreadCount.value = 0
    } catch (e) {
      logger.error('Failed to mark all as read', e)
    }
  }

  // ===============================
  // Actions: Preferences
  // ===============================
  async function fetchPreferences(): Promise<void> {
    isLoadingPreferences.value = true
    try {
      preferences.value = await notificationsApi.getPreferences()
    } catch (e) {
      logger.error('Failed to fetch notification preferences', e)
    } finally {
      isLoadingPreferences.value = false
    }
  }

  async function updateCategorySetting(
    category: NotificationCategory,
    channel: 'in_app' | 'telegram' | 'email',
    enabled: boolean
  ): Promise<void> {
    if (!preferences.value) return

    const currentSettings = preferences.value.enabled_categories[category]
    const newSettings = {
      ...currentSettings,
      [channel]: enabled,
    }

    try {
      await notificationsApi.updatePreferences({
        [category]: newSettings,
      })
      // Обновляем локально
      preferences.value.enabled_categories[category] = newSettings
    } catch (e) {
      logger.error('Failed to update category setting', e)
      throw e
    }
  }

  // ===============================
  // Actions: Telegram
  // ===============================
  async function disconnectTelegram(): Promise<void> {
    try {
      await notificationsApi.disconnectTelegram()
      if (preferences.value) {
        preferences.value.telegram = {
          connected: false,
        }
      }
    } catch (e) {
      logger.error('Failed to disconnect telegram', e)
      throw e
    }
  }

  // ===============================
  // Actions: Email
  // ===============================
  async function setEmail(email: string): Promise<void> {
    try {
      await notificationsApi.setEmail(email)
      if (preferences.value) {
        preferences.value.email = {
          connected: true,
          email,
          verified: false,
          marketing_consent: false,
        }
      }
    } catch (e) {
      logger.error('Failed to set email', e)
      throw e
    }
  }

  async function disconnectEmail(): Promise<void> {
    try {
      await notificationsApi.disconnectEmail()
      if (preferences.value) {
        preferences.value.email = {
          connected: false,
          verified: false,
          marketing_consent: false,
        }
      }
    } catch (e) {
      logger.error('Failed to disconnect email', e)
      throw e
    }
  }

  async function setMarketingConsent(consent: boolean): Promise<void> {
    try {
      await notificationsApi.setMarketingConsent(consent)
      if (preferences.value) {
        preferences.value.email.marketing_consent = consent
      }
    } catch (e) {
      logger.error('Failed to set marketing consent', e)
      throw e
    }
  }

  async function resendVerification(): Promise<void> {
    try {
      await notificationsApi.resendVerification()
    } catch (e) {
      logger.error('Failed to resend verification', e)
      throw e
    }
  }

  // ===============================
  // Polling with Page Visibility API
  // ===============================
  let visibilityHandler: (() => void) | null = null

  function doPollingTick(): void {
    // Не полим если вкладка неактивна
    if (document.hidden) return
    fetchUnreadCount()
    fetchRecentNotifications()
  }

  function startPolling(): void {
    if (pollingInterval.value) return

    // Сразу загружаем
    doPollingTick()

    pollingInterval.value = window.setInterval(doPollingTick, POLLING_INTERVAL_MS)

    // Подписываемся на изменение видимости вкладки
    visibilityHandler = () => {
      if (!document.hidden) {
        // Вкладка стала видимой — сразу обновляем
        doPollingTick()
      }
    }
    document.addEventListener('visibilitychange', visibilityHandler)
  }

  function stopPolling(): void {
    if (pollingInterval.value) {
      window.clearInterval(pollingInterval.value)
      pollingInterval.value = null
    }
    if (visibilityHandler) {
      document.removeEventListener('visibilitychange', visibilityHandler)
      visibilityHandler = null
    }
  }

  // ===============================
  // Lifecycle
  // ===============================
  async function initialize(): Promise<void> {
    await Promise.all([
      fetchUnreadCount(),
      fetchRecentNotifications(),
    ])
    startPolling()
  }

  function cleanup(): void {
    stopPolling()
    notifications.value = []
    unreadCount.value = 0
    preferences.value = null
    error.value = null
  }

  return {
    // State
    notifications,
    unreadCount,
    preferences,
    isLoading,
    isLoadingPreferences,
    error,

    // Computed
    hasUnread,
    isTelegramConnected,
    isEmailConnected,
    isEmailVerified,
    recentNotifications,

    // Actions: Notifications
    fetchNotifications,
    fetchRecentNotifications,
    fetchUnreadCount,
    markAsRead,
    markAllAsRead,

    // Actions: Preferences
    fetchPreferences,
    updateCategorySetting,

    // Actions: Telegram
    disconnectTelegram,

    // Actions: Email
    setEmail,
    disconnectEmail,
    setMarketingConsent,
    resendVerification,

    // Lifecycle
    initialize,
    cleanup,
    startPolling,
    stopPolling,
  }
})
