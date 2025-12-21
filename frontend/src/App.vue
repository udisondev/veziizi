<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppHeader from '@/components/ui/AppHeader.vue'
import DevUserSwitcher from '@/components/dev/DevUserSwitcher.vue'
import { Toaster } from '@/components/ui/toast'
import { devApi } from '@/api/dev'

const route = useRoute()
const auth = useAuthStore()

const showHeader = computed(() => {
  // Don't show header on public pages and admin pages
  if (route.meta.public || route.meta.admin) return false
  // Don't show on inactive org pages
  if (route.meta.allowInactiveOrg) return false
  // Show only for authenticated users
  return auth.isAuthenticated
})

const isDevMode = ref(false)

onMounted(async () => {
  if (import.meta.env.DEV) {
    try {
      const status = await devApi.getStatus()
      isDevMode.value = status.enabled
    } catch {
      isDevMode.value = false
    }
  }
})
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader v-if="showHeader" />
    <RouterView :key="route.path" />
    <DevUserSwitcher v-if="isDevMode" />
    <Toaster />
  </div>
</template>
