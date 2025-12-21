<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { freightRequestsApi } from '@/api/freightRequests'
import { ordersApi } from '@/api/orders'
import { membersApi, type MemberProfile } from '@/api/members'
import { historyApi } from '@/api/history'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import type { OrderListItem } from '@/types/order'
import LeafletMap from '@/components/freight-request/shared/LeafletMap.vue'
import EventHistory from '@/components/EventHistory.vue'
import type {
  FreightRequest,
  Offer,
  MakeOfferRequest,
  Currency,
  VatType,
  PaymentMethod,
} from '@/types/freightRequest'
import {
  freightRequestStatusLabels,
  cargoTypeLabels,
  bodyTypeLabels,
  loadingTypeLabels,
  currencyLabels,
  vatTypeLabels,
  paymentMethodLabels,
  paymentTermsLabels,
  adrClassLabels,
  offerStatusLabels,
  offerStatusColors,
  currencyOptions,
  vatTypeOptions,
  paymentMethodOptions,
} from '@/types/freightRequest'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const permissions = usePermissions()

// State
const freightRequest = ref<FreightRequest | null>(null)
const offers = ref<Offer[]>([])
const creatorProfile = ref<MemberProfile | null>(null)
const linkedOrder = ref<OrderListItem | null>(null)
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)

// Tabs
type TabType = 'details' | 'history'
const currentTab = ref<TabType>('details')

// History loader
function loadFreightRequestHistory(limit: number, offset: number) {
  const id = route.params.id as string
  return historyApi.getFreightRequestHistory(id, { limit, offset })
}

// Check if user can view history
const canViewHistory = computed(() => {
  if (!freightRequest.value) return false
  // Only owner/admin of the customer organization can view history
  if (freightRequest.value.customer_org_id !== auth.organizationId) return false
  return auth.role === 'owner' || auth.role === 'administrator'
})

// Modals
const showMakeOfferModal = ref(false)
const showCancelModal = ref(false)
const cancelReason = ref('')

const showRejectModal = ref(false)
const rejectOfferId = ref<string | null>(null)
const rejectReason = ref('')

const showWithdrawModal = ref(false)
const withdrawOfferId = ref<string | null>(null)
const withdrawReason = ref('')


// Make offer form
const offerForm = ref<MakeOfferRequest>({
  price: { amount: 0, currency: 'RUB' as Currency },
  comment: '',
  vat_type: 'included' as VatType,
  payment_method: 'bank_transfer' as PaymentMethod,
})

// Status colors
const statusColors: Record<string, string> = {
  published: 'bg-green-100 text-green-800',
  selected: 'bg-blue-100 text-blue-800',
  confirmed: 'bg-purple-100 text-purple-800',
  cancelled: 'bg-red-100 text-red-800',
  expired: 'bg-gray-100 text-gray-800',
}

// Computed
const isOwner = computed(() => {
  if (!freightRequest.value) return false
  return permissions.isFreightRequestOwner(freightRequest.value.customer_org_id)
})

const canMakeOffer = computed(() => {
  if (!freightRequest.value) return false
  return (
    permissions.canCreateOffer(freightRequest.value.customer_org_id) &&
    ['published', 'selected'].includes(freightRequest.value.status)
  )
})

const canCancel = computed(() => {
  if (!freightRequest.value) return false
  return (
    permissions.canCancelFreightRequest(
      freightRequest.value.customer_org_id,
      freightRequest.value.customer_member_id
    ) &&
    ['published', 'selected'].includes(freightRequest.value.status)
  )
})

const canManageOffers = computed(() => {
  if (!freightRequest.value) return false
  return permissions.canSelectOffer(
    freightRequest.value.customer_org_id,
    freightRequest.value.customer_member_id
  )
})

const canEdit = computed(() => {
  if (!freightRequest.value) return false
  return (
    permissions.canEditFreightRequest(
      freightRequest.value.customer_org_id,
      freightRequest.value.customer_member_id
    ) &&
    freightRequest.value.status === 'published'
  )
})

const canReassign = computed(() => {
  if (!freightRequest.value) return false
  return (
    permissions.canReassignFreightRequest(freightRequest.value.customer_org_id) &&
    ['published', 'selected'].includes(freightRequest.value.status)
  )
})

// Может видеть ссылку на заказ: владельцы/администраторы организаций заказчика и перевозчика, ответственный за заявку
const canViewOrderLink = computed(() => {
  if (!freightRequest.value || !linkedOrder.value) return false

  const fr = freightRequest.value
  const order = linkedOrder.value
  const isOwnerOrAdmin = auth.role === 'owner' || auth.role === 'administrator'

  // Владелец/администратор организации заказчика
  if (isOwnerOrAdmin && fr.customer_org_id === auth.organizationId) {
    return true
  }

  // Владелец/администратор организации перевозчика
  if (isOwnerOrAdmin && order.carrier_org_id === auth.organizationId) {
    return true
  }

  // Ответственный за заявку
  if (fr.customer_member_id === auth.memberId) {
    return true
  }

  // Ответственный за оффер (перевозчик, который сделал подтверждённый оффер)
  const confirmedOffer = offers.value.find(o => o.status === 'confirmed')
  if (confirmedOffer && confirmedOffer.carrier_member_id === auth.memberId) {
    return true
  }

  return false
})

const myOffers = computed(() => {
  return offers.value.filter((o) => o.carrier_org_id === auth.organizationId)
})

const myActiveOffer = computed(() => {
  return offers.value.find(
    (o) =>
      o.carrier_org_id === auth.organizationId &&
      ['pending', 'selected'].includes(o.status)
  )
})

const visibleOffers = computed(() => {
  if (isOwner.value) {
    return offers.value
  }
  // Non-owner sees only their own offers
  return myOffers.value
})

const requestNumber = computed(() => {
  if (!freightRequest.value) return 0
  return freightRequest.value.request_number
})

// Methods
async function loadData() {
  isLoading.value = true
  error.value = ''
  creatorProfile.value = null
  linkedOrder.value = null
  try {
    const id = route.params.id as string
    const [fr, offersList] = await Promise.all([
      freightRequestsApi.get(id),
      freightRequestsApi.listOffers(id),
    ])
    freightRequest.value = fr
    offers.value = offersList

    // Загружаем профиль создателя, если это член той же организации
    if (fr.customer_org_id === auth.organizationId) {
      try {
        creatorProfile.value = await membersApi.getProfile(fr.customer_member_id)
      } catch {
        // Игнорируем ошибку загрузки профиля, это не критично
      }
    }

    // Загружаем связанный заказ, если заявка подтверждена
    if (fr.status === 'confirmed') {
      try {
        const orders = await ordersApi.list({ freight_request_id: fr.id })
        if (orders.length > 0 && orders[0]) {
          linkedOrder.value = orders[0]
        }
      } catch {
        // Игнорируем ошибку загрузки заказа, это не критично
      }
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatPrice(amount: number, currency: Currency): string {
  const value = amount / 100
  return `${value.toLocaleString('ru-RU')} ${currencyLabels[currency]}`
}

function getPointTypeLabel(point: { is_loading: boolean; is_unloading: boolean }): string {
  if (point.is_loading && point.is_unloading) return 'Погрузка/Разгрузка'
  if (point.is_loading) return 'Погрузка'
  if (point.is_unloading) return 'Разгрузка'
  return 'Точка'
}

// Actions
async function handleCancel() {
  if (!freightRequest.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.cancel(freightRequest.value.id, cancelReason.value || undefined)
    router.push('/')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
    showCancelModal.value = false
  }
}

async function handleMakeOffer() {
  if (!freightRequest.value) return
  if (!offerForm.value.price.amount) {
    error.value = 'Укажите цену'
    return
  }
  actionLoading.value = true
  try {
    await freightRequestsApi.makeOffer(freightRequest.value.id, {
      price: {
        amount: Math.round(offerForm.value.price.amount * 100),
        currency: offerForm.value.price.currency,
      },
      comment: offerForm.value.comment || undefined,
      vat_type: offerForm.value.vat_type,
      payment_method: offerForm.value.payment_method,
    })
    showMakeOfferModal.value = false
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

async function handleSelectOffer(offerId: string) {
  if (!freightRequest.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.selectOffer(freightRequest.value.id, offerId)
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function openRejectModal(offerId: string) {
  rejectOfferId.value = offerId
  rejectReason.value = ''
  showRejectModal.value = true
}

async function confirmRejectOffer() {
  if (!freightRequest.value || !rejectOfferId.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.rejectOffer(freightRequest.value.id, rejectOfferId.value, rejectReason.value || undefined)
    showRejectModal.value = false
    rejectOfferId.value = null
    rejectReason.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function openWithdrawModal(offerId: string) {
  withdrawOfferId.value = offerId
  withdrawReason.value = ''
  showWithdrawModal.value = true
}

async function confirmWithdrawOffer() {
  if (!freightRequest.value || !withdrawOfferId.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.withdrawOffer(freightRequest.value.id, withdrawOfferId.value, withdrawReason.value || undefined)
    showWithdrawModal.value = false
    withdrawOfferId.value = null
    withdrawReason.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function goToReassign() {
  if (!freightRequest.value) return
  router.push({
    path: '/members',
    query: {
      selectFor: 'freightRequest',
      frId: freightRequest.value.id,
    },
  })
}

async function handleConfirmOffer(offerId: string) {
  if (!freightRequest.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.confirmOffer(freightRequest.value.id, offerId)
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

async function handleDeclineOffer(offerId: string) {
  if (!freightRequest.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.declineOffer(freightRequest.value.id, offerId)
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-white shadow">
      <div class="max-w-5xl mx-auto px-4 py-4">
        <router-link to="/" class="text-blue-600 hover:text-blue-800 text-sm">
          &larr; К списку заявок
        </router-link>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-5xl mx-auto px-4 py-6">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="text-gray-500">Загрузка...</div>
      </div>

      <!-- Error -->
      <div v-else-if="error && !freightRequest" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
        {{ error }}
        <button @click="loadData" class="ml-4 text-red-600 underline">Повторить</button>
      </div>

      <!-- Content -->
      <div v-else-if="freightRequest" class="space-y-6">
        <!-- Error banner -->
        <div v-if="error" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
          {{ error }}
          <button @click="error = ''" class="ml-4 text-red-600">&times;</button>
        </div>

        <!-- Tab switcher -->
        <div v-if="canViewHistory" class="mb-6">
          <select
            v-model="currentTab"
            class="w-full sm:w-auto px-4 py-2.5 bg-white border border-gray-300 rounded-lg text-sm font-medium text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 cursor-pointer"
          >
            <option value="details">Детали заявки</option>
            <option value="history">История</option>
          </select>
        </div>

        <!-- Details Tab -->
        <template v-if="currentTab === 'details'">
        <!-- Header -->
        <div class="bg-white rounded-lg shadow p-4 sm:p-6">
          <div class="flex flex-col gap-3 sm:gap-4">
            <div>
              <h1 class="text-xl sm:text-2xl font-bold text-gray-900">Заявка #{{ requestNumber }}</h1>
              <p v-if="creatorProfile" class="text-gray-600 text-sm mt-1">
                Ответственный:
                <router-link
                  :to="`/members/${freightRequest.customer_member_id}`"
                  class="text-blue-600 hover:text-blue-800"
                >
                  {{ creatorProfile.name }}
                </router-link>
                <button
                  v-if="canReassign"
                  @click="goToReassign"
                  class="ml-2 text-gray-400 hover:text-gray-600 text-xs"
                >
                  Ред.
                </button>
              </p>
              <p class="text-gray-500 text-sm mt-1">
                Создана {{ formatDateTime(freightRequest.created_at) }}
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2 sm:gap-3">
              <span :class="[statusColors[freightRequest.status], 'px-3 py-1 rounded-full text-sm font-medium']">
                {{ freightRequestStatusLabels[freightRequest.status] }}
              </span>
              <!-- Ссылка на заказ для подтверждённых заявок -->
              <router-link
                v-if="canViewOrderLink && linkedOrder"
                :to="`/orders/${linkedOrder.id}`"
                class="px-3 py-1.5 sm:px-4 sm:py-2 bg-purple-100 hover:bg-purple-200 text-purple-800 rounded-lg text-sm font-medium inline-flex items-center gap-1.5"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                Заказ #{{ linkedOrder.order_number }}
              </router-link>
              <router-link
                v-if="canEdit"
                :to="`/freight-requests/${freightRequest.id}/edit`"
                class="px-3 py-1.5 sm:px-4 sm:py-2 text-blue-600 hover:bg-blue-50 rounded-lg text-sm font-medium"
              >
                Редактировать
              </router-link>
              <button
                v-if="canCancel"
                @click="showCancelModal = true"
                class="px-3 py-1.5 sm:px-4 sm:py-2 text-red-600 hover:bg-red-50 rounded-lg text-sm font-medium"
              >
                Отменить
              </button>
            </div>
          </div>
        </div>

        <!-- Route Section -->
        <div class="bg-white rounded-lg shadow p-4 sm:p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Маршрут</h2>

          <!-- Map -->
          <LeafletMap
            :points="freightRequest.route.points"
            height="300px"
            class="mb-4"
          />

          <!-- Route Points -->
          <div class="space-y-3">
            <div
              v-for="(point, index) in freightRequest.route.points"
              :key="index"
              class="border border-gray-200 rounded-lg p-4"
            >
              <div class="flex items-start gap-3">
                <div
                  :class="[
                    'w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium',
                    point.is_loading && point.is_unloading ? '' :
                    point.is_loading ? 'bg-blue-500' :
                    point.is_unloading ? 'bg-green-500' : 'bg-gray-400'
                  ]"
                  :style="point.is_loading && point.is_unloading ? 'background: linear-gradient(to right, #3b82f6 50%, #22c55e 50%)' : ''"
                >
                  {{ index + 1 }}
                </div>
                <div class="flex-1">
                  <div class="flex items-center gap-2 mb-1">
                    <span class="text-xs font-medium text-gray-500 uppercase">
                      {{ getPointTypeLabel(point) }}
                    </span>
                  </div>
                  <p class="font-medium text-gray-900 break-words">{{ point.address }}</p>
                  <div class="mt-2 text-sm text-gray-600 space-y-1">
                    <p>
                      <span class="text-gray-500">Дата:</span>
                      {{ formatDate(point.date_from) }}
                      <template v-if="point.date_to"> — {{ formatDate(point.date_to) }}</template>
                    </p>
                    <p v-if="point.time_from">
                      <span class="text-gray-500">Время:</span>
                      {{ point.time_from }}
                      <template v-if="point.time_to"> — {{ point.time_to }}</template>
                    </p>
                    <p v-if="point.contact_name">
                      <span class="text-gray-500">Контакт:</span>
                      {{ point.contact_name }}
                      <template v-if="point.contact_phone">, {{ point.contact_phone }}</template>
                    </p>
                    <p v-if="point.comment" class="text-gray-500 italic break-words">{{ point.comment }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Cargo Section -->
        <div class="bg-white rounded-lg shadow p-4 sm:p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Груз</h2>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <dt class="text-sm text-gray-500">Описание</dt>
              <dd class="text-gray-900 break-words">{{ freightRequest.cargo.description }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Тип груза</dt>
              <dd class="text-gray-900">{{ cargoTypeLabels[freightRequest.cargo.type] }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Вес</dt>
              <dd class="text-gray-900">{{ freightRequest.cargo.weight }} кг</dd>
            </div>
            <div v-if="freightRequest.cargo.volume">
              <dt class="text-sm text-gray-500">Объём</dt>
              <dd class="text-gray-900">{{ freightRequest.cargo.volume }} м³</dd>
            </div>
            <div v-if="freightRequest.cargo.dimensions">
              <dt class="text-sm text-gray-500">Габариты (ДхШхВ)</dt>
              <dd class="text-gray-900">
                {{ freightRequest.cargo.dimensions.length }} x
                {{ freightRequest.cargo.dimensions.width }} x
                {{ freightRequest.cargo.dimensions.height }} м
              </dd>
            </div>
            <div v-if="freightRequest.cargo.quantity">
              <dt class="text-sm text-gray-500">Количество</dt>
              <dd class="text-gray-900">{{ freightRequest.cargo.quantity }} шт</dd>
            </div>
            <div v-if="freightRequest.cargo.adr_class && freightRequest.cargo.adr_class !== 'none'">
              <dt class="text-sm text-gray-500">ADR класс</dt>
              <dd class="text-gray-900">{{ adrClassLabels[freightRequest.cargo.adr_class] }}</dd>
            </div>
          </dl>
        </div>

        <!-- Vehicle Requirements Section -->
        <div class="bg-white rounded-lg shadow p-4 sm:p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Требования к транспорту</h2>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="sm:col-span-2">
              <dt class="text-sm text-gray-500 mb-2">Типы кузова</dt>
              <dd class="flex flex-wrap gap-2">
                <span
                  v-for="bodyType in freightRequest.vehicle_requirements.body_types"
                  :key="bodyType"
                  class="px-2 py-1 bg-gray-100 text-gray-700 rounded text-sm"
                >
                  {{ bodyTypeLabels[bodyType] }}
                </span>
              </dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.loading_types?.length" class="sm:col-span-2">
              <dt class="text-sm text-gray-500 mb-2">Типы загрузки</dt>
              <dd class="flex flex-wrap gap-2">
                <span
                  v-for="loadingType in freightRequest.vehicle_requirements.loading_types"
                  :key="loadingType"
                  class="px-2 py-1 bg-gray-100 text-gray-700 rounded text-sm"
                >
                  {{ loadingTypeLabels[loadingType] }}
                </span>
              </dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.capacity">
              <dt class="text-sm text-gray-500">Грузоподъёмность</dt>
              <dd class="text-gray-900">{{ freightRequest.vehicle_requirements.capacity }} т</dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.volume">
              <dt class="text-sm text-gray-500">Объём</dt>
              <dd class="text-gray-900">{{ freightRequest.vehicle_requirements.volume }} м³</dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.length">
              <dt class="text-sm text-gray-500">Длина</dt>
              <dd class="text-gray-900">{{ freightRequest.vehicle_requirements.length }} м</dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.width">
              <dt class="text-sm text-gray-500">Ширина</dt>
              <dd class="text-gray-900">{{ freightRequest.vehicle_requirements.width }} м</dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.height">
              <dt class="text-sm text-gray-500">Высота</dt>
              <dd class="text-gray-900">{{ freightRequest.vehicle_requirements.height }} м</dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.requires_adr">
              <dt class="text-sm text-gray-500">ADR</dt>
              <dd class="text-gray-900">Требуется</dd>
            </div>
            <div v-if="freightRequest.vehicle_requirements.temperature">
              <dt class="text-sm text-gray-500">Температурный режим</dt>
              <dd class="text-gray-900">
                от {{ freightRequest.vehicle_requirements.temperature.min }}°C
                до {{ freightRequest.vehicle_requirements.temperature.max }}°C
              </dd>
            </div>
          </dl>
        </div>

        <!-- Payment Section -->
        <div class="bg-white rounded-lg shadow p-4 sm:p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Оплата</h2>

          <!-- Если цена указана -->
          <template v-if="freightRequest.payment.price && freightRequest.payment.price.amount > 0">
            <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <dt class="text-sm text-gray-500">Цена</dt>
                <dd class="text-gray-900 text-xl font-semibold">
                  {{ formatPrice(freightRequest.payment.price.amount, freightRequest.payment.price.currency) }}
                </dd>
              </div>
              <div>
                <dt class="text-sm text-gray-500">НДС</dt>
                <dd class="text-gray-900">{{ vatTypeLabels[freightRequest.payment.vat_type] }}</dd>
              </div>
              <div>
                <dt class="text-sm text-gray-500">Способ оплаты</dt>
                <dd class="text-gray-900">{{ paymentMethodLabels[freightRequest.payment.method] }}</dd>
              </div>
              <div>
                <dt class="text-sm text-gray-500">Условия оплаты</dt>
                <dd class="text-gray-900">
                  {{ paymentTermsLabels[freightRequest.payment.terms] }}
                  <template v-if="freightRequest.payment.terms === 'deferred' && freightRequest.payment.deferred_days">
                    ({{ freightRequest.payment.deferred_days }} дней)
                  </template>
                </dd>
              </div>
            </dl>
          </template>

          <!-- Если цена не указана -->
          <template v-else>
            <p class="text-gray-500">Цена не указана — перевозчики предложат свою</p>
          </template>
        </div>

        <!-- Comment -->
        <div v-if="freightRequest.comment" class="bg-white rounded-lg shadow p-4 sm:p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-2">Комментарий</h2>
          <p class="text-gray-700 break-words">{{ freightRequest.comment }}</p>
        </div>

        <!-- Offers Section -->
        <div v-if="visibleOffers.length > 0 || isOwner" class="bg-white rounded-lg shadow p-4 sm:p-6">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-semibold text-gray-900">
              Предложения
              <span v-if="isOwner" class="text-gray-500 font-normal">({{ offers.length }})</span>
            </h2>
          </div>

          <div v-if="visibleOffers.length === 0" class="text-center py-8 text-gray-500">
            Пока нет предложений
          </div>

          <div v-else class="space-y-4">
            <div
              v-for="offer in visibleOffers"
              :key="offer.id"
              :class="[
                'border rounded-lg p-4',
                offer.status === 'selected' ? 'border-blue-300 bg-blue-50' :
                offer.status === 'confirmed' ? 'border-green-300 bg-green-50' :
                'border-gray-200'
              ]"
            >
              <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                <div class="flex-1">
                  <div class="flex items-center gap-3 mb-2">
                    <span class="text-xl font-semibold text-gray-900">
                      {{ formatPrice(offer.price.amount, offer.price.currency) }}
                    </span>
                    <span :class="[offerStatusColors[offer.status], 'px-2 py-0.5 rounded text-xs font-medium']">
                      {{ offerStatusLabels[offer.status] }}
                    </span>
                  </div>
                  <div v-if="offer.carrier_org_name || offer.carrier_member_name" class="mb-2 flex flex-wrap items-center gap-x-2 min-w-0">
                    <router-link
                      v-if="isOwner && offer.carrier_org_name"
                      :to="`/organizations/${offer.carrier_org_id}`"
                      class="text-blue-600 hover:text-blue-800 font-medium truncate max-w-full"
                    >
                      {{ offer.carrier_org_name }}
                    </router-link>
                    <span v-if="isOwner && offer.carrier_org_name && offer.carrier_member_name" class="text-gray-400 shrink-0">•</span>
                    <router-link
                      v-if="offer.carrier_member_name"
                      :to="`/members/${offer.carrier_member_id}`"
                      class="text-blue-600 hover:text-blue-800 truncate max-w-full"
                    >
                      {{ offer.carrier_member_name }}
                    </router-link>
                  </div>
                  <div class="text-sm text-gray-600 space-y-1">
                    <p>
                      <span class="text-gray-500">НДС:</span>
                      {{ vatTypeLabels[offer.vat_type] }}
                    </p>
                    <p>
                      <span class="text-gray-500">Способ оплаты:</span>
                      {{ paymentMethodLabels[offer.payment_method] }}
                    </p>
                    <p v-if="offer.comment" class="break-words">
                      <span class="text-gray-500">Комментарий:</span>
                      {{ offer.comment }}
                    </p>
                    <p class="text-gray-400 text-xs">
                      {{ formatDateTime(offer.created_at) }}
                    </p>
                  </div>
                </div>

                <!-- Owner/Admin actions -->
                <div v-if="canManageOffers && offer.status === 'pending' && freightRequest.status === 'published'" class="flex gap-2">
                  <button
                    @click="handleSelectOffer(offer.id)"
                    :disabled="actionLoading"
                    class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg disabled:opacity-50"
                  >
                    Выбрать
                  </button>
                  <button
                    @click="openRejectModal(offer.id)"
                    :disabled="actionLoading"
                    class="px-3 py-1.5 bg-gray-200 hover:bg-gray-300 text-gray-700 text-sm rounded-lg disabled:opacity-50"
                  >
                    Отклонить
                  </button>
                </div>

                <!-- Carrier actions (own offer) -->
                <div v-if="!isOwner && offer.carrier_org_id === auth.organizationId" class="flex gap-2">
                  <template v-if="offer.status === 'pending' && permissions.canWithdrawOffer(offer.carrier_org_id, offer.carrier_member_id)">
                    <button
                      @click="openWithdrawModal(offer.id)"
                      :disabled="actionLoading"
                      class="px-3 py-1.5 bg-gray-200 hover:bg-gray-300 text-gray-700 text-sm rounded-lg disabled:opacity-50"
                    >
                      Отозвать
                    </button>
                  </template>
                  <template v-if="offer.status === 'selected'">
                    <button
                      @click="handleConfirmOffer(offer.id)"
                      :disabled="actionLoading"
                      class="px-3 py-1.5 bg-green-600 hover:bg-green-700 text-white text-sm rounded-lg disabled:opacity-50"
                    >
                      Подтвердить
                    </button>
                    <button
                      @click="handleDeclineOffer(offer.id)"
                      :disabled="actionLoading"
                      class="px-3 py-1.5 bg-gray-200 hover:bg-gray-300 text-gray-700 text-sm rounded-lg disabled:opacity-50"
                    >
                      Отказаться
                    </button>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Make Offer Button -->
        <div v-if="canMakeOffer && !myActiveOffer" class="flex justify-center">
          <button
            @click="showMakeOfferModal = true"
            class="px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg"
          >
            Сделать предложение
          </button>
        </div>
        </template>

        <!-- History Tab -->
        <template v-if="currentTab === 'history'">
          <div class="bg-white rounded-lg shadow p-6">
            <h2 class="text-lg font-semibold text-gray-900 mb-4">История изменений</h2>
            <EventHistory :load-fn="loadFreightRequestHistory" />
          </div>
        </template>
      </div>
    </main>

    <!-- Make Offer Modal -->
    <div v-if="showMakeOfferModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-[1000]">
      <div class="bg-white rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Сделать предложение</h3>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Цена *</label>
            <div class="flex gap-2">
              <input
                v-model.number="offerForm.price.amount"
                type="number"
                min="0"
                step="100"
                placeholder="0"
                class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <select
                v-model="offerForm.price.currency"
                class="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option v-for="opt in currencyOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">НДС</label>
            <select
              v-model="offerForm.vat_type"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="opt in vatTypeOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Способ оплаты</label>
            <select
              v-model="offerForm.payment_method"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option v-for="opt in paymentMethodOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Комментарий</label>
            <textarea
              v-model="offerForm.comment"
              rows="2"
              placeholder="Дополнительная информация..."
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            ></textarea>
          </div>
        </div>

        <div class="flex gap-3 mt-6">
          <button
            @click="showMakeOfferModal = false"
            class="flex-1 py-2 px-4 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg"
          >
            Отмена
          </button>
          <button
            @click="handleMakeOffer"
            :disabled="!offerForm.price.amount || actionLoading"
            class="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отправка...' : 'Отправить' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Cancel Modal -->
    <div v-if="showCancelModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-[1000]">
      <div class="bg-white rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Отменить заявку</h3>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-1">Причина отмены</label>
          <textarea
            v-model="cancelReason"
            rows="3"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Укажите причину (опционально)..."
          ></textarea>
        </div>

        <div class="flex gap-3">
          <button
            @click="showCancelModal = false"
            class="flex-1 py-2 px-4 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg"
          >
            Назад
          </button>
          <button
            @click="handleCancel"
            :disabled="actionLoading"
            class="flex-1 py-2 px-4 bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отмена...' : 'Отменить заявку' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Reject Offer Modal -->
    <div v-if="showRejectModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-[1000]">
      <div class="bg-white rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Отклонить предложение</h3>

        <p class="text-gray-600 mb-4">Вы уверены, что хотите отклонить это предложение?</p>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-1">Причина отклонения</label>
          <textarea
            v-model="rejectReason"
            rows="3"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Укажите причину (опционально)..."
          ></textarea>
        </div>

        <div class="flex gap-3">
          <button
            @click="showRejectModal = false"
            class="flex-1 py-2 px-4 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg"
          >
            Отмена
          </button>
          <button
            @click="confirmRejectOffer"
            :disabled="actionLoading"
            class="flex-1 py-2 px-4 bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отклонение...' : 'Отклонить' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Withdraw Offer Modal -->
    <div v-if="showWithdrawModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-[1000]">
      <div class="bg-white rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Отозвать предложение</h3>

        <p class="text-gray-600 mb-4">Вы уверены, что хотите отозвать своё предложение?</p>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-1">Причина отзыва</label>
          <textarea
            v-model="withdrawReason"
            rows="3"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Укажите причину (опционально)..."
          ></textarea>
        </div>

        <div class="flex gap-3">
          <button
            @click="showWithdrawModal = false"
            class="flex-1 py-2 px-4 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg"
          >
            Отмена
          </button>
          <button
            @click="confirmWithdrawOffer"
            :disabled="actionLoading"
            class="flex-1 py-2 px-4 bg-orange-600 hover:bg-orange-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отзыв...' : 'Отозвать' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>
