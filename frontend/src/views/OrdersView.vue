<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ordersApi } from '@/api/orders'
import type { OrderListItem, OrderStatus } from '@/types/order'
import {
  orderStatusLabels,
  orderStatusColors,
  orderStatusOptions,
} from '@/types/order'

const router = useRouter()
const auth = useAuthStore()

const items = ref<OrderListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters
type ViewMode = 'all' | 'as_customer' | 'as_carrier'
const viewMode = ref<ViewMode>('all')
const statusFilter = ref<OrderStatus | ''>('')

const viewModeOptions = [
  { value: 'all' as const, label: 'Все заказы' },
  { value: 'as_customer' as const, label: 'Как заказчик' },
  { value: 'as_carrier' as const, label: 'Как перевозчик' },
]

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
      <h1 class="text-2xl font-bold text-gray-900 mb-4 sm:mb-0">Заказы</h1>
    </div>

    <!-- Filters -->
    <div class="bg-white shadow rounded-lg p-4 mb-6">
      <div class="flex flex-col sm:flex-row gap-4">
        <!-- View mode toggle -->
        <div class="flex rounded-md shadow-sm">
          <button
            v-for="(opt, idx) in viewModeOptions"
            :key="opt.value"
            @click="viewMode = opt.value"
            :class="[
              'px-4 py-2 text-sm font-medium border',
              idx === 0 ? 'rounded-l-md' : '',
              idx === viewModeOptions.length - 1 ? 'rounded-r-md' : '',
              idx > 0 ? '-ml-px' : '',
              viewMode === opt.value
                ? 'bg-blue-600 text-white border-blue-600'
                : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
            ]"
          >
            {{ opt.label }}
          </button>
        </div>

        <!-- Status filter -->
        <select
          v-model="statusFilter"
          class="block w-full sm:w-48 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 text-sm"
        >
          <option v-for="opt in orderStatusOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
      </div>
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
              Заказ #{{ item.id.slice(0, 8) }}
            </div>
          </div>

          <div class="text-sm text-gray-500">
            {{ formatDate(item.created_at) }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
