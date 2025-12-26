import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { subscriptionsApi } from '@/api/subscriptions'
import type {
  FreightSubscription,
  FreightSubscriptionCreate,
  FreightSubscriptionUpdate,
} from '@/types/subscription'
import { MAX_SUBSCRIPTIONS_PER_MEMBER } from '@/types/subscription'

export const useSubscriptionsStore = defineStore('subscriptions', () => {
  // ===============================
  // State
  // ===============================
  const subscriptions = ref<FreightSubscription[]>([])
  const isLoading = ref(false)
  const isSaving = ref(false)
  const error = ref<string | null>(null)

  // ===============================
  // Computed
  // ===============================
  const activeSubscriptions = computed(() =>
    subscriptions.value.filter(s => s.is_active)
  )

  const inactiveSubscriptions = computed(() =>
    subscriptions.value.filter(s => !s.is_active)
  )

  const canCreateMore = computed(() =>
    subscriptions.value.length < MAX_SUBSCRIPTIONS_PER_MEMBER
  )

  const subscriptionsCount = computed(() => subscriptions.value.length)

  const activeCount = computed(() => activeSubscriptions.value.length)

  // ===============================
  // Actions
  // ===============================
  async function fetchSubscriptions(): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      subscriptions.value = await subscriptionsApi.list()
    } catch (e) {
      error.value = 'Не удалось загрузить подписки'
      console.error('Failed to fetch subscriptions:', e)
    } finally {
      isLoading.value = false
    }
  }

  async function getSubscription(id: string): Promise<FreightSubscription | null> {
    try {
      return await subscriptionsApi.get(id)
    } catch (e) {
      console.error('Failed to get subscription:', e)
      return null
    }
  }

  async function createSubscription(data: FreightSubscriptionCreate): Promise<FreightSubscription | null> {
    if (!canCreateMore.value) {
      error.value = `Достигнут лимит подписок (${MAX_SUBSCRIPTIONS_PER_MEMBER})`
      return null
    }

    isSaving.value = true
    error.value = null
    try {
      const subscription = await subscriptionsApi.create(data)
      subscriptions.value.push(subscription)
      return subscription
    } catch (e) {
      error.value = 'Не удалось создать подписку'
      console.error('Failed to create subscription:', e)
      return null
    } finally {
      isSaving.value = false
    }
  }

  async function updateSubscription(
    id: string,
    data: FreightSubscriptionUpdate
  ): Promise<FreightSubscription | null> {
    isSaving.value = true
    error.value = null
    try {
      const subscription = await subscriptionsApi.update(id, data)
      const index = subscriptions.value.findIndex(s => s.id === id)
      if (index !== -1) {
        subscriptions.value[index] = subscription
      }
      return subscription
    } catch (e) {
      error.value = 'Не удалось обновить подписку'
      console.error('Failed to update subscription:', e)
      return null
    } finally {
      isSaving.value = false
    }
  }

  async function deleteSubscription(id: string): Promise<boolean> {
    isSaving.value = true
    error.value = null
    try {
      await subscriptionsApi.delete(id)
      subscriptions.value = subscriptions.value.filter(s => s.id !== id)
      return true
    } catch (e) {
      error.value = 'Не удалось удалить подписку'
      console.error('Failed to delete subscription:', e)
      return false
    } finally {
      isSaving.value = false
    }
  }

  async function toggleActive(id: string): Promise<boolean> {
    const subscription = subscriptions.value.find(s => s.id === id)
    if (!subscription) return false

    try {
      const updated = await subscriptionsApi.setActive(id, !subscription.is_active)
      const index = subscriptions.value.findIndex(s => s.id === id)
      if (index !== -1) {
        subscriptions.value[index] = updated
      }
      return true
    } catch (e) {
      console.error('Failed to toggle subscription active:', e)
      return false
    }
  }

  function clearError(): void {
    error.value = null
  }

  function cleanup(): void {
    subscriptions.value = []
    error.value = null
    isLoading.value = false
    isSaving.value = false
  }

  return {
    // State
    subscriptions,
    isLoading,
    isSaving,
    error,

    // Computed
    activeSubscriptions,
    inactiveSubscriptions,
    canCreateMore,
    subscriptionsCount,
    activeCount,

    // Actions
    fetchSubscriptions,
    getSubscription,
    createSubscription,
    updateSubscription,
    deleteSubscription,
    toggleActive,
    clearError,
    cleanup,
  }
})
