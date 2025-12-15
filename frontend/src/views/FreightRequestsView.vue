<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import { freightRequestsApi } from '@/api/freightRequests'
import type { FreightRequestListItem, FreightRequestStatus } from '@/types/freightRequest'
import {
  freightRequestStatusLabels,
  cargoTypeLabels,
  bodyTypeLabels,
  currencyLabels,
} from '@/types/freightRequest'

const router = useRouter()
const auth = useAuthStore()
const { canCreateFreightRequest } = usePermissions()

const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters
type ViewMode = 'all' | 'my'
const viewMode = ref<ViewMode>('all')
const statusFilter = ref<FreightRequestStatus | ''>('')

const statusOptions: { value: FreightRequestStatus | '', label: string }[] = [
  { value: '', label: 'Все статусы' },
  { value: 'published', label: 'Опубликованы' },
  { value: 'selected', label: 'Выбран перевозчик' },
  { value: 'confirmed', label: 'Подтверждены' },
  { value: 'cancelled', label: 'Отменены' },
  { value: 'expired', label: 'Истекли' },
]

async function loadItems() {
  isLoading.value = true
  error.value = null

  try {
    const params: Parameters<typeof freightRequestsApi.list>[0] = {}

    if (viewMode.value === 'my' && auth.organizationId) {
      params.customer_org_id = auth.organizationId
    }

    if (statusFilter.value) {
      params.status = statusFilter.value
    }

    items.value = await freightRequestsApi.list(params)
  } catch (e) {
    error.value = 'Не удалось загрузить заявки'
    console.error(e)
  } finally {
    isLoading.value = false
  }
}

function goToDetail(id: string) {
  router.push(`/freight-requests/${id}`)
}

function goToCreate() {
  router.push('/freight-requests/new')
}

function formatPrice(amount?: number, currency?: string): string {
  if (!amount || !currency) return '—'
  const value = amount / 100
  const symbol = currencyLabels[currency as keyof typeof currencyLabels] || currency
  return `${value.toLocaleString('ru-RU')} ${symbol}`
}

function formatWeight(weight?: number): string {
  if (!weight) return '—'
  return `${weight.toLocaleString('ru-RU')} т`
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function formatBodyTypes(types?: string[]): string {
  if (!types || types.length === 0) return '—'
  return types
    .map(t => bodyTypeLabels[t as keyof typeof bodyTypeLabels] || t)
    .join(', ')
}

function getStatusColor(status: FreightRequestStatus): string {
  switch (status) {
    case 'published':
      return 'bg-green-100 text-green-800'
    case 'selected':
      return 'bg-yellow-100 text-yellow-800'
    case 'confirmed':
      return 'bg-blue-100 text-blue-800'
    case 'cancelled':
      return 'bg-red-100 text-red-800'
    case 'expired':
      return 'bg-gray-100 text-gray-800'
    default:
      return 'bg-gray-100 text-gray-800'
  }
}

function isExpiringSoon(expiresAt: string): boolean {
  const expires = new Date(expiresAt)
  const now = new Date()
  const diffDays = (expires.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  return diffDays > 0 && diffDays <= 3
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
        <h1 class="text-2xl font-bold text-gray-900 mb-4 sm:mb-0">Заявки на перевозку</h1>

        <button
          v-if="canCreateFreightRequest"
          @click="goToCreate"
          class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          + Новая заявка
        </button>
      </div>

      <!-- Filters -->
      <div class="bg-white shadow rounded-lg p-4 mb-6">
        <div class="flex flex-col sm:flex-row gap-4">
          <!-- View mode toggle -->
          <div class="flex rounded-md shadow-sm">
            <button
              @click="viewMode = 'all'"
              :class="[
                'px-4 py-2 text-sm font-medium rounded-l-md border',
                viewMode === 'all'
                  ? 'bg-blue-600 text-white border-blue-600'
                  : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              ]"
            >
              Все заявки
            </button>
            <button
              @click="viewMode = 'my'"
              :class="[
                'px-4 py-2 text-sm font-medium rounded-r-md border-t border-r border-b -ml-px',
                viewMode === 'my'
                  ? 'bg-blue-600 text-white border-blue-600'
                  : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              ]"
            >
              Мои заявки
            </button>
          </div>

          <!-- Status filter -->
          <select
            v-model="statusFilter"
            class="block w-full sm:w-48 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 text-sm"
          >
            <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
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
        <div class="text-gray-400 text-5xl mb-4">📦</div>
        <h3 class="text-lg font-medium text-gray-900 mb-2">Заявок пока нет</h3>
        <p class="text-gray-500 mb-4">
          {{ viewMode === 'my' ? 'Вы ещё не создали ни одной заявки' : 'Нет заявок с выбранными фильтрами' }}
        </p>
        <button
          v-if="canCreateFreightRequest"
          @click="goToCreate"
          class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
        >
          Создать заявку
        </button>
      </div>

      <!-- List -->
      <div v-else class="space-y-4">
        <div
          v-for="item in items"
          :key="item.id"
          @click="goToDetail(item.id)"
          class="bg-white shadow rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
        >
          <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <!-- Route -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <span :class="['inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium', getStatusColor(item.status)]">
                  {{ freightRequestStatusLabels[item.status] }}
                </span>
                <span v-if="item.status === 'published' && isExpiringSoon(item.expires_at)" class="text-xs text-orange-600">
                  ⏰ Истекает скоро
                </span>
              </div>

              <div class="text-lg font-medium text-gray-900 truncate">
                {{ item.origin_address || 'Не указан' }}
              </div>
              <div class="flex items-center text-gray-500 text-sm">
                <span class="mx-2">→</span>
              </div>
              <div class="text-lg font-medium text-gray-900 truncate">
                {{ item.destination_address || 'Не указан' }}
              </div>
            </div>

            <!-- Details -->
            <div class="flex flex-wrap gap-4 lg:gap-6 text-sm">
              <!-- Cargo -->
              <div class="min-w-24">
                <div class="text-gray-500">Груз</div>
                <div class="font-medium">
                  {{ item.cargo_type ? cargoTypeLabels[item.cargo_type] : '—' }}
                </div>
                <div class="text-gray-600">{{ formatWeight(item.cargo_weight) }}</div>
              </div>

              <!-- Vehicle -->
              <div class="min-w-24">
                <div class="text-gray-500">Кузов</div>
                <div class="font-medium truncate max-w-32">
                  {{ formatBodyTypes(item.body_types) }}
                </div>
              </div>

              <!-- Price -->
              <div class="min-w-24">
                <div class="text-gray-500">Ставка</div>
                <div class="font-medium text-green-600">
                  {{ formatPrice(item.price_amount, item.price_currency) }}
                </div>
              </div>

              <!-- Date -->
              <div class="min-w-24">
                <div class="text-gray-500">Создана</div>
                <div class="font-medium">{{ formatDate(item.created_at) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
  </div>
</template>
