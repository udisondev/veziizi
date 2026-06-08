<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useBreakpoint } from '@/composables/useBreakpoint'
import { useLlamaPreference } from '@/composables/useLlamaPreference'
import { RouterView, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppHeader from '@/components/ui/AppHeader.vue'
import AuthHeader from '@/components/ui/AuthHeader.vue'
import AccountBlockedBanner from '@/components/AccountBlockedBanner.vue'
import DevUserSwitcher from '@/components/dev/DevUserSwitcher.vue'
import { Toaster } from '@/components/ui/toast'
import { devApi } from '@/api/dev'
import {
  TutorialOverlay,
  TutorialTooltip,
  FirstLoginHint,
  SandboxIndicator,
} from '@/components/tutorial'
import LlamaWalker from '@/components/shared/LlamaWalker.vue'

const route = useRoute()
const auth = useAuthStore()
const { isMobile } = useBreakpoint()
const { enabled: llamaEnabled } = useLlamaPreference()

const showHeader = computed(() => {
  // Don't show header on public pages and admin pages
  if (route.meta.public || route.meta.admin) return false
  // Don't show on inactive org pages
  if (route.meta.allowInactiveOrg) return false
  // Show only for authenticated users
  return auth.isAuthenticated
})

const showAuthHeader = computed(() => !!(route.meta.public && !route.meta.admin))

const showBanner = computed(() => auth.isAuthenticated && auth.isBlocked)

const isDevMode = ref(false)

onMounted(async () => {
  // Sandbox interceptor и loadProgress инициализируются в main.ts ДО монтирования app

  if (import.meta.env.VITE_DEV_TOOLS === 'true') {
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
  <div class="min-h-screen bg-background flex flex-col">
    <!-- Account Blocked Banner - Fixed at top, above all content -->
    <AccountBlockedBanner />

    <div class="flex-1" :class="{ 'pt-24': showBanner }">
      <AppHeader v-if="showHeader" />
      <AuthHeader v-else-if="showAuthHeader" />
      <RouterView :key="route.path" />
    </div>

    <LlamaWalker v-if="!isMobile && llamaEnabled" @close="llamaEnabled = false" />
    <DevUserSwitcher v-if="isDevMode" />
    <Toaster />

    <!-- Tutorial System -->
    <FirstLoginHint />
    <TutorialOverlay />
    <TutorialTooltip />
    <SandboxIndicator />
  </div>
</template>
