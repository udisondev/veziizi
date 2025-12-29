<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useNotificationsStore } from '@/stores/notifications'
import { useAuthStore } from '@/stores/auth'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Separator } from '@/components/ui/separator'
import NotificationItem from './NotificationItem.vue'

import { Bell } from 'lucide-vue-next'

const router = useRouter()
const notificationsStore = useNotificationsStore()
const authStore = useAuthStore()

const isOpen = ref(false)

const recentNotifications = computed(() => notificationsStore.recentNotifications)
const unreadCount = computed(() => notificationsStore.unreadCount)
const hasUnread = computed(() => notificationsStore.hasUnread)

function handleNotificationClick(notification: { id: string; link?: string }) {
  isOpen.value = false
  notificationsStore.markAsRead(notification.id)
  if (notification.link) {
    router.push(notification.link)
  }
}

function goToAllNotifications() {
  isOpen.value = false
  router.push('/notifications')
}

function handleMarkAllRead() {
  notificationsStore.markAllAsRead()
}

onMounted(() => {
  if (authStore.isAuthenticated) {
    notificationsStore.initialize()
  }
})

onUnmounted(() => {
  notificationsStore.cleanup()
})
</script>

<template>
  <DropdownMenu v-model:open="isOpen">
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" class="relative">
        <Bell class="h-5 w-5" />
        <!-- Badge с количеством -->
        <span
          v-if="hasUnread"
          class="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-xs text-destructive-foreground"
        >
          {{ unreadCount > 99 ? '99+' : unreadCount }}
        </span>
      </Button>
    </DropdownMenuTrigger>

    <DropdownMenuContent align="end" class="w-80 p-0">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b">
        <span class="font-semibold">Уведомления</span>
        <Button
          v-if="hasUnread"
          variant="ghost"
          size="sm"
          class="text-xs h-7"
          @click="handleMarkAllRead"
        >
          Прочитать все
        </Button>
      </div>

      <!-- Notifications list -->
      <div class="max-h-96 overflow-y-auto">
        <template v-if="recentNotifications.length > 0">
          <NotificationItem
            v-for="notification in recentNotifications"
            :key="notification.id"
            :notification="notification"
            compact
            @click="handleNotificationClick(notification)"
          />
        </template>
        <div v-else class="py-8 text-center text-muted-foreground text-sm">
          Нет уведомлений
        </div>
      </div>

      <!-- Footer -->
      <Separator />
      <div class="p-2">
        <Button
          variant="ghost"
          class="w-full justify-center text-sm"
          @click="goToAllNotifications"
        >
          Все уведомления
        </Button>
      </div>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
