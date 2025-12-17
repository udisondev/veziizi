<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { offersApi, type MyOfferListItem } from '@/api/offers'
import type { OfferStatus } from '@/types/freightRequest'
import {
  offerStatusLabels,
  offerStatusColors,
  offerStatusOptions,
  currencyLabels,
  type Currency,
} from '@/types/freightRequest'

const router = useRouter()

const items = ref<MyOfferListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters (applied state)
const statusFilter = ref<OfferStatus | ''>('')

// Temp filters for modal
const tempStatus = ref<OfferStatus | ''>('')
const showFilterModal = ref(false)

// Computed
const hasActiveFilters = computed(() => statusFilter.value !== '')

// Modal functions
function openFilterModal() {
  tempStatus.value = statusFilter.value
  showFilterModal.value = true
}

function applyFilters() {
  statusFilter.value = tempStatus.value
  showFilterModal.value = false
}

function clearFilters() {
  tempStatus.value = ''
}

function resetAllFilters() {
  statusFilter.value = ''
}

function closeFilterModal() {
  showFilterModal.value = false
}

// Action state
const actionLoading = ref<string | null>(null)
const actionError = ref<string | null>(null)

// Confirm modal
const showConfirmModal = ref(false)
const confirmAction = ref<{ type: 'withdraw' | 'confirm' | 'decline'; item: MyOfferListItem } | null>(null)
const declineReason = ref('')

async function loadItems() {
  isLoading.value = true
  error.value = null

  try {
    const params = statusFilter.value ? { status: statusFilter.value } : undefined
    items.value = await offersApi.listMy(params)
  } catch (e) {
    error.value = 'Не удалось загрузить офферы'
    console.error(e)
  } finally {
    isLoading.value = false
  }
}

function goToFreightRequest(frId: string) {
  router.push(`/freight-requests/${frId}`)
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function formatPrice(amount?: number, currency?: string): string {
  if (!amount || !currency) return '—'
  const formatted = (amount / 100).toLocaleString('ru-RU', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  })
  const symbol = currencyLabels[currency as Currency] || currency
  return `${formatted} ${symbol}`
}

function formatWeight(weight?: number): string {
  if (!weight) return '—'
  return `${weight.toLocaleString('ru-RU')} кг`
}

function formatRoute(item: MyOfferListItem): string {
  const origin = item.origin_address || '?'
  const dest = item.destination_address || '?'
  return `${origin} → ${dest}`
}

// Action handlers
function canWithdraw(status: OfferStatus): boolean {
  return status === 'pending'
}

function canConfirm(status: OfferStatus): boolean {
  return status === 'selected'
}

function canDecline(status: OfferStatus): boolean {
  return status === 'selected'
}

function openConfirmModal(type: 'withdraw' | 'confirm' | 'decline', item: MyOfferListItem) {
  confirmAction.value = { type, item }
  declineReason.value = ''
  showConfirmModal.value = true
}

function closeConfirmModal() {
  showConfirmModal.value = false
  confirmAction.value = null
  declineReason.value = ''
}

const confirmModalTitle = computed(() => {
  if (!confirmAction.value) return ''
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'Отозвать оффер?'
    case 'confirm':
      return 'Подтвердить оффер?'
    case 'decline':
      return 'Отказаться от оффера?'
    default:
      return ''
  }
})

const confirmModalDescription = computed(() => {
  if (!confirmAction.value) return ''
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'Ваш оффер будет отозван и больше не будет виден заказчику.'
    case 'confirm':
      return 'После подтверждения будет создан заказ и вы станете перевозчиком.'
    case 'decline':
      return 'Вы отказываетесь от выбранного оффера. Заказчик сможет выбрать другого перевозчика.'
    default:
      return ''
  }
})

const confirmModalButtonText = computed(() => {
  if (!confirmAction.value) return ''
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'Отозвать'
    case 'confirm':
      return 'Подтвердить'
    case 'decline':
      return 'Отказаться'
    default:
      return ''
  }
})

const confirmModalButtonClass = computed(() => {
  if (!confirmAction.value) return ''
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'bg-gray-600 hover:bg-gray-700'
    case 'confirm':
      return 'bg-green-600 hover:bg-green-700'
    case 'decline':
      return 'bg-red-600 hover:bg-red-700'
    default:
      return 'bg-blue-600 hover:bg-blue-700'
  }
})

async function executeAction() {
  if (!confirmAction.value) return

  const { type, item } = confirmAction.value
  actionLoading.value = item.id
  actionError.value = null

  try {
    switch (type) {
      case 'withdraw':
        await offersApi.withdraw(item.freight_request_id, item.id)
        break
      case 'confirm':
        await offersApi.confirm(item.freight_request_id, item.id)
        break
      case 'decline':
        await offersApi.decline(item.freight_request_id, item.id, declineReason.value || undefined)
        break
    }
    closeConfirmModal()
    await loadItems()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Не удалось выполнить действие'
    console.error(e)
  } finally {
    actionLoading.value = null
  }
}

// Watch filters
watch(statusFilter, () => {
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
      <h1 class="text-xl sm:text-2xl font-bold text-gray-900 mb-4 sm:mb-0">Предложения</h1>

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
      <div class="text-sm text-blue-700">
        Статус: {{ offerStatusOptions.find(o => o.value === statusFilter)?.label }}
      </div>
      <button
        @click="resetAllFilters"
        class="text-blue-600 hover:text-blue-800 text-sm underline whitespace-nowrap ml-2"
      >
        Сбросить
      </button>
    </div>

    <!-- Action error -->
    <div v-if="actionError" class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 text-red-700">
      {{ actionError }}
      <button @click="actionError = null" class="ml-2 underline">Закрыть</button>
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
      <div class="text-gray-400 text-5xl mb-4">📝</div>
      <h3 class="text-lg font-medium text-gray-900 mb-2">Предложений пока нет</h3>
      <p class="text-gray-500 mb-4">
        Вы ещё не делали предложений на заявки
      </p>
      <router-link
        to="/"
        class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
      >
        Найти заявки
      </router-link>
    </div>

    <!-- List -->
    <div v-else class="space-y-4">
      <div
        v-for="item in items"
        :key="item.id"
        class="bg-white shadow rounded-lg p-4 hover:shadow-md transition-shadow"
      >
        <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <!-- Main info -->
          <div class="flex-1 cursor-pointer" @click="goToFreightRequest(item.freight_request_id)">
            <!-- Status -->
            <div class="flex items-center gap-2 mb-2">
              <span :class="['inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium', offerStatusColors[item.status]]">
                {{ offerStatusLabels[item.status] }}
              </span>
            </div>

            <!-- Route -->
            <div class="text-sm text-gray-900 font-medium mb-1">
              {{ formatRoute(item) }}
            </div>

            <!-- Details -->
            <div class="flex flex-wrap gap-4 text-sm text-gray-500">
              <span>Вес: {{ formatWeight(item.cargo_weight) }}</span>
              <span>Ставка: {{ formatPrice(item.price_amount, item.price_currency) }}</span>
              <span>{{ formatDate(item.created_at) }}</span>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex gap-2 flex-shrink-0">
            <!-- Withdraw (pending) -->
            <button
              v-if="canWithdraw(item.status)"
              @click.stop="openConfirmModal('withdraw', item)"
              :disabled="actionLoading === item.id"
              class="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md disabled:opacity-50"
            >
              Отозвать
            </button>

            <!-- Confirm (selected) -->
            <button
              v-if="canConfirm(item.status)"
              @click.stop="openConfirmModal('confirm', item)"
              :disabled="actionLoading === item.id"
              class="px-3 py-1.5 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md disabled:opacity-50"
            >
              Подтвердить
            </button>

            <!-- Decline (selected) -->
            <button
              v-if="canDecline(item.status)"
              @click.stop="openConfirmModal('decline', item)"
              :disabled="actionLoading === item.id"
              class="px-3 py-1.5 text-sm font-medium text-red-700 bg-red-100 hover:bg-red-200 rounded-md disabled:opacity-50"
            >
              Отказаться
            </button>

            <!-- View -->
            <button
              @click.stop="goToFreightRequest(item.freight_request_id)"
              class="px-3 py-1.5 text-sm font-medium text-blue-700 bg-blue-100 hover:bg-blue-200 rounded-md"
            >
              К заявке
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Confirm Modal -->
    <Teleport to="body">
      <div
        v-if="showConfirmModal"
        class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"
        @click.self="closeConfirmModal"
      >
        <div class="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
          <h3 class="text-lg font-medium text-gray-900 mb-2">
            {{ confirmModalTitle }}
          </h3>
          <p class="text-sm text-gray-500 mb-4">
            {{ confirmModalDescription }}
          </p>

          <!-- Decline reason -->
          <div v-if="confirmAction?.type === 'decline'" class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Причина отказа (необязательно)
            </label>
            <textarea
              v-model="declineReason"
              rows="2"
              class="w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 text-sm"
              placeholder="Укажите причину..."
            ></textarea>
          </div>

          <!-- Action error in modal -->
          <div v-if="actionError" class="bg-red-50 border border-red-200 rounded p-3 mb-4 text-sm text-red-700">
            {{ actionError }}
          </div>

          <div class="flex justify-end gap-3">
            <button
              @click="closeConfirmModal"
              :disabled="actionLoading !== null"
              class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md disabled:opacity-50"
            >
              Отмена
            </button>
            <button
              @click="executeAction"
              :disabled="actionLoading !== null"
              :class="['px-4 py-2 text-sm font-medium text-white rounded-md disabled:opacity-50', confirmModalButtonClass]"
            >
              <span v-if="actionLoading" class="flex items-center gap-2">
                <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                Выполнение...
              </span>
              <span v-else>{{ confirmModalButtonText }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Filter Modal -->
    <div v-if="showFilterModal" class="fixed inset-0 bg-black/25 flex items-center justify-center p-4 z-50" @click="closeFilterModal">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
        <h2 class="text-xl font-bold mb-4">Фильтры</h2>

        <div class="space-y-4">
          <!-- Status -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Статус
            </label>
            <select
              v-model="tempStatus"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="opt in offerStatusOptions" :key="opt.value" :value="opt.value">
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
