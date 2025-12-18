<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { EventHistoryItem, EventHistoryPage } from '@/api/history'
import { eventTypeLabels, formatEventDetails, isAutomaticEvent } from '@/types/eventHistory'

const props = defineProps<{
  loadFn: (limit: number, offset: number) => Promise<EventHistoryPage>
}>()

const items = ref<EventHistoryItem[]>([])
const total = ref(0)
const isLoading = ref(false)
const error = ref('')
const page = ref(1)
const limit = 20

async function loadData() {
  isLoading.value = true
  error.value = ''
  try {
    const offset = (page.value - 1) * limit
    const result = await props.loadFn(limit, offset)
    items.value = result.items
    total.value = result.total
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки истории'
    console.error('Failed to load history:', e)
  } finally {
    isLoading.value = false
  }
}

const totalPages = computed(() => Math.ceil(total.value / limit))

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function getEventLabel(eventType: string): string {
  return eventTypeLabels[eventType] || eventType
}

function getEventDetails(event: EventHistoryItem): string {
  return formatEventDetails(event.event_type, event.data)
}

function isAutomatic(eventType: string): boolean {
  return isAutomaticEvent(eventType)
}

onMounted(loadData)
watch(page, loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- Loading -->
    <div v-if="isLoading" class="text-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
      <p class="mt-2 text-gray-600">Загрузка...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
      {{ error }}
      <button @click="loadData" class="ml-2 underline hover:no-underline">Повторить</button>
    </div>

    <!-- Empty -->
    <div v-else-if="items.length === 0" class="text-center py-12 text-gray-500">
      История пуста
    </div>

    <!-- Events List -->
    <template v-else>
      <div class="space-y-3">
        <div
          v-for="event in items"
          :key="event.id"
          class="bg-white border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
        >
          <!-- Header: Event type and date -->
          <div class="flex items-start justify-between mb-2">
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-900">
                {{ getEventLabel(event.event_type) }}
              </span>
              <span
                v-if="isAutomatic(event.event_type)"
                class="px-2 py-0.5 text-xs font-medium rounded-full bg-gray-100 text-gray-600"
              >
                Автоматически
              </span>
            </div>
            <span class="text-sm text-gray-500 whitespace-nowrap ml-4">
              {{ formatDateTime(event.occurred_at) }}
            </span>
          </div>

          <!-- Actor info -->
          <div v-if="event.actor" class="text-sm text-gray-600 mb-2">
            <span class="text-gray-500">Инициатор:</span>
            <span class="ml-1 font-medium">{{ event.actor.name }}</span>
            <span v-if="event.actor.email" class="text-gray-400 ml-1">({{ event.actor.email }})</span>
          </div>

          <!-- Event details -->
          <div
            v-if="getEventDetails(event)"
            class="text-sm text-gray-600 bg-gray-50 rounded px-3 py-2"
          >
            {{ getEventDetails(event) }}
          </div>

          <!-- Version badge -->
          <div class="mt-2 text-xs text-gray-400">
            Версия: {{ event.version }}
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-center gap-4 pt-4">
        <button
          @click="page--"
          :disabled="page <= 1"
          class="px-4 py-2 text-sm bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          Назад
        </button>
        <span class="text-sm text-gray-600">
          Страница {{ page }} из {{ totalPages }}
        </span>
        <button
          @click="page++"
          :disabled="page >= totalPages"
          class="px-4 py-2 text-sm bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          Вперёд
        </button>
      </div>

      <!-- Total count -->
      <div class="text-center text-sm text-gray-500">
        Всего событий: {{ total }}
      </div>
    </template>
  </div>
</template>
