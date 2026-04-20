<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Button } from '@/components/ui/button'
import type { DisplayableHistoryItem, DisplayableHistoryPage } from '@/api/history'
import { isAutomaticEvent } from '@/types/eventHistory'
import { logger } from '@/utils/logger'

const props = defineProps<{
  loadFn: (limit: number, offset: number) => Promise<DisplayableHistoryPage>
}>()

const items = ref<DisplayableHistoryItem[]>([])
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
    logger.error('Failed to load history', e)
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

function isAutomatic(eventType: string): boolean {
  return isAutomaticEvent(eventType)
}

function getSeverityClass(severity?: string): string {
  switch (severity) {
    case 'success':
      return 'border-l-success'
    case 'warning':
      return 'border-l-warning'
    case 'error':
      return 'border-l-destructive'
    default:
      return 'border-l-primary'
  }
}

onMounted(loadData)
watch(page, loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- Loading -->
    <div v-if="isLoading" class="text-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      <p class="mt-2 text-muted-foreground">Загрузка...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-destructive">
      {{ error }}
      <button @click="loadData" class="ml-2 underline hover:no-underline text-sm">Повторить</button>
    </div>

    <!-- Empty -->
    <div v-else-if="items.length === 0" class="text-center py-12 text-muted-foreground">
      История пуста
    </div>

    <!-- Events List -->
    <template v-else>
      <div class="space-y-3">
        <div
          v-for="event in items"
          :key="event.id"
          class="bg-white border border-border rounded-lg p-4 hover:border-border/80 transition-colors border-l-4"
          :class="getSeverityClass(event.display.severity)"
        >
          <!-- Header: Title and date -->
          <div class="flex items-start justify-between mb-2 gap-2">
            <div class="flex items-center gap-2 min-w-0 flex-1">
              <span class="font-medium text-foreground break-words">
                {{ event.display.title }}
              </span>
              <span
                v-if="isAutomatic(event.event_type)"
                class="px-2 py-0.5 text-xs font-medium rounded-full bg-muted text-muted-foreground"
              >
                Автоматически
              </span>
            </div>
            <span class="text-sm text-muted-foreground whitespace-nowrap ml-4">
              {{ formatDateTime(event.occurred_at) }}
            </span>
          </div>

          <!-- Description -->
          <p class="text-sm text-muted-foreground mb-3 break-words">
            {{ event.display.description }}
          </p>

          <!-- Actor info -->
          <div v-if="event.actor" class="text-sm text-muted-foreground mb-3">
            <span>Инициатор:</span>
            <span class="ml-1 font-medium text-foreground">{{ event.actor.name }}</span>
            <span v-if="event.actor.email" class="ml-1">({{ event.actor.email }})</span>
          </div>

          <!-- Fields -->
          <div
            v-if="event.display.fields && event.display.fields.length > 0"
            class="bg-muted rounded-lg p-3 mb-3"
          >
            <div class="grid grid-cols-2 gap-2">
              <template v-for="field in event.display.fields" :key="field.label">
                <div class="text-sm text-muted-foreground">{{ field.label }}</div>
                <div class="text-sm text-foreground font-medium break-words">{{ field.value }}</div>
              </template>
            </div>
          </div>

          <!-- Diffs -->
          <div
            v-if="event.display.diffs && event.display.diffs.length > 0"
            class="space-y-3"
          >
            <div
              v-for="diff in event.display.diffs"
              :key="diff.label"
              class="text-sm"
            >
              <div class="text-muted-foreground mb-1">{{ diff.label }}:</div>
              <div class="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2 pl-2">
                <span class="text-destructive line-through break-words">{{ diff.old_value }}</span>
                <span class="text-muted-foreground hidden sm:inline">&rarr;</span>
                <span class="text-muted-foreground sm:hidden">&darr;</span>
                <span class="text-success font-medium break-words">{{ diff.new_value }}</span>
              </div>
            </div>
          </div>

          <!-- Version badge -->
          <div class="mt-3 text-xs text-muted-foreground">
            Версия: {{ event.version }}
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-center gap-4 pt-4">
        <Button
          variant="outline"
          size="sm"
          :disabled="page <= 1"
          @click="page--"
        >
          Назад
        </Button>
        <span class="text-sm text-muted-foreground">
          Страница {{ page }} из {{ totalPages }}
        </span>
        <Button
          variant="outline"
          size="sm"
          :disabled="page >= totalPages"
          @click="page++"
        >
          Вперёд
        </Button>
      </div>

      <!-- Total count -->
      <div class="text-center text-sm text-muted-foreground">
        Всего событий: {{ total }}
      </div>
    </template>
  </div>
</template>
