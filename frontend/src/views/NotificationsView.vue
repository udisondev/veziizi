<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useNotificationsStore } from '@/stores/notifications'
import type { NotificationCategory } from '@/types/notification'
import { categoryLabels, allCategories } from '@/types/notification'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

// Shared Components
import {
  PageHeader,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
  FilterSheet,
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
const showFilters = ref(false)

// Temp filters for sheet
const tempReadFilter = ref<'all' | 'unread' | 'read'>('all')
const tempCategory = ref<NotificationCategory | 'all'>('all')

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

const activeFiltersCount = computed(() => {
  let count = 0
  if (readFilter.value !== 'all') count++
  if (categoryFilter.value !== 'all') count++
  return count
})

async function loadNotifications() {
  await notificationsStore.fetchNotifications({
    category: categoryFilter.value !== 'all' ? categoryFilter.value : undefined,
    is_read: readFilter.value === 'unread' ? false :
             readFilter.value === 'read' ? true : undefined,
  })
}

function handleNotificationClick(notification: { id: string; link?: string }) {
  notificationsStore.markAsRead(notification.id)
  if (notification.link) {
    router.push(notification.link)
  }
}

function goToSettings() {
  router.push('/notifications/settings')
}

// Filter sheet handlers
function openFilters() {
  tempReadFilter.value = readFilter.value
  tempCategory.value = categoryFilter.value
  showFilters.value = true
}

function applyFilters() {
  readFilter.value = tempReadFilter.value
  categoryFilter.value = tempCategory.value
  showFilters.value = false
}

function resetFilters() {
  tempReadFilter.value = 'all'
  tempCategory.value = 'all'
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
        <div class="flex gap-2">
          <Button variant="outline" @click="goToSettings">
            <Settings class="mr-2 h-4 w-4" />
            Настройки
          </Button>

          <FilterSheet
            v-model:open="showFilters"
            :active-filters-count="activeFiltersCount"
            description="Фильтрация уведомлений"
            @open="openFilters"
            @apply="applyFilters"
            @reset="resetFilters"
          >
            <!-- Read filter -->
            <div class="space-y-2">
              <Label>Статус</Label>
              <Tabs v-model="tempReadFilter" class="w-full">
                <TabsList class="grid w-full grid-cols-3">
                  <TabsTrigger value="all">Все</TabsTrigger>
                  <TabsTrigger value="unread">Непрочитанные</TabsTrigger>
                  <TabsTrigger value="read">Прочитанные</TabsTrigger>
                </TabsList>
              </Tabs>
            </div>

            <!-- Category filter -->
            <div class="space-y-2">
              <Label>Категория</Label>
              <Select v-model="tempCategory">
                <SelectTrigger>
                  <SelectValue placeholder="Все категории" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все категории</SelectItem>
                  <SelectItem
                    v-for="cat in allCategories"
                    :key="cat"
                    :value="cat"
                  >
                    {{ categoryLabels[cat] }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </FilterSheet>
        </div>
      </template>
    </PageHeader>

    <!-- Active filters indicator -->
    <Card v-if="hasActiveFilters" class="mb-6 border-primary/20 bg-primary/5">
      <CardContent class="flex items-center justify-between py-3">
        <div class="text-sm text-primary">
          Активные фильтры: {{ activeFiltersCount }}
        </div>
        <Button variant="ghost" size="sm" @click="resetAllFilters">
          Сбросить
        </Button>
      </CardContent>
    </Card>

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
      >
        <NotificationItem
          :notification="notification"
          @click="handleNotificationClick(notification)"
        />
      </Card>
    </div>
  </div>
</template>
