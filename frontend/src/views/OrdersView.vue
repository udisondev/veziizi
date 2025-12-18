<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ordersApi } from '@/api/orders'
import type { OrderListItem, OrderStatus, ViewMode } from '@/types/order'
import {
  orderStatusLabels,
  orderStatusColors,
  orderStatusOptions,
  viewModeOptions,
  viewModeLabels,
} from '@/types/order'

const router = useRouter()
const auth = useAuthStore()

const items = ref<OrderListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters (applied state)
const viewMode = ref<ViewMode>('all')
const statusFilter = ref<OrderStatus | ''>('')

// Temp filters for modal
const tempViewMode = ref<ViewMode>('all')
const tempStatus = ref<OrderStatus | ''>('')
const showFilterModal = ref(false)

// Computed
const hasActiveFilters = computed(() =>
  viewMode.value !== 'all' || statusFilter.value !== ''
)

async function loadItems() {
  isLoading.value = true
  error.value = null

  try {
    const params: Parameters<typeof ordersApi.list>[0] = {}

    if (viewMode.value === 'as_customer' && auth.organizationId) {
      params.customer_org_id = auth.organizationId
    } else if (viewMode.value === 'as_carrier' && auth.organizationId) {
      params.carrier_org_id = auth.organizationId
    }

    if (statusFilter.value) {
      params.status = statusFilter.value
    }

    items.value = await ordersApi.list(params)
  } catch (e) {
    error.value = 'Не удалось загрузить заказы'
    console.error(e)
  } finally {
    isLoading.value = false
  }
}

function goToDetail(id: string) {
  router.push(`/orders/${id}`)
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function getRole(item: OrderListItem): string {
  if (item.customer_org_id === auth.organizationId) return 'Заказчик'
  if (item.carrier_org_id === auth.organizationId) return 'Перевозчик'
  return ''
}

// Modal functions
function openFilterModal() {
  tempViewMode.value = viewMode.value
  tempStatus.value = statusFilter.value
  showFilterModal.value = true
}

function applyFilters() {
  viewMode.value = tempViewMode.value
  statusFilter.value = tempStatus.value
  showFilterModal.value = false
}

function clearFilters() {
  tempViewMode.value = 'all'
  tempStatus.value = ''
}

function resetAllFilters() {
  viewMode.value = 'all'
  statusFilter.value = ''
}

function closeFilterModal() {
  showFilterModal.value = false
}

// Watch filters
watch([viewMode, statusFilter], () => {
  loadItems()
})

onMounted(() => {
  loadItems()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-900 mb-4 sm:mb-0">Заказы</h1>

      <button
        @click="openFilterModal"
        class="px-4 py-2 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors flex items-center gap-2"
      >
        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
          />
        </svg>
        Фильтры
        <span
          v-if="hasActiveFilters"
          class="w-2 h-2 bg-blue-600 rounded-full"
        ></span>
      </button>
    </div>

    <!-- Active filters indicator -->
    <div v-if="hasActiveFilters" class="bg-blue-50 border border-blue-200 rounded-lg p-3 mb-6 flex items-center justify-between">
      <div class="text-sm text-blue-700 flex flex-wrap gap-x-2 gap-y-1">
        <span v-if="viewMode !== 'all'">
          {{ viewModeLabels[viewMode] }}
        </span>
        <span v-if="statusFilter">
          <span v-if="viewMode !== 'all'">, </span>
          Статус: {{ orderStatusOptions.find(o => o.value === statusFilter)?.label }}
        </span>
      </div>
      <button
        @click="resetAllFilters"
        class="text-blue-600 hover:text-blue-800 text-sm underline whitespace-nowrap ml-2"
      >
        Сбросить
      </button>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
      {{ error }}
      <button @click="loadItems" class="ml-2 underline">Повторить</button>
    </div>

    <!-- Empty state -->
    <div v-else-if="items.length === 0" class="bg-white shadow rounded-lg p-12 text-center">
      <div class="text-gray-400 text-5xl mb-4">📋</div>
      <h3 class="text-lg font-medium text-gray-900 mb-2">Заказов пока нет</h3>
      <p class="text-gray-500">
        Заказы появятся после подтверждения офферов на заявки
      </p>
    </div>

    <!-- List -->
    <div v-else class="space-y-4">
      <div
        v-for="item in items"
        :key="item.id"
        @click="goToDetail(item.id)"
        class="bg-white shadow rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
      >
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div class="flex-1">
            <div class="flex items-center gap-2 mb-2">
              <span :class="['inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium', orderStatusColors[item.status]]">
                {{ orderStatusLabels[item.status] }}
              </span>
              <span v-if="getRole(item)" class="text-xs text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
                {{ getRole(item) }}
              </span>
            </div>
            <div class="text-sm text-gray-600">
              Заказ #{{ item.order_number }}
            </div>
          </div>

          <div class="text-sm text-gray-500">
            {{ formatDate(item.created_at) }}
          </div>
        </div>
      </div>
    </div>

    <!-- Filter Modal -->
    <div v-if="showFilterModal" class="fixed inset-0 bg-black/25 flex items-center justify-center p-4 z-50" @click="closeFilterModal">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
        <h2 class="text-xl font-bold mb-4">Фильтры</h2>

        <div class="space-y-4">
          <!-- View Mode -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Отображение
            </label>
            <select
              v-model="tempViewMode"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="opt in viewModeOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>

          <!-- Status -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Статус
            </label>
            <select
              v-model="tempStatus"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="opt in orderStatusOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>
        </div>

        <div class="flex flex-col gap-2 mt-6">
          <button
            @click="applyFilters"
            class="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Применить
          </button>
          <div class="flex gap-2">
            <button
              @click="clearFilters"
              class="flex-1 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
            >
              Очистить
            </button>
            <button
              @click="closeFilterModal"
              class="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
            >
              Отмена
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
