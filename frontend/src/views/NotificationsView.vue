<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useNotificationsStore } from '@/stores/notifications'
import type { Notification, NotificationCategory } from '@/types/notification'
import { categoryLabels, allCategories, getNotificationLink } from '@/types/notification'

// UI Components
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { TabsSlider } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select-field'

// Shared Components
import {
  PageHeader,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
} from '@/components/shared'

// Components
import NotificationItem from '@/components/notifications/NotificationItem.vue'

// Icons
import { Bell, Settings } from 'lucide-vue-next'

const router = useRouter()
const notificationsStore = useNotificationsStore()

// Filters
const readFilter = ref<'all' | 'unread' | 'read'>('all')
const categoryFilter = ref<NotificationCategory | 'all'>('all')

const readFilterItems = [
  { value: 'all', label: 'Все' },
  { value: 'unread', label: 'Непрочитанные' },
  { value: 'read', label: 'Прочитанные' },
]

const categoryOptions = [
  { value: 'all', label: 'Все категории' },
  ...allCategories.map(cat => ({ value: cat, label: categoryLabels[cat] })),
]

const isLoading = computed(() => notificationsStore.isLoading)
const error = computed(() => notificationsStore.error)

const filteredNotifications = computed(() => {
  let result = notificationsStore.notifications

  if (readFilter.value === 'unread') {
    result = result.filter(n => !n.is_read)
  } else if (readFilter.value === 'read') {
    result = result.filter(n => n.is_read)
  }

  return result
})

const hasActiveFilters = computed(() =>
  readFilter.value !== 'all' || categoryFilter.value !== 'all'
)

async function loadNotifications() {
  await notificationsStore.fetchNotifications({
    category: categoryFilter.value !== 'all' ? categoryFilter.value : undefined,
    is_read: readFilter.value === 'unread' ? false :
             readFilter.value === 'read' ? true : undefined,
  })
}

function handleNotificationClick(notification: Notification) {
  notificationsStore.markAsRead(notification.id)
  const link = getNotificationLink(notification)
  if (link) router.push(link)
}

function goToSettings() {
  router.push('/notifications/settings')
}

function resetAllFilters() {
  readFilter.value = 'all'
  categoryFilter.value = 'all'
}

watch([readFilter, categoryFilter], () => {
  loadNotifications()
})

onMounted(() => {
  loadNotifications()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Уведомления" class="mb-6">
      <template #actions>
        <Button variant="outline" @click="goToSettings">
          <Settings class="mr-2 h-4 w-4" />
          Настройки
        </Button>
      </template>
    </PageHeader>

    <!-- Filters -->
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center gap-3">
      <TabsSlider v-model="readFilter" :items="readFilterItems" stretch class="sm:hidden" />
      <TabsSlider v-model="readFilter" :items="readFilterItems" class="hidden sm:block" />

      <div class="flex items-center gap-3">
        <div class="w-full sm:w-56">
          <SelectField
            v-model="categoryFilter"
            :options="categoryOptions"
            sheet-label="Категория"
          />
        </div>
        <Button
          v-if="hasActiveFilters"
          variant="ghost"
          size="sm"
          class="text-muted-foreground shrink-0"
          @click="resetAllFilters"
        >
          Сбросить
        </Button>
      </div>
    </div>

    <!-- Mark all as read -->
    <div v-if="notificationsStore.hasUnread" class="mb-4 flex justify-end">
      <Button variant="outline" size="sm" @click="notificationsStore.markAllAsRead()">
        Прочитать все
      </Button>
    </div>

    <!-- Loading -->
    <LoadingSpinner v-if="isLoading" text="Загрузка уведомлений..." />

    <!-- Error -->
    <ErrorBanner
      v-else-if="error"
      :message="error"
      @retry="loadNotifications"
    />

    <!-- Empty state -->
    <EmptyState
      v-else-if="filteredNotifications.length === 0"
      :icon="Bell"
      title="Нет уведомлений"
      :description="hasActiveFilters ? 'Попробуйте изменить фильтры' : 'Когда появятся новые уведомления, вы увидите их здесь'"
    />

    <!-- Notifications list -->
    <div v-else class="space-y-2">
      <Card
        v-for="notification in filteredNotifications"
        :key="notification.id"
        class="overflow-hidden"
        interactive
      >
        <NotificationItem
          :notification="notification"
          @click="handleNotificationClick(notification)"
        />
      </Card>
    </div>
  </div>
</template>
