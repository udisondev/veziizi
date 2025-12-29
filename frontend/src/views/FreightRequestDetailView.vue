<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
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
import FreightRequestOffersTab from '@/components/freight-request/FreightRequestOffersTab.vue'
import type {
  FreightRequest,
  Offer,
  MakeOfferRequest,
  Currency,
  VatType,
  PaymentMethod,
} from '@/types/freightRequest'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  loadingTypeLabels,
  currencyLabels,
  vatTypeLabels,
  paymentMethodLabels,
  paymentTermsLabels,
  adrClassLabels,
  currencyOptions,
  vatTypeOptions,
  paymentMethodOptions,
} from '@/types/freightRequest'
import { freightRequestStatusMap, offerStatusMap } from '@/constants/statusMaps'
import { formatDate, formatDateTime, formatMoney } from '@/utils/formatters'
import { logger } from '@/utils/logger'

// UI Components
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent } from '@/components/ui/tabs'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

// Shared Components
import { DetailPageHeader, StatusBadge, LoadingSpinner, ErrorBanner, TabsDropdown, type TabItem } from '@/components/shared'

// Icons
import {
  FileText,
  Pencil,
  XCircle,
  MoreVertical,
  Users,
  MapPin,
  Package,
  Truck,
  CreditCard,
  MessageSquare,
  Check,
  X,
  Clock,
  Building2,
} from 'lucide-vue-next'

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
const currentTab = ref('details')

// History loader
function loadFreightRequestHistory(limit: number, offset: number) {
  const id = route.params.id as string
  return historyApi.getFreightRequestHistory(id, { limit, offset })
}

// Check if user can view history
const canViewHistory = computed(() => {
  if (!freightRequest.value) return false
  if (freightRequest.value.customer_org_id !== auth.organizationId) return false
  return auth.role === 'owner' || auth.role === 'administrator'
})

const tabItems = computed((): TabItem[] => {
  const items: TabItem[] = [
    { value: 'details', label: 'Детали заявки', icon: FileText },
  ]
  if (visibleOffers.value.length > 0 || isOwner.value) {
    items.push({ value: 'offers', label: 'Предложения', icon: Users, badge: offers.value.length || undefined })
  }
  if (canViewHistory.value) {
    items.push({ value: 'history', label: 'История', icon: Clock, separator: true })
  }
  return items
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
const acceptRequestTerms = ref(false)

// Computed
const hasRequestRate = computed(() => {
  return freightRequest.value?.payment?.price?.amount && freightRequest.value.payment.price.amount > 0
})
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

const canViewOrderLink = computed(() => {
  if (!freightRequest.value || !linkedOrder.value) return false

  const fr = freightRequest.value
  const order = linkedOrder.value

  // Любой член организации-заказчика или организации-перевозчика может видеть ссылку
  return fr.customer_org_id === auth.organizationId || order.carrier_org_id === auth.organizationId
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
  return offers.value.filter((o) => o.carrier_org_id === auth.organizationId)
})

const requestNumber = computed(() => {
  if (!freightRequest.value) return 0
  return freightRequest.value.request_number
})

// Watch для автозаполнения формы при согласии с условиями
watch(acceptRequestTerms, (accepted) => {
  if (!freightRequest.value) return

  if (accepted && hasRequestRate.value) {
    // Заполнить данными из заявки (amount в копейках -> конвертируем в рубли)
    const payment = freightRequest.value.payment
    offerForm.value.price.amount = payment.price!.amount / 100
    offerForm.value.price.currency = payment.price!.currency
    offerForm.value.vat_type = payment.vat_type
    offerForm.value.payment_method = payment.method
  } else {
    // Сброс к значениям по умолчанию
    offerForm.value.price.amount = 0
    offerForm.value.price.currency = 'RUB'
    offerForm.value.vat_type = 'included'
    offerForm.value.payment_method = 'bank_transfer'
  }
})

// Сброс галочки при закрытии модалки
watch(showMakeOfferModal, (open) => {
  if (!open) {
    acceptRequestTerms.value = false
  }
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

    if (fr.customer_org_id === auth.organizationId) {
      try {
        creatorProfile.value = await membersApi.getProfile(fr.customer_member_id)
      } catch (e) {
        logger.warn('Failed to load creator profile', e)
      }
    }

    if (fr.status === 'confirmed') {
      try {
        const orders = await ordersApi.list({ freight_request_id: fr.id })
        if (orders.length > 0 && orders[0]) {
          linkedOrder.value = orders[0]
        }
      } catch (e) {
        logger.warn('Failed to load linked order', e)
      }
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatPrice(amount: number, currency: Currency): string {
  return formatMoney({ amount, currency })
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
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <DetailPageHeader back-to="/" back-label="К списку заявок">
      <template #actions>
        <div class="flex items-center gap-2">
          <!-- Make Offer Button for carriers -->
          <Button
            v-if="canMakeOffer && !myActiveOffer"
            size="sm"
            @click="showMakeOfferModal = true"
          >
            Сделать предложение
          </Button>

          <DropdownMenu v-if="freightRequest && (canEdit || canCancel)">
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="icon">
                <MoreVertical class="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                v-if="canEdit"
                @click="router.push(`/freight-requests/${freightRequest.id}/edit`)"
              >
                <Pencil class="mr-2 h-4 w-4" />
                Редактировать
              </DropdownMenuItem>
              <DropdownMenuSeparator v-if="canEdit && canCancel" />
              <DropdownMenuItem
                v-if="canCancel"
                class="text-destructive focus:text-destructive"
                @click="showCancelModal = true"
              >
                <XCircle class="mr-2 h-4 w-4" />
                Отменить заявку
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </template>
    </DetailPageHeader>

    <!-- Content -->
    <main class="max-w-5xl mx-auto px-4 py-6">
      <!-- Loading -->
      <LoadingSpinner v-if="isLoading" text="Загрузка заявки..." />

      <!-- Error -->
      <ErrorBanner
        v-else-if="error && !freightRequest"
        :message="error"
        @retry="loadData"
      />

      <!-- Content -->
      <div v-else-if="freightRequest" class="space-y-6">
        <!-- Error banner -->
        <Card v-if="error" class="border-destructive/50 bg-destructive/5">
          <CardContent class="flex items-center justify-between py-3">
            <span class="text-sm text-destructive">{{ error }}</span>
            <Button variant="ghost" size="sm" @click="error = ''">
              <X class="h-4 w-4" />
            </Button>
          </CardContent>
        </Card>

        <!-- Tabs -->
        <Tabs v-model="currentTab" class="w-full">
          <!-- Tab selector dropdown -->
          <div v-if="tabItems.length > 1" class="mb-6">
            <TabsDropdown v-model="currentTab" :items="tabItems" />
          </div>

          <!-- Details Tab -->
          <TabsContent value="details" class="space-y-6">
            <!-- Header Card -->
            <Card>
              <CardContent class="p-4 sm:p-6">
                <div class="flex flex-col gap-4">
                  <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                    <div class="min-w-0">
                      <div class="flex items-center gap-3">
                        <h1 class="text-xl sm:text-2xl font-bold text-foreground">
                          Заявка #{{ requestNumber }}
                        </h1>
                        <StatusBadge :status="freightRequest.status" :status-map="freightStatusMap" />
                      </div>
                      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground mt-2">
                        <!-- Заказчик -->
                        <router-link
                          :to="`/organizations/${freightRequest.customer_org_id}`"
                          class="inline-flex items-center gap-1 text-primary hover:underline max-w-[200px] sm:max-w-[280px]"
                          :title="freightRequest.customer_org_name || 'Организация'"
                        >
                          <Building2 class="h-4 w-4 shrink-0" />
                          <span class="truncate">{{ freightRequest.customer_org_name || 'Организация' }}</span>
                        </router-link>

                        <!-- Ответственный -->
                        <div v-if="creatorProfile" class="inline-flex items-center gap-1 max-w-[200px] sm:max-w-[280px]">
                          <Users class="h-4 w-4 shrink-0 text-muted-foreground" />
                          <router-link
                            :to="`/members/${freightRequest.customer_member_id}`"
                            class="text-primary hover:underline truncate"
                            :title="creatorProfile.name"
                          >
                            {{ creatorProfile.name }}
                          </router-link>
                          <Button
                            v-if="canReassign"
                            variant="ghost"
                            size="sm"
                            class="h-auto p-0.5 shrink-0"
                            @click="goToReassign"
                          >
                            <Pencil class="h-3 w-3" />
                          </Button>
                        </div>

                        <!-- Дата создания -->
                        <span class="inline-flex items-center gap-1">
                          <Clock class="h-4 w-4 shrink-0" />
                          {{ formatDateTime(freightRequest.created_at) }}
                        </span>
                      </div>
                    </div>

                    <!-- Order link -->
                    <router-link
                      v-if="canViewOrderLink && linkedOrder"
                      :to="`/orders/${linkedOrder.id}`"
                      class="text-primary hover:underline text-sm flex items-center gap-1"
                    >
                      <FileText class="h-3 w-3" />
                      Перейти к заказу
                    </router-link>
                  </div>
                </div>
              </CardContent>
            </Card>

            <!-- Route Section -->
            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2">
                  <MapPin class="h-5 w-5" />
                  Маршрут
                </CardTitle>
              </CardHeader>
              <CardContent class="space-y-4">
                <!-- Map -->
                <LeafletMap
                  :points="freightRequest.route.points"
                  height="300px"
                />

                <!-- Route Points -->
                <div class="divide-y">
                  <div
                    v-for="(point, index) in freightRequest.route.points"
                    :key="index"
                    class="py-4 first:pt-0 last:pb-0"
                  >
                    <div class="flex items-start gap-3">
                      <div
                        :class="[
                          'w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium shrink-0',
                          point.is_loading && point.is_unloading ? 'bg-gradient-to-r from-primary from-50% to-success to-50%' :
                          point.is_loading ? 'bg-primary' :
                          point.is_unloading ? 'bg-success' : 'bg-muted-foreground'
                        ]"
                      >
                        {{ index + 1 }}
                      </div>
                      <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2 mb-1">
                          <Badge variant="outline">
                            {{ getPointTypeLabel(point) }}
                          </Badge>
                        </div>
                        <p class="font-medium text-foreground break-words">{{ point.address }}</p>
                        <p class="text-sm text-muted-foreground mt-1">
                          {{ formatDate(point.date_from) }}<template v-if="point.date_to"> — {{ formatDate(point.date_to) }}</template>
                          <template v-if="point.time_from">
                            <span class="mx-1">·</span>
                            {{ point.time_from }}<template v-if="point.time_to"> — {{ point.time_to }}</template>
                          </template>
                        </p>
                        <p v-if="point.contact_name" class="text-sm text-muted-foreground">
                          {{ point.contact_name }}<template v-if="point.contact_phone">, {{ point.contact_phone }}</template>
                        </p>
                        <p v-if="point.comment" class="text-sm text-muted-foreground italic mt-1 break-words">{{ point.comment }}</p>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            <!-- Cargo Section -->
            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2">
                  <Package class="h-5 w-5" />
                  Груз
                </CardTitle>
              </CardHeader>
              <CardContent>
                <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <dt class="text-sm text-muted-foreground">Описание</dt>
                    <dd class="text-foreground break-words">{{ freightRequest.cargo.description }}</dd>
                  </div>
                  <div>
                    <dt class="text-sm text-muted-foreground">Вес</dt>
                    <dd class="text-foreground">{{ freightRequest.cargo.weight }} кг</dd>
                  </div>
                  <div v-if="freightRequest.cargo.volume">
                    <dt class="text-sm text-muted-foreground">Объём</dt>
                    <dd class="text-foreground">{{ freightRequest.cargo.volume }} м³</dd>
                  </div>
                  <div v-if="freightRequest.cargo.dimensions">
                    <dt class="text-sm text-muted-foreground">Габариты (ДхШхВ)</dt>
                    <dd class="text-foreground">
                      {{ freightRequest.cargo.dimensions.length }} x
                      {{ freightRequest.cargo.dimensions.width }} x
                      {{ freightRequest.cargo.dimensions.height }} м
                    </dd>
                  </div>
                  <div v-if="freightRequest.cargo.quantity">
                    <dt class="text-sm text-muted-foreground">Количество</dt>
                    <dd class="text-foreground">{{ freightRequest.cargo.quantity }} шт</dd>
                  </div>
                  <div v-if="freightRequest.cargo.adr_class && freightRequest.cargo.adr_class !== 'none'">
                    <dt class="text-sm text-muted-foreground">ADR класс</dt>
                    <dd class="text-foreground">{{ adrClassLabels[freightRequest.cargo.adr_class] }}</dd>
                  </div>
                </dl>
              </CardContent>
            </Card>

            <!-- Vehicle Requirements Section -->
            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2">
                  <Truck class="h-5 w-5" />
                  Требования к транспорту
                </CardTitle>
              </CardHeader>
              <CardContent>
                <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <dt class="text-sm text-muted-foreground">Тип транспорта</dt>
                    <dd class="text-foreground">{{ vehicleTypeLabels[freightRequest.vehicle_requirements.vehicle_type] }}</dd>
                  </div>
                  <div>
                    <dt class="text-sm text-muted-foreground">Тип кузова</dt>
                    <dd class="text-foreground">{{ vehicleSubTypeLabels[freightRequest.vehicle_requirements.vehicle_subtype] }}</dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.loading_types?.length">
                    <dt class="text-sm text-muted-foreground">Типы загрузки</dt>
                    <dd class="text-foreground">
                      {{ freightRequest.vehicle_requirements.loading_types.map(t => loadingTypeLabels[t]).join(', ') }}
                    </dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.capacity">
                    <dt class="text-sm text-muted-foreground">Грузоподъёмность</dt>
                    <dd class="text-foreground">{{ freightRequest.vehicle_requirements.capacity }} т</dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.volume">
                    <dt class="text-sm text-muted-foreground">Объём</dt>
                    <dd class="text-foreground">{{ freightRequest.vehicle_requirements.volume }} м³</dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.length">
                    <dt class="text-sm text-muted-foreground">Длина</dt>
                    <dd class="text-foreground">{{ freightRequest.vehicle_requirements.length }} м</dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.width">
                    <dt class="text-sm text-muted-foreground">Ширина</dt>
                    <dd class="text-foreground">{{ freightRequest.vehicle_requirements.width }} м</dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.height">
                    <dt class="text-sm text-muted-foreground">Высота</dt>
                    <dd class="text-foreground">{{ freightRequest.vehicle_requirements.height }} м</dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.temperature">
                    <dt class="text-sm text-muted-foreground">Температурный режим</dt>
                    <dd class="text-foreground">
                      от {{ freightRequest.vehicle_requirements.temperature.min }}°C
                      до {{ freightRequest.vehicle_requirements.temperature.max }}°C
                    </dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.thermograph">
                    <dt class="text-sm text-muted-foreground">Термописец</dt>
                    <dd class="text-foreground">Да</dd>
                  </div>
                </dl>
              </CardContent>
            </Card>

            <!-- Payment Section -->
            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2">
                  <CreditCard class="h-5 w-5" />
                  Оплата
                </CardTitle>
              </CardHeader>
              <CardContent>
                <template v-if="freightRequest.payment.price && freightRequest.payment.price.amount > 0">
                  <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <dt class="text-sm text-muted-foreground">Цена</dt>
                      <dd class="text-xl font-semibold text-success">
                        {{ formatPrice(freightRequest.payment.price.amount, freightRequest.payment.price.currency) }}
                      </dd>
                    </div>
                    <div>
                      <dt class="text-sm text-muted-foreground">НДС</dt>
                      <dd class="text-foreground">{{ vatTypeLabels[freightRequest.payment.vat_type] }}</dd>
                    </div>
                    <div>
                      <dt class="text-sm text-muted-foreground">Способ оплаты</dt>
                      <dd class="text-foreground">{{ paymentMethodLabels[freightRequest.payment.method] }}</dd>
                    </div>
                    <div>
                      <dt class="text-sm text-muted-foreground">Условия оплаты</dt>
                      <dd class="text-foreground">
                        {{ paymentTermsLabels[freightRequest.payment.terms] }}
                        <template v-if="freightRequest.payment.terms === 'deferred' && freightRequest.payment.deferred_days">
                          ({{ freightRequest.payment.deferred_days }} дней)
                        </template>
                      </dd>
                    </div>
                  </dl>
                </template>
                <template v-else>
                  <p class="text-muted-foreground">Цена не указана — перевозчики предложат свою</p>
                </template>
              </CardContent>
            </Card>

            <!-- Comment -->
            <Card v-if="freightRequest.comment">
              <CardHeader>
                <CardTitle class="flex items-center gap-2">
                  <MessageSquare class="h-5 w-5" />
                  Комментарий
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p class="text-foreground break-words">{{ freightRequest.comment }}</p>
              </CardContent>
            </Card>
          </TabsContent>

          <!-- Offers Tab -->
          <TabsContent value="offers">
            <FreightRequestOffersTab
              :freight-request="freightRequest"
              :offers="offers"
              :action-loading="actionLoading"
              @select="handleSelectOffer"
              @reject="openRejectModal"
              @withdraw="openWithdrawModal"
              @confirm="handleConfirmOffer"
              @decline="handleDeclineOffer"
            />
          </TabsContent>

          <!-- History Tab -->
          <TabsContent value="history">
            <Card>
              <CardHeader>
                <CardTitle>История изменений</CardTitle>
              </CardHeader>
              <CardContent>
                <EventHistory :load-fn="loadFreightRequestHistory" />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </main>

    <!-- Make Offer Dialog -->
    <Dialog v-model:open="showMakeOfferModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Сделать предложение</DialogTitle>
          <DialogDescription>
            Укажите условия вашего предложения
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-4">
          <!-- Чекбокс "Согласен с условиями" - показывается только если есть ставка -->
          <div v-if="hasRequestRate" class="flex items-center space-x-2">
            <Checkbox
              id="accept-terms"
              :checked="acceptRequestTerms"
              @update:checked="acceptRequestTerms = $event"
            />
            <label
              for="accept-terms"
              class="text-sm font-medium leading-none cursor-pointer"
            >
              Согласен с условиями заказчика
            </label>
          </div>

          <div class="space-y-2">
            <Label>Цена *</Label>
            <div class="flex gap-2">
              <Input
                v-model.number="offerForm.price.amount"
                type="number"
                min="0"
                step="100"
                placeholder="0"
                class="flex-1"
                :disabled="acceptRequestTerms"
              />
              <Select v-model="offerForm.price.currency" :disabled="acceptRequestTerms">
                <SelectTrigger class="w-24">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="opt in currencyOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div class="space-y-2">
            <Label>НДС</Label>
            <Select v-model="offerForm.vat_type" :disabled="acceptRequestTerms">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in vatTypeOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="space-y-2">
            <Label>Способ оплаты</Label>
            <Select v-model="offerForm.payment_method" :disabled="acceptRequestTerms">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in paymentMethodOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="space-y-2">
            <Label>Комментарий</Label>
            <Textarea
              v-model="offerForm.comment"
              rows="2"
              placeholder="Дополнительная информация..."
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showMakeOfferModal = false">
            Отмена
          </Button>
          <Button
            :disabled="!offerForm.price.amount || actionLoading"
            @click="handleMakeOffer"
          >
            {{ actionLoading ? 'Отправка...' : 'Отправить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Cancel Dialog -->
    <Dialog v-model:open="showCancelModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отменить заявку</DialogTitle>
          <DialogDescription>
            Это действие нельзя отменить
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отмены</Label>
          <Textarea
            v-model="cancelReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showCancelModal = false">
            Назад
          </Button>
          <Button
            variant="destructive"
            :disabled="actionLoading"
            @click="handleCancel"
          >
            {{ actionLoading ? 'Отмена...' : 'Отменить заявку' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Reject Offer Dialog -->
    <Dialog v-model:open="showRejectModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отклонить предложение</DialogTitle>
          <DialogDescription>
            Вы уверены, что хотите отклонить это предложение?
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отклонения</Label>
          <Textarea
            v-model="rejectReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showRejectModal = false">
            Отмена
          </Button>
          <Button
            variant="destructive"
            :disabled="actionLoading"
            @click="confirmRejectOffer"
          >
            {{ actionLoading ? 'Отклонение...' : 'Отклонить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Withdraw Offer Dialog -->
    <Dialog v-model:open="showWithdrawModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отозвать предложение</DialogTitle>
          <DialogDescription>
            Вы уверены, что хотите отозвать своё предложение?
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отзыва</Label>
          <Textarea
            v-model="withdrawReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showWithdrawModal = false">
            Отмена
          </Button>
          <Button
            variant="secondary"
            :disabled="actionLoading"
            @click="confirmWithdrawOffer"
          >
            {{ actionLoading ? 'Отзыв...' : 'Отозвать' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
