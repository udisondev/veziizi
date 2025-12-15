import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { adminApi } from '@/api/admin'
import type { AdminLoginRequest } from '@/types/admin'

export const useAdminStore = defineStore('admin', () => {
  const adminId = ref<string | null>(localStorage.getItem('adminId'))
  const email = ref<string | null>(localStorage.getItem('adminEmail'))
  const name = ref<string | null>(localStorage.getItem('adminName'))
  const isLoading = ref(false)

  const isAuthenticated = computed(() => !!adminId.value)

  async function login(credentials: AdminLoginRequest) {
    isLoading.value = true
    try {
      const response = await adminApi.login(credentials)
      adminId.value = response.admin_id
      email.value = response.email
      name.value = response.name
      localStorage.setItem('adminId', response.admin_id)
      localStorage.setItem('adminEmail', response.email)
      localStorage.setItem('adminName', response.name)
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

  function clearAuth() {
    adminId.value = null
    email.value = null
    name.value = null
    localStorage.removeItem('adminId')
    localStorage.removeItem('adminEmail')
    localStorage.removeItem('adminName')
  }

  return {
    adminId,
    email,
    name,
    isLoading,
    isAuthenticated,
    login,
    logout,
    clearAuth,
  }
})
