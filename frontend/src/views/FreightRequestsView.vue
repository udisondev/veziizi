<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import { freightRequestsApi } from '@/api/freightRequests'
import type { FreightRequestListItem, FreightRequestStatus, OwnershipFilter, Country } from '@/types/freightRequest'
import {
  freightRequestStatusLabels,
  cargoTypeLabels,
  bodyTypeLabels,
  currencyLabels,
  ownershipOptions,
  statusOptions,
  countryOptions,
  countryLabels,
} from '@/types/freightRequest'

const router = useRouter()
const auth = useAuthStore()
const { canCreateFreightRequest } = usePermissions()

const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters (applied state)
const ownershipFilter = ref<OwnershipFilter>('all')
const statusFilter = ref<FreightRequestStatus | ''>('')
const orgNameFilter = ref('')
const orgINNFilter = ref('')
const orgCountryFilter = ref<Country | ''>('')

// Temp filters for modal
const tempOwnership = ref<OwnershipFilter>('all')
const tempStatus = ref<FreightRequestStatus | ''>('')
const tempOrgName = ref('')
const tempOrgINN = ref('')
const tempOrgCountry = ref<Country | ''>('')

const showFilterModal = ref(false)

// Computed
const hasActiveFilters = computed(() =>
  ownershipFilter.value !== 'all' ||
  statusFilter.value !== '' ||
  orgNameFilter.value !== '' ||
  orgINNFilter.value !== '' ||
  orgCountryFilter.value !== ''
)

// Modal functions
function openFilterModal() {
  tempOwnership.value = ownershipFilter.value
  tempStatus.value = statusFilter.value
  tempOrgName.value = orgNameFilter.value
  tempOrgINN.value = orgINNFilter.value
  tempOrgCountry.value = orgCountryFilter.value
  showFilterModal.value = true
}

function applyFilters() {
  ownershipFilter.value = tempOwnership.value
  statusFilter.value = tempStatus.value
  orgNameFilter.value = tempOrgName.value
  orgINNFilter.value = tempOrgINN.value
  orgCountryFilter.value = tempOrgCountry.value
  showFilterModal.value = false
}

function clearFilters() {
  tempOwnership.value = 'all'
  tempStatus.value = ''
  tempOrgName.value = ''
  tempOrgINN.value = ''
  tempOrgCountry.value = ''
}

function resetAllFilters() {
  ownershipFilter.value = 'all'
  statusFilter.value = ''
  orgNameFilter.value = ''
  orgINNFilter.value = ''
  orgCountryFilter.value = ''
}

function closeFilterModal() {
  showFilterModal.value = false
}

// Load data with filters
async function loadItems() {
  isLoading.value = true
  error.value = null

  try {
    const params: Parameters<typeof freightRequestsApi.list>[0] = {}

    // Ownership filter
    if (ownershipFilter.value === 'my_org' && auth.organizationId) {
      params.customer_org_id = auth.organizationId
    } else if (ownershipFilter.value === 'my' && auth.memberId) {
      params.member_id = auth.memberId
    }

    if (statusFilter.value) params.status = statusFilter.value
    if (orgNameFilter.value) params.org_name = orgNameFilter.value
    if (orgINNFilter.value) params.org_inn = orgINNFilter.value
    if (orgCountryFilter.value) params.org_country = orgCountryFilter.value

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

// Watch filters and reload
watch(
  [ownershipFilter, statusFilter, orgNameFilter, orgINNFilter, orgCountryFilter],
  () => loadItems()
)

onMounted(() => {
  loadItems()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-900 mb-4 sm:mb-0">Заявки на перевозку</h1>

      <div class="flex gap-2">
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

        <button
          v-if="canCreateFreightRequest"
          @click="goToCreate"
          class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          + Новая заявка
        </button>
      </div>
    </div>

    <!-- Active filters indicator -->
    <div v-if="hasActiveFilters" class="bg-blue-50 border border-blue-200 rounded-lg p-3 mb-6 flex items-center justify-between">
      <div class="text-sm text-blue-700 flex flex-wrap gap-x-2 gap-y-1">
        <span v-if="ownershipFilter !== 'all'">
          {{ ownershipOptions.find(o => o.value === ownershipFilter)?.label }}
        </span>
        <span v-if="statusFilter">
          <span v-if="ownershipFilter !== 'all'">, </span>
          Статус: {{ statusOptions.find(o => o.value === statusFilter)?.label }}
        </span>
        <span v-if="orgNameFilter">
          <span v-if="ownershipFilter !== 'all' || statusFilter">, </span>
          Организация: "{{ orgNameFilter }}"
        </span>
        <span v-if="orgINNFilter">
          <span v-if="ownershipFilter !== 'all' || statusFilter || orgNameFilter">, </span>
          ИНН: "{{ orgINNFilter }}"
        </span>
        <span v-if="orgCountryFilter">
          <span v-if="ownershipFilter !== 'all' || statusFilter || orgNameFilter || orgINNFilter">, </span>
          Страна: {{ countryLabels[orgCountryFilter] }}
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
      <div class="text-gray-400 text-5xl mb-4">📦</div>
      <h3 class="text-lg font-medium text-gray-900 mb-2">Заявок пока нет</h3>
      <p class="text-gray-500 mb-4">
        {{ hasActiveFilters ? 'Нет заявок по заданным фильтрам' : 'Заявок пока нет' }}
      </p>
      <button
        v-if="canCreateFreightRequest && !hasActiveFilters"
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

            <!-- Organization info -->
            <div v-if="item.customer_org_name" class="text-sm text-gray-500 mt-2">
              {{ item.customer_org_name }}
              <span v-if="item.customer_org_country" class="text-gray-400">
                ({{ countryLabels[item.customer_org_country as Country] || item.customer_org_country }})
              </span>
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

    <!-- Filter Modal -->
    <div v-if="showFilterModal" class="fixed inset-0 bg-black/25 flex items-center justify-center p-4 z-50" @click="closeFilterModal">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
        <h2 class="text-xl font-bold mb-4">Фильтры</h2>

        <div class="space-y-4">
          <!-- Ownership -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Принадлежность
            </label>
            <select
              v-model="tempOwnership"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="opt in ownershipOptions" :key="opt.value" :value="opt.value">
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
              <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>

          <!-- Organization name -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Название организации
            </label>
            <input
              v-model="tempOrgName"
              type="text"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Поиск по названию"
            />
          </div>

          <!-- INN -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              ИНН
            </label>
            <input
              v-model="tempOrgINN"
              type="text"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Поиск по ИНН"
            />
          </div>

          <!-- Country -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Страна
            </label>
            <select
              v-model="tempOrgCountry"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="opt in countryOptions" :key="opt.value" :value="opt.value">
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
