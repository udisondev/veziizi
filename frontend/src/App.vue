<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppHeader from '@/components/ui/AppHeader.vue'

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
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <AppHeader v-if="showHeader" />
    <RouterView />
  </div>
</template>
