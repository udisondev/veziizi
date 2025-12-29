import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import { getFingerprint } from '@/composables/useFingerprint'
import { logger } from '@/utils/logger'
import type {
  MemberRole,
  OrganizationBrief,
  LoginRequest,
} from '@/types/api'

export const useAuthStore = defineStore('auth', () => {
  const memberId = ref<string | null>(null)
  const organizationId = ref<string | null>(null)
  const role = ref<MemberRole | null>(null)
  const email = ref<string | null>(null)
  const name = ref<string | null>(null)
  const phone = ref<string | null>(null)
  const telegramId = ref<number | null>(null)
  const organization = ref<OrganizationBrief | null>(null)
  const isLoading = ref(false)
  const isInitialized = ref(false)

  const isAuthenticated = computed(() => memberId.value !== null)

  async function login(credentials: LoginRequest): Promise<void> {
    isLoading.value = true
    try {
      const fingerprint = await getFingerprint()
      await authApi.login(credentials, fingerprint)
      await fetchMe()
    } finally {
      isLoading.value = false
    }
  }

  async function logout(): Promise<void> {
    isLoading.value = true
    try {
      await authApi.logout()
      clearAuth()
    } finally {
      isLoading.value = false
    }
  }

  async function fetchMe(): Promise<void> {
    try {
      const data = await authApi.me()
      memberId.value = data.member_id
      organizationId.value = data.organization_id
      role.value = data.role
      email.value = data.email
      name.value = data.name
      phone.value = data.phone ?? null
      telegramId.value = data.telegram_id ?? null
      organization.value = data.organization ?? null
    } catch (e) {
      logger.error('Failed to fetch user', e)
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

  function clearAuth(): void {
    memberId.value = null
    organizationId.value = null
    role.value = null
    email.value = null
    name.value = null
    phone.value = null
    telegramId.value = null
    organization.value = null
  }

  return {
    // State
    memberId,
    organizationId,
    role,
    email,
    name,
    phone,
    telegramId,
    organization,
    isLoading,
    isInitialized,

    // Computed
    isAuthenticated,

    // Actions
    login,
    logout,
    fetchMe,
    initialize,
    clearAuth,
  }
})
