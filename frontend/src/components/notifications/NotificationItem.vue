<script setup lang="ts">
import { computed } from 'vue'
import type { Notification } from '@/types/notification'
import { getCategoryByType } from '@/types/notification'
import { cn } from '@/lib/utils'

import {
  Truck,
  Package,
  Star,
  Users,
  Bell,
} from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  notification: Notification
  compact?: boolean
  dark?: boolean
}>(), {
  compact: false,
  dark: false,
})

const emit = defineEmits<{
  click: []
}>()

// Иконка по типу уведомления
const icon = computed(() => {
  const category = getCategoryByType(props.notification.notification_type)
  const categoryIconMap: Record<string, any> = {
    freight_requests: Truck,
    offers: Package,
    reviews: Star,
    organization: Users,
  }
  return categoryIconMap[category] || Bell
})

// Цвет фона иконки
const iconBgClass = computed(() => {
  const category = getCategoryByType(props.notification.notification_type)
  const categoryColorMap: Record<string, string> = {
    freight_requests: 'bg-warning/10 text-warning',
    offers: 'bg-accent text-accent-foreground',
    reviews: 'bg-warning/10 text-warning',
    organization: 'bg-accent text-accent-foreground',
  }
  return categoryColorMap[category] || 'bg-muted text-muted-foreground'
})

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'только что'
  if (diffMins < 60) return `${diffMins} мин назад`
  if (diffHours < 24) return `${diffHours} ч назад`
  if (diffDays < 7) return `${diffDays} дн назад`

  return date.toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
  })
}
</script>

<template>
  <div
    :class="cn(
      'flex gap-3 cursor-pointer transition-colors',
      compact
        ? dark ? 'px-4 py-3 hover:bg-slate-700' : 'px-4 py-3 hover:bg-muted/50'
        : dark ? 'p-4 hover:bg-slate-700 rounded-lg' : 'p-4 hover:bg-muted/30 rounded-lg',
      !notification.is_read && (dark ? 'bg-slate-700/50' : 'bg-primary/5')
    )"
    @click="emit('click')"
  >
    <!-- Icon -->
    <div
      :class="cn(
        'flex-shrink-0 rounded-full flex items-center justify-center',
        compact ? 'h-8 w-8' : 'h-10 w-10',
        iconBgClass
      )"
    >
      <component :is="icon" :class="compact ? 'h-4 w-4' : 'h-5 w-5'" />
    </div>

    <!-- Content -->
    <div class="flex-1 min-w-0">
      <div class="flex items-start justify-between gap-2">
        <p
          :class="cn(
            'font-medium truncate',
            compact ? 'text-sm' : 'text-base',
            dark ? 'text-slate-100' : (!notification.is_read && 'text-foreground')
          )"
        >
          {{ notification.title }}
        </p>
        <!-- Unread indicator -->
        <div
          v-if="!notification.is_read"
          class="flex-shrink-0 h-2 w-2 rounded-full bg-primary mt-1.5"
        />
      </div>
      <p
        v-if="notification.body"
        :class="cn(
          compact ? 'text-xs line-clamp-1' : 'text-sm line-clamp-2 mt-0.5',
          dark ? 'text-slate-400' : 'text-muted-foreground'
        )"
      >
        {{ notification.body }}
      </p>
      <p :class="['text-xs mt-1', dark ? 'text-slate-500' : 'text-muted-foreground']">
        {{ formatDate(notification.created_at) }}
      </p>
    </div>
  </div>
</template>
