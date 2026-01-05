<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import AppHeader from '@/components/ui/AppHeader.vue'
import DevUserSwitcher from '@/components/dev/DevUserSwitcher.vue'
import { Toaster } from '@/components/ui/toast'
import { devApi } from '@/api/dev'
import { initSandboxInterceptor } from '@/sandbox/api/interceptor'
import {
  TutorialOverlay,
  TutorialTooltip,
  FirstLoginHint,
  SandboxIndicator,
} from '@/components/tutorial'

const route = useRoute()
const auth = useAuthStore()
const onboarding = useOnboardingStore()

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
  // Initialize sandbox API interceptor
  initSandboxInterceptor()

  // Load onboarding progress from localStorage
  onboarding.loadProgress()

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

    <!-- Tutorial System -->
    <FirstLoginHint />
    <TutorialOverlay />
    <TutorialTooltip />
    <SandboxIndicator />
  </div>
</template>
