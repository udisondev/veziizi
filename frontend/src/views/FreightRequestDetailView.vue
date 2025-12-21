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
  cargoTypeLabels,
  bodyTypeLabels,
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

// UI Components
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import { BackLink, StatusBadge, LoadingSpinner, ErrorBanner, TabsDropdown, type TabItem } from '@/components/shared'

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
  Send,
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

const tabItems = computed((): TabItem[] => [
  { value: 'details', label: 'Детали заявки', icon: FileText },
  { value: 'history', label: 'История', icon: Clock },
])

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

// Status map for StatusBadge
const freightStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  published: { label: 'Опубликована', variant: 'success' },
  selected: { label: 'Выбран исполнитель', variant: 'warning' },
  confirmed: { label: 'Подтверждена', variant: 'info' },
  cancelled: { label: 'Отменена', variant: 'destructive' },
  expired: { label: 'Истекла', variant: 'secondary' },
}

const offerStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  pending: { label: 'Ожидает', variant: 'secondary' },
  selected: { label: 'Выбран', variant: 'info' },
  confirmed: { label: 'Подтверждён', variant: 'success' },
  rejected: { label: 'Отклонён', variant: 'destructive' },
  withdrawn: { label: 'Отозван', variant: 'secondary' },
  declined: { label: 'Отказ', variant: 'destructive' },
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

const canViewOrderLink = computed(() => {
  if (!freightRequest.value || !linkedOrder.value) return false

  const fr = freightRequest.value
  const order = linkedOrder.value
  const isOwnerOrAdmin = auth.role === 'owner' || auth.role === 'administrator'

  if (isOwnerOrAdmin && fr.customer_org_id === auth.organizationId) {
    return true
  }

  if (isOwnerOrAdmin && order.carrier_org_id === auth.organizationId) {
    return true
  }

  if (fr.customer_member_id === auth.memberId) {
    return true
  }

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

    if (fr.customer_org_id === auth.organizationId) {
      try {
        creatorProfile.value = await membersApi.getProfile(fr.customer_member_id)
      } catch {
        // Ignore
      }
    }

    if (fr.status === 'confirmed') {
      try {
        const orders = await ordersApi.list({ freight_request_id: fr.id })
        if (orders.length > 0 && orders[0]) {
          linkedOrder.value = orders[0]
        }
      } catch {
        // Ignore
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
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <header class="bg-card border-b">
      <div class="max-w-5xl mx-auto px-4 py-4">
        <BackLink to="/" label="К списку заявок" />
      </div>
    </header>

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
          <div v-if="canViewHistory" class="mb-6">
            <TabsDropdown v-model="currentTab" :items="tabItems" />
          </div>

          <!-- Details Tab -->
          <TabsContent value="details" class="space-y-6">
            <!-- Header Card -->
            <Card>
              <CardContent class="p-4 sm:p-6">
                <div class="flex flex-col gap-4">
                  <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                    <div>
                      <h1 class="text-xl sm:text-2xl font-bold text-foreground">
                        Заявка #{{ requestNumber }}
                      </h1>
                      <p v-if="creatorProfile" class="text-muted-foreground text-sm mt-1">
                        Ответственный:
                        <router-link
                          :to="`/members/${freightRequest.customer_member_id}`"
                          class="text-primary hover:underline"
                        >
                          {{ creatorProfile.name }}
                        </router-link>
                        <Button
                          v-if="canReassign"
                          variant="ghost"
                          size="sm"
                          class="ml-1 h-auto py-0 px-1 text-xs"
                          @click="goToReassign"
                        >
                          <Pencil class="h-3 w-3" />
                        </Button>
                      </p>
                      <p class="text-muted-foreground text-sm mt-1">
                        Создана {{ formatDateTime(freightRequest.created_at) }}
                      </p>
                    </div>

                    <!-- Actions -->
                    <div class="flex flex-wrap items-center gap-2">
                      <StatusBadge :status="freightRequest.status" :status-map="freightStatusMap" />

                      <!-- Order link -->
                      <router-link
                        v-if="canViewOrderLink && linkedOrder"
                        :to="`/orders/${linkedOrder.id}`"
                      >
                        <Badge variant="info" class="cursor-pointer">
                          <FileText class="mr-1 h-3 w-3" />
                          Заказ #{{ linkedOrder.order_number }}
                        </Badge>
                      </router-link>

                      <!-- Actions dropdown -->
                      <DropdownMenu v-if="canEdit || canCancel">
                        <DropdownMenuTrigger as-child>
                          <Button variant="outline" size="icon">
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
                <div class="space-y-3">
                  <div
                    v-for="(point, index) in freightRequest.route.points"
                    :key="index"
                    class="border rounded-lg p-4"
                  >
                    <div class="flex items-start gap-3">
                      <div
                        :class="[
                          'w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium shrink-0',
                          point.is_loading && point.is_unloading ? '' :
                          point.is_loading ? 'bg-primary' :
                          point.is_unloading ? 'bg-success' : 'bg-muted-foreground'
                        ]"
                        :style="point.is_loading && point.is_unloading ? 'background: linear-gradient(to right, hsl(var(--primary)) 50%, hsl(var(--success)) 50%)' : ''"
                      >
                        {{ index + 1 }}
                      </div>
                      <div class="flex-1 min-w-0">
                        <Badge variant="outline" class="mb-2">
                          {{ getPointTypeLabel(point) }}
                        </Badge>
                        <p class="font-medium text-foreground break-words">{{ point.address }}</p>
                        <div class="mt-2 text-sm text-muted-foreground space-y-1">
                          <p>
                            <span class="text-muted-foreground/70">Дата:</span>
                            {{ formatDate(point.date_from) }}
                            <template v-if="point.date_to"> — {{ formatDate(point.date_to) }}</template>
                          </p>
                          <p v-if="point.time_from">
                            <span class="text-muted-foreground/70">Время:</span>
                            {{ point.time_from }}
                            <template v-if="point.time_to"> — {{ point.time_to }}</template>
                          </p>
                          <p v-if="point.contact_name">
                            <span class="text-muted-foreground/70">Контакт:</span>
                            {{ point.contact_name }}
                            <template v-if="point.contact_phone">, {{ point.contact_phone }}</template>
                          </p>
                          <p v-if="point.comment" class="italic break-words">{{ point.comment }}</p>
                        </div>
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
                    <dt class="text-sm text-muted-foreground">Тип груза</dt>
                    <dd class="text-foreground">{{ cargoTypeLabels[freightRequest.cargo.type] }}</dd>
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
                  <div class="sm:col-span-2">
                    <dt class="text-sm text-muted-foreground mb-2">Типы кузова</dt>
                    <dd class="flex flex-wrap gap-2">
                      <Badge
                        v-for="bodyType in freightRequest.vehicle_requirements.body_types"
                        :key="bodyType"
                        variant="secondary"
                      >
                        {{ bodyTypeLabels[bodyType] }}
                      </Badge>
                    </dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.loading_types?.length" class="sm:col-span-2">
                    <dt class="text-sm text-muted-foreground mb-2">Типы загрузки</dt>
                    <dd class="flex flex-wrap gap-2">
                      <Badge
                        v-for="loadingType in freightRequest.vehicle_requirements.loading_types"
                        :key="loadingType"
                        variant="secondary"
                      >
                        {{ loadingTypeLabels[loadingType] }}
                      </Badge>
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
                  <div v-if="freightRequest.vehicle_requirements.requires_adr">
                    <dt class="text-sm text-muted-foreground">ADR</dt>
                    <dd class="text-foreground">Требуется</dd>
                  </div>
                  <div v-if="freightRequest.vehicle_requirements.temperature">
                    <dt class="text-sm text-muted-foreground">Температурный режим</dt>
                    <dd class="text-foreground">
                      от {{ freightRequest.vehicle_requirements.temperature.min }}°C
                      до {{ freightRequest.vehicle_requirements.temperature.max }}°C
                    </dd>
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

            <!-- Offers Section -->
            <Card v-if="visibleOffers.length > 0 || isOwner">
              <CardHeader>
                <CardTitle class="flex items-center gap-2">
                  <Users class="h-5 w-5" />
                  Предложения
                  <Badge v-if="isOwner" variant="secondary">{{ offers.length }}</Badge>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div v-if="visibleOffers.length === 0" class="text-center py-8 text-muted-foreground">
                  Пока нет предложений
                </div>

                <div v-else class="space-y-4">
                  <div
                    v-for="offer in visibleOffers"
                    :key="offer.id"
                    :class="[
                      'border rounded-lg p-4',
                      offer.status === 'selected' ? 'border-info/50 bg-info/5' :
                      offer.status === 'confirmed' ? 'border-success/50 bg-success/5' :
                      'border-border'
                    ]"
                  >
                    <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                      <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-3 mb-2">
                          <span class="text-xl font-semibold text-foreground">
                            {{ formatPrice(offer.price.amount, offer.price.currency) }}
                          </span>
                          <StatusBadge :status="offer.status" :status-map="offerStatusMap" />
                        </div>
                        <div v-if="offer.carrier_org_name || offer.carrier_member_name" class="mb-2 flex flex-wrap items-center gap-x-2">
                          <router-link
                            v-if="isOwner && offer.carrier_org_name"
                            :to="`/organizations/${offer.carrier_org_id}`"
                            class="text-primary hover:underline font-medium truncate"
                          >
                            <Building2 class="inline h-4 w-4 mr-1" />
                            {{ offer.carrier_org_name }}
                          </router-link>
                          <span v-if="isOwner && offer.carrier_org_name && offer.carrier_member_name" class="text-muted-foreground">•</span>
                          <router-link
                            v-if="offer.carrier_member_name"
                            :to="`/members/${offer.carrier_member_id}`"
                            class="text-primary hover:underline truncate"
                          >
                            {{ offer.carrier_member_name }}
                          </router-link>
                        </div>
                        <div class="text-sm text-muted-foreground space-y-1">
                          <p>
                            <span class="text-muted-foreground/70">НДС:</span>
                            {{ vatTypeLabels[offer.vat_type] }}
                          </p>
                          <p>
                            <span class="text-muted-foreground/70">Способ оплаты:</span>
                            {{ paymentMethodLabels[offer.payment_method] }}
                          </p>
                          <p v-if="offer.comment" class="break-words">
                            <span class="text-muted-foreground/70">Комментарий:</span>
                            {{ offer.comment }}
                          </p>
                          <p class="text-xs flex items-center gap-1">
                            <Clock class="h-3 w-3" />
                            {{ formatDateTime(offer.created_at) }}
                          </p>
                        </div>
                      </div>

                      <!-- Owner/Admin actions -->
                      <div v-if="canManageOffers && offer.status === 'pending' && freightRequest.status === 'published'" class="flex gap-2 shrink-0">
                        <Button
                          size="sm"
                          :disabled="actionLoading"
                          @click="handleSelectOffer(offer.id)"
                        >
                          <Check class="mr-1 h-4 w-4" />
                          Выбрать
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          :disabled="actionLoading"
                          @click="openRejectModal(offer.id)"
                        >
                          Отклонить
                        </Button>
                      </div>

                      <!-- Carrier actions (own offer) -->
                      <div v-if="!isOwner && offer.carrier_org_id === auth.organizationId" class="flex gap-2 shrink-0">
                        <template v-if="offer.status === 'pending' && permissions.canWithdrawOffer(offer.carrier_org_id, offer.carrier_member_id)">
                          <Button
                            variant="outline"
                            size="sm"
                            :disabled="actionLoading"
                            @click="openWithdrawModal(offer.id)"
                          >
                            Отозвать
                          </Button>
                        </template>
                        <template v-if="offer.status === 'selected'">
                          <Button
                            size="sm"
                            :disabled="actionLoading"
                            @click="handleConfirmOffer(offer.id)"
                          >
                            <Check class="mr-1 h-4 w-4" />
                            Подтвердить
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            :disabled="actionLoading"
                            @click="handleDeclineOffer(offer.id)"
                          >
                            Отказаться
                          </Button>
                        </template>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            <!-- Make Offer Button -->
            <div v-if="canMakeOffer && !myActiveOffer" class="flex justify-center">
              <Button size="lg" @click="showMakeOfferModal = true">
                <Send class="mr-2 h-5 w-5" />
                Сделать предложение
              </Button>
            </div>
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
              />
              <Select v-model="offerForm.price.currency">
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
            <Select v-model="offerForm.vat_type">
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
            <Select v-model="offerForm.payment_method">
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
