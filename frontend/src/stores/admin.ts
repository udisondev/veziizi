import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { adminApi } from '@/api/admin'
import { logger } from '@/utils/logger'
import type { AdminLoginRequest } from '@/types/admin'

export const useAdminStore = defineStore('admin', () => {
  const adminId = ref<string | null>(null)
  const email = ref<string | null>(null)
  const name = ref<string | null>(null)
  const isLoading = ref(false)
  const isInitialized = ref(false)

  const isAuthenticated = computed(() => !!adminId.value)

  async function login(credentials: AdminLoginRequest) {
    isLoading.value = true
    try {
      const response = await adminApi.login(credentials)
      adminId.value = response.admin_id
      email.value = response.email
      name.value = response.name
    } finally {
      isLoading.value = false
    }
  }

  async function logout() {
    try {
      await adminApi.logout()
    } finally {
      clearAuth()
    }
  }

  async function fetchMe(): Promise<void> {
    try {
      const data = await adminApi.me()
      adminId.value = data.admin_id
      email.value = data.email
      name.value = data.name
    } catch (e) {
      logger.error('Failed to fetch admin', e)
      clearAuth()
    }
  }

  async function initialize(): Promise<void> {
    if (isInitialized.value) return
    isLoading.value = true
    try {
      await fetchMe()
    } finally {
      isLoading.value = false
      isInitialized.value = true
    }
  }

  function clearAuth() {
    adminId.value = null
    email.value = null
    name.value = null
  }

  return {
    adminId,
    email,
    name,
    isLoading,
    isInitialized,
    isAuthenticated,
    login,
    logout,
    fetchMe,
    initialize,
    clearAuth,
  }
})
