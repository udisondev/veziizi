<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import type { Component } from 'vue'
import { useAuthStore } from '@/stores/auth'
import {
  offersApi,
  type OutgoingOfferListItem,
  type IncomingOfferListItem,
  type OutgoingOfferListParams,
  type IncomingOfferListParams,
} from '@/api/offers'
import {
  offerStatusLabels,
  paymentMethodOptions,
  vatTypeOptions,
  paymentMethodLabels,
  vatTypeLabels,
  currencyLabels,
  type OfferStatus,
  type PaymentMethod,
  type VatType,
  type Currency,
} from '@/types/freightRequest'
import { offerStatusMap } from '@/constants/statusMaps'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { useBreakpoint } from '@/composables/useBreakpoint'
import { formatDateShort } from '@/utils/formatters'
import { logger } from '@/utils/logger'
import { useNotificationsStore } from '@/stores/notifications'
import { getCategoryByType } from '@/types/notification'
import { eventStream } from '@/services/eventStream'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { TabsSlider, type TabSliderItem } from '@/components/ui/tabs'
import { MultiSelectField } from '@/components/ui/multi-select'
import { RangeInput, ChipButtonGroup } from '@/components/filters'
import { Tooltip } from '@/components/ui/tooltip'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { StatusBadge, LoadingSpinner, EmptyState, ErrorBanner } from '@/components/shared'
import {
  Send,
  Inbox,
  SlidersHorizontal,
  HandCoins,
  Package,
  ArrowRight,
  CalendarDays,
  ArrowDownWideNarrow,
  ArrowUpNarrowWide,
  Building2,
  User,
  Undo2,
  Check,
  XCircle,
  AlertCircle,
  BadgeCheck,
  Hash,
} from 'lucide-vue-next'

const PAGE_SIZE = 20

const router = useRouter()
const auth = useAuthStore()
const { isBelowLg } = useBreakpoint()

// ─── Direction ──────────────────────────────────────────────────────────────

type Direction = 'outgoing' | 'incoming'
const direction = ref<Direction>('outgoing')

const DIRECTION_ITEMS: TabSliderItem[] = [
  { value: 'outgoing', label: 'Исходящие', icon: Send },
  { value: 'incoming', label: 'Входящие', icon: Inbox },
]

// ─── Sort ────────────────────────────────────────────────────────────────────

type SortBy = 'created_at_desc' | 'price_desc' | 'price_asc'
interface SortOption { value: SortBy; label: string; icon: Component }
const SORT_OPTIONS: SortOption[] = [
  { value: 'created_at_desc', label: 'Новые', icon: CalendarDays },
  { value: 'price_desc', label: 'Цена ↓', icon: ArrowDownWideNarrow },
  { value: 'price_asc', label: 'Цена ↑', icon: ArrowUpNarrowWide },
]
const sortBy = ref<SortBy>('created_at_desc')

// ─── Filter options ──────────────────────────────────────────────────────────

const STATUS_OPTIONS = Object.entries(offerStatusLabels).map(([value, label]) => ({
  value: value as OfferStatus,
  label,
}))

const OWNERSHIP_ITEMS: TabSliderItem[] = [
  { value: 'org', label: 'Организации' },
  { value: 'my', label: 'Мои' },
]

// ─── Outgoing filters ────────────────────────────────────────────────────────

const outOwnership = ref<'org' | 'my'>('org')
const outStatuses = ref<OfferStatus[]>([])
const outMinPrice = ref<number | undefined>()
const outMaxPrice = ref<number | undefined>()
const outPaymentMethods = ref<PaymentMethod[]>([])
const outVatTypes = ref<VatType[]>([])

// ─── Incoming filters ────────────────────────────────────────────────────────

const inOwnership = ref<'org' | 'my'>('org')
const inStatuses = ref<OfferStatus[]>([])
const inCarrierOrgName = ref('')
const inRequestNumber = ref<number | undefined>()
const inMinPrice = ref<number | undefined>()
const inMaxPrice = ref<number | undefined>()
const inPaymentMethods = ref<PaymentMethod[]>([])
const inVatTypes = ref<VatType[]>([])

const mobileFiltersOpen = ref(false)

const hasActiveFilters = computed(() => {
  if (direction.value === 'outgoing') {
    return outStatuses.value.length > 0 || outOwnership.value !== 'org' ||
           outMinPrice.value !== undefined || outMaxPrice.value !== undefined ||
           outPaymentMethods.value.length > 0 || outVatTypes.value.length > 0
  }
  return inOwnership.value !== 'org' || inStatuses.value.length > 0 || !!inCarrierOrgName.value ||
         inRequestNumber.value !== undefined || inMinPrice.value !== undefined ||
         inMaxPrice.value !== undefined || inPaymentMethods.value.length > 0 ||
         inVatTypes.value.length > 0
})

function resetFilters() {
  if (direction.value === 'outgoing') {
    outOwnership.value = 'org'
    outStatuses.value = []
    outMinPrice.value = undefined
    outMaxPrice.value = undefined
    outPaymentMethods.value = []
    outVatTypes.value = []
  } else {
    inOwnership.value = 'org'
    inStatuses.value = []
    inCarrierOrgName.value = ''
    inRequestNumber.value = undefined
    inMinPrice.value = undefined
    inMaxPrice.value = undefined
    inPaymentMethods.value = []
    inVatTypes.value = []
  }
}

// ─── Data ────────────────────────────────────────────────────────────────────

const outItems = ref<OutgoingOfferListItem[]>([])
const inItems = ref<IncomingOfferListItem[]>([])
const isLoading = ref(false)
const isLoadingMore = ref(false)
const error = ref<string | null>(null)
const cursor = ref<string | undefined>()
const hasMore = ref(false)
const hasUpdates = ref(false)

const isEmpty = computed(() =>
  !isLoading.value && (direction.value === 'outgoing' ? outItems.value.length === 0 : inItems.value.length === 0)
)
const itemsCount = computed(() =>
  direction.value === 'outgoing' ? outItems.value.length : inItems.value.length
)

function buildOutgoingParams(): OutgoingOfferListParams {
  const p: OutgoingOfferListParams = { limit: PAGE_SIZE, sort_by: sortBy.value }
  if (outStatuses.value.length > 0) p.statuses = outStatuses.value.join(',')
  if (outOwnership.value === 'my' && auth.memberId) p.member_id = auth.memberId
  if (outMinPrice.value !== undefined) p.min_price = outMinPrice.value
  if (outMaxPrice.value !== undefined) p.max_price = outMaxPrice.value
  if (outPaymentMethods.value.length > 0) p.payment_methods = outPaymentMethods.value.join(',')
  if (outVatTypes.value.length > 0) p.vat_types = outVatTypes.value.join(',')
  return p
}

function buildIncomingParams(): IncomingOfferListParams {
  const p: IncomingOfferListParams = { limit: PAGE_SIZE, sort_by: sortBy.value }
  if (inStatuses.value.length > 0) p.statuses = inStatuses.value.join(',')
  if (inOwnership.value === 'my' && auth.memberId) p.member_id = auth.memberId
  if (inCarrierOrgName.value) p.carrier_org_name = inCarrierOrgName.value
  if (inRequestNumber.value) p.freight_request_number = inRequestNumber.value
  if (inMinPrice.value !== undefined) p.min_price = inMinPrice.value
  if (inMaxPrice.value !== undefined) p.max_price = inMaxPrice.value
  if (inPaymentMethods.value.length > 0) p.payment_methods = inPaymentMethods.value.join(',')
  if (inVatTypes.value.length > 0) p.vat_types = inVatTypes.value.join(',')
  return p
}

async function loadItems() {
  isLoading.value = true
  error.value = null
  cursor.value = undefined
  try {
    if (direction.value === 'outgoing') {
      outItems.value = []
      const res = await offersApi.listOutgoing(buildOutgoingParams())
      outItems.value = res.items
      cursor.value = res.next_cursor
      hasMore.value = res.has_more
    } else {
      inItems.value = []
      const res = await offersApi.listIncoming(buildIncomingParams())
      inItems.value = res.items
      cursor.value = res.next_cursor
      hasMore.value = res.has_more
    }
  } catch (e) {
    error.value = 'Не удалось загрузить предложения'
    logger.error('Failed to load offers', e)
  } finally {
    isLoading.value = false
  }
}

async function loadMoreItems() {
  if (!hasMore.value || isLoadingMore.value || !cursor.value) return
  isLoadingMore.value = true
  try {
    if (direction.value === 'outgoing') {
      const params = { ...buildOutgoingParams(), cursor: cursor.value }
      const res = await offersApi.listOutgoing(params)
      outItems.value.push(...res.items)
      cursor.value = res.next_cursor
      hasMore.value = res.has_more
    } else {
      const params = { ...buildIncomingParams(), cursor: cursor.value }
      const res = await offersApi.listIncoming(params)
      inItems.value.push(...res.items)
      cursor.value = res.next_cursor
      hasMore.value = res.has_more
    }
  } catch (e) {
    logger.error('Failed to load more offers', e)
  } finally {
    isLoadingMore.value = false
  }
}

const canLoadMore = computed(() => hasMore.value && !isLoadingMore.value && !!cursor.value)

// Два отдельных sentinel ref для исходящих и входящих списков.
// Активный ref задаётся через computed, чтобы useInfiniteScroll переподключался при смене direction.
const outSentinelRef = ref<HTMLElement | null>(null)
const inSentinelRef = ref<HTMLElement | null>(null)
const activeSentinel = computed(() =>
  direction.value === 'outgoing' ? outSentinelRef.value : inSentinelRef.value
)
const { sentinelRef, reset: resetInfiniteScroll } = useInfiniteScroll(loadMoreItems, {
  threshold: 300,
  enabled: canLoadMore,
})
// Синхронизируем активный sentinel с composable
watch(activeSentinel, (el) => { sentinelRef.value = el }, { immediate: true })

// ─── Actions ─────────────────────────────────────────────────────────────────

type ActionType = 'withdraw' | 'confirm' | 'decline' | 'select' | 'reject' | 'unselect'

const ACTION_CONFIG: Record<ActionType, {
  title: string
  description: string
  btn: string
  variant: 'default' | 'destructive' | 'secondary'
  hasReason: boolean
}> = {
  withdraw: {
    title: 'Отозвать оффер?',
    description: 'Ваш оффер будет отозван и больше не будет виден заказчику.',
    btn: 'Отозвать', variant: 'secondary', hasReason: false,
  },
  confirm: {
    title: 'Подтвердить оффер?',
    description: 'После подтверждения будет создан заказ и вы станете перевозчиком.',
    btn: 'Подтвердить', variant: 'default', hasReason: false,
  },
  decline: {
    title: 'Отказаться от оффера?',
    description: 'Вы отказываетесь от выбранного оффера. Заказчик сможет выбрать другого перевозчика.',
    btn: 'Отказаться', variant: 'destructive', hasReason: true,
  },
  select: {
    title: 'Выбрать перевозчика?',
    description: 'Перевозчик будет уведомлён и сможет подтвердить или отказаться от заказа.',
    btn: 'Выбрать', variant: 'default', hasReason: false,
  },
  reject: {
    title: 'Отклонить оффер?',
    description: 'Оффер будет отклонён. Перевозчик получит уведомление.',
    btn: 'Отклонить', variant: 'destructive', hasReason: true,
  },
  unselect: {
    title: 'Отменить выбор перевозчика?',
    description: 'Перевозчик будет уведомлён. Вы сможете выбрать другого.',
    btn: 'Отменить выбор', variant: 'secondary', hasReason: true,
  },
}

const showModal = ref(false)
const modalAction = ref<{ type: ActionType; freightRequestId: string; offerId: string } | null>(null)
const modalReason = ref('')
const actionLoading = ref<string | null>(null)
const actionError = ref<string | null>(null)

const modalConfig = computed(() => modalAction.value ? ACTION_CONFIG[modalAction.value.type] : null)

function openModal(type: ActionType, freightRequestId: string, offerId: string) {
  modalAction.value = { type, freightRequestId, offerId }
  modalReason.value = ''
  actionError.value = null
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  modalAction.value = null
  modalReason.value = ''
}

async function executeAction() {
  if (!modalAction.value) return
  const { type, freightRequestId, offerId } = modalAction.value
  actionLoading.value = offerId
  actionError.value = null
  try {
    const reason = modalReason.value || undefined
    switch (type) {
      case 'withdraw': await offersApi.withdraw(freightRequestId, offerId); break
      case 'confirm':  await offersApi.confirm(freightRequestId, offerId); break
      case 'decline':  await offersApi.decline(freightRequestId, offerId, reason); break
      case 'select':   await offersApi.select(freightRequestId, offerId); break
      case 'reject':   await offersApi.reject(freightRequestId, offerId, reason); break
      case 'unselect': await offersApi.unselect(freightRequestId, offerId, reason); break
    }
    closeModal()
    await loadItems()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Не удалось выполнить действие'
    logger.error('Failed to perform offer action', e)
  } finally {
    actionLoading.value = null
  }
}

// ─── Formatters ───────────────────────────────────────────────────────────────

function formatPrice(amount?: number, currency?: string): string {
  if (!amount || !currency) return '—'
  const value = (amount / 100).toLocaleString('ru-RU', { minimumFractionDigits: 0, maximumFractionDigits: 2 })
  const symbol = currencyLabels[currency as Currency] || currency
  return `${value} ${symbol}`
}

function formatRoute(origin?: string, destination?: string): string {
  return `${origin || '?'} → ${destination || '?'}`
}

function formatPaymentInfo(paymentMethod?: PaymentMethod, vatType?: VatType): string {
  const parts: string[] = []
  if (paymentMethod) parts.push(paymentMethodLabels[paymentMethod] ?? paymentMethod)
  if (vatType) parts.push(vatTypeLabels[vatType] ?? vatType)
  return parts.join(' · ')
}

// ─── Watches ─────────────────────────────────────────────────────────────────

// Флаг, блокирующий debounce-watch пока direction watch выполняет сброс + загрузку.
let directionChanging = false

watch(direction, async () => {
  directionChanging = true
  sortBy.value = 'created_at_desc'
  mobileFiltersOpen.value = false
  if (debounceTimer) { clearTimeout(debounceTimer); debounceTimer = null }
  try {
    await loadItems()
    await nextTick()
    resetInfiniteScroll()
  } finally {
    directionChanging = false
  }
})

let debounceTimer: ReturnType<typeof setTimeout> | null = null
watch(
  [
    sortBy,
    outOwnership, outStatuses, outMinPrice, outMaxPrice, outPaymentMethods, outVatTypes,
    inOwnership, inStatuses, inCarrierOrgName, inRequestNumber, inMinPrice, inMaxPrice, inPaymentMethods, inVatTypes,
  ],
  () => {
    if (directionChanging) return
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      await loadItems()
      await nextTick()
      resetInfiniteScroll()
    }, 300)
  },
  { deep: true }
)

function handleFreightRequestEvent() {
  if (itemsCount.value > 0) hasUpdates.value = true
}

async function applyUpdates() {
  hasUpdates.value = false
  await loadItems()
  await nextTick()
  resetInfiniteScroll()
}

onUnmounted(() => {
  if (debounceTimer) clearTimeout(debounceTimer)
  eventStream.off('freight_request', handleFreightRequestEvent)
  eventStream.offConnected(handleFreightRequestEvent)
})

const notificationsStore = useNotificationsStore()
const lastOfferNotificationId = computed(() =>
  notificationsStore.notifications.find(n => getCategoryByType(n.notification_type) === 'offers')?.id ?? null
)
watch(lastOfferNotificationId, (newId, oldId) => {
  if (newId && newId !== oldId) loadItems()
})

onMounted(() => {
  loadItems()
  eventStream.on('freight_request', handleFreightRequestEvent)
  eventStream.onConnected(handleFreightRequestEvent)
})
</script>

<template>
  <div class="max-w-7xl mx-auto pt-4 pb-6 px-4 sm:px-6 lg:px-8">
    <!-- 2-column layout -->
    <div class="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-6 items-start">

      <!-- Filters sidebar -->
      <div class="lg:sticky lg:top-16">

        <!-- Уровень 1: направление (навигация) -->
        <div class="mb-3">
          <TabsSlider v-model="direction" :items="DIRECTION_ITEMS" no-overflow stretch />
        </div>

        <!-- Уровень 2: принадлежность (контекст, всегда видна) -->
        <div class="pb-3 mb-3 border-b">
          <TabsSlider
            v-if="direction === 'outgoing'"
            v-model="outOwnership"
            :items="OWNERSHIP_ITEMS"
            no-overflow
            stretch
          />
          <TabsSlider
            v-else
            v-model="inOwnership"
            :items="OWNERSHIP_ITEMS"
            no-overflow
            stretch
          />
        </div>

        <!-- Mobile: кнопка раскрытия фильтров -->
        <div class="lg:hidden mb-2">
          <Button variant="outline" class="w-full" @click="mobileFiltersOpen = !mobileFiltersOpen">
            <SlidersHorizontal class="h-4 w-4 mr-2" />
            Фильтры
            <span
              v-if="hasActiveFilters"
              class="ml-2 inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium"
            >
              ●
            </span>
          </Button>
        </div>

        <!-- Уровень 3: детальные фильтры -->
        <div :class="['accordion-grid filters-accordion', mobileFiltersOpen ? 'is-open' : '']">
          <div class="overflow-hidden lg:overflow-visible">
            <div class="space-y-4 pb-2">

              <!-- Исходящие -->
              <template v-if="direction === 'outgoing'">
                <MultiSelectField
                  v-model="outStatuses"
                  :options="STATUS_OPTIONS"
                  placeholder="Все статусы"
                  sheet-label="Статус"
                  search-placeholder="Поиск статуса..."
                />
                <RangeInput
                  :min-value="outMinPrice"
                  :max-value="outMaxPrice"
                  label="Ставка, руб."
                  :min="0"
                  :step="1000"
                  @update:min-value="outMinPrice = $event"
                  @update:max-value="outMaxPrice = $event"
                />
                <ChipButtonGroup
                  v-model="outPaymentMethods"
                  :options="paymentMethodOptions"
                  label="Способ оплаты"
                />
                <ChipButtonGroup
                  v-model="outVatTypes"
                  :options="vatTypeOptions"
                  label="НДС"
                />
              </template>

              <!-- Входящие -->
              <template v-else>
                <MultiSelectField
                  v-model="inStatuses"
                  :options="STATUS_OPTIONS"
                  placeholder="Все статусы"
                  sheet-label="Статус"
                  search-placeholder="Поиск статуса..."
                />
                <div>
                  <Label>Перевозчик</Label>
                  <Input v-model="inCarrierOrgName" placeholder="Название организации" class="mt-2" />
                </div>
                <div>
                  <Label>Номер заявки</Label>
                  <Input
                    type="number"
                    :model-value="inRequestNumber"
                    placeholder="Например, 42"
                    class="mt-2"
                    @update:model-value="(v: string | number) => inRequestNumber = v === '' ? undefined : Number(v)"
                  />
                </div>
                <RangeInput
                  :min-value="inMinPrice"
                  :max-value="inMaxPrice"
                  label="Ставка, руб."
                  :min="0"
                  :step="1000"
                  @update:min-value="inMinPrice = $event"
                  @update:max-value="inMaxPrice = $event"
                />
                <ChipButtonGroup
                  v-model="inPaymentMethods"
                  :options="paymentMethodOptions"
                  label="Способ оплаты"
                />
                <ChipButtonGroup
                  v-model="inVatTypes"
                  :options="vatTypeOptions"
                  label="НДС"
                />
              </template>

              <div v-if="hasActiveFilters" class="flex justify-center pt-1">
                <Button variant="ghost" size="sm" @click="resetFilters">Сбросить фильтры</Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- List column -->
      <div>
        <!-- Sort bar -->
        <div class="flex gap-1 mb-4 pb-4 border-b">
          <template v-if="!isBelowLg">
            <Tooltip v-for="opt in SORT_OPTIONS" :key="opt.value" :text="opt.label" side="top">
              <button
                type="button"
                class="flex justify-center rounded-md border p-1.5 transition-colors"
                :class="sortBy === opt.value ? 'bg-primary border-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-muted/50'"
                @click="sortBy = opt.value"
              >
                <component :is="opt.icon" class="h-3.5 w-3.5" />
              </button>
            </Tooltip>
          </template>
          <template v-else>
            <button
              v-for="opt in SORT_OPTIONS"
              :key="opt.value"
              type="button"
              class="rounded-md border px-2 py-1 text-xs font-medium transition-colors"
              :class="sortBy === opt.value ? 'bg-primary border-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-muted/50'"
              @click="sortBy = opt.value"
            >
              {{ opt.label }}
            </button>
          </template>
        </div>

        <!-- Action error (outside modal) -->
        <div
          v-if="actionError && !showModal"
          class="mb-4 flex items-center justify-between gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
        >
          <div class="flex items-center gap-2">
            <AlertCircle class="h-4 w-4 shrink-0" />
            {{ actionError }}
          </div>
          <Button variant="ghost" size="sm" @click="actionError = null">Закрыть</Button>
        </div>

        <!-- SSE: баннер новых данных -->
        <div
          v-if="hasUpdates && !isLoading"
          class="mb-4 flex items-center justify-between rounded-lg border border-primary/30 bg-primary/5 px-4 py-2.5 text-sm"
        >
          <span class="text-foreground">Список обновился — есть новые данные</span>
          <Button size="sm" variant="outline" @click="applyUpdates">Обновить</Button>
        </div>

        <LoadingSpinner v-if="isLoading" text="Загрузка предложений..." />

        <ErrorBanner v-else-if="error" :message="error" @retry="loadItems" />

        <EmptyState
          v-else-if="isEmpty"
          :icon="HandCoins"
          :title="direction === 'outgoing' ? 'Исходящих предложений нет' : 'Входящих предложений нет'"
          :description="hasActiveFilters
            ? 'Нет предложений по заданным фильтрам'
            : direction === 'outgoing'
              ? 'Вы ещё не делали предложений на заявки'
              : 'На ваши заявки пока не поступало предложений'"
        >
          <template v-if="direction === 'outgoing' && !hasActiveFilters" #action>
            <Button as-child>
              <router-link to="/requests">
                <Package class="mr-2 h-4 w-4" />
                Найти заявки
              </router-link>
            </Button>
          </template>
        </EmptyState>

        <!-- Outgoing list -->
        <div v-else-if="direction === 'outgoing'" class="space-y-3">
          <Card
            v-for="item in outItems"
            :key="item.id"
            interactive
            @click="router.push(`/freight-requests/${item.freight_request_id}?tab=offers`)"
          >
            <CardContent class="p-4">
              <!-- Header: status + price -->
              <div class="flex items-start justify-between gap-3 mb-3">
                <div class="flex items-center gap-2 flex-wrap">
                  <StatusBadge :status="item.status" :status-map="offerStatusMap" />
                  <span v-if="item.freight_request_number" class="flex items-center gap-1 text-xs text-muted-foreground">
                    <Hash class="h-3 w-3" />{{ item.freight_request_number }}
                  </span>
                </div>
                <span v-if="item.price_amount" class="text-base font-bold text-emerald-600 dark:text-emerald-400 shrink-0">
                  {{ formatPrice(item.price_amount, item.price_currency) }}
                </span>
              </div>

              <!-- Route -->
              <div class="text-sm font-medium mb-2 break-words">
                {{ formatRoute(item.origin_address, item.destination_address) }}
              </div>

              <!-- Meta -->
              <div class="flex items-center gap-3 flex-wrap text-xs text-muted-foreground mb-3">
                <span v-if="item.customer_org_name" class="flex items-center gap-1">
                  <Building2 class="h-3.5 w-3.5 shrink-0" />
                  {{ item.customer_org_name }}
                </span>
                <span v-if="item.payment_method || item.vat_type">
                  {{ formatPaymentInfo(item.payment_method, item.vat_type) }}
                </span>
                <span class="flex items-center gap-1">
                  <CalendarDays class="h-3.5 w-3.5 shrink-0" />
                  {{ formatDateShort(item.created_at) }}
                </span>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-2 flex-wrap pt-2.5 border-t" @click.stop>
                <Button
                  v-if="item.status === 'pending'"
                  variant="secondary" size="sm"
                  :disabled="actionLoading === item.id"
                  @click="openModal('withdraw', item.freight_request_id, item.id)"
                >
                  <Undo2 class="mr-1 h-3.5 w-3.5" /> Отозвать
                </Button>
                <Button
                  v-if="item.status === 'selected'"
                  size="sm"
                  :disabled="actionLoading === item.id"
                  @click="openModal('confirm', item.freight_request_id, item.id)"
                >
                  <Check class="mr-1 h-3.5 w-3.5" /> Подтвердить
                </Button>
                <Button
                  v-if="item.status === 'selected'"
                  variant="destructive" size="sm"
                  :disabled="actionLoading === item.id"
                  @click="openModal('decline', item.freight_request_id, item.id)"
                >
                  <XCircle class="mr-1 h-3.5 w-3.5" /> Отказаться
                </Button>
                <Button
                  variant="ghost" size="sm" class="ml-auto"
                  @click="router.push(`/freight-requests/${item.freight_request_id}?tab=details`)"
                >
                  К заявке <ArrowRight class="ml-1 h-3.5 w-3.5" />
                </Button>
              </div>
            </CardContent>
          </Card>

          <div ref="outSentinelRef" class="h-16 flex items-center justify-center">
            <LoadingSpinner v-if="isLoadingMore" text="Загрузка..." />
            <span v-else-if="!hasMore && itemsCount > 0" class="text-sm text-muted-foreground">
              Все предложения загружены ({{ itemsCount }})
            </span>
          </div>
        </div>

        <!-- Incoming list -->
        <div v-else class="space-y-3">
          <Card
            v-for="item in inItems"
            :key="item.id"
            interactive
            @click="router.push(`/freight-requests/${item.freight_request_id}?tab=offers`)"
          >
            <CardContent class="p-4">
              <!-- Header: status + price -->
              <div class="flex items-start justify-between gap-3 mb-3">
                <div class="flex items-center gap-2 flex-wrap">
                  <StatusBadge :status="item.status" :status-map="offerStatusMap" />
                  <span v-if="item.freight_request_number" class="flex items-center gap-1 text-xs text-muted-foreground">
                    <Hash class="h-3 w-3" />{{ item.freight_request_number }}
                  </span>
                </div>
                <span class="text-base font-bold text-emerald-600 dark:text-emerald-400 shrink-0">
                  {{ formatPrice(item.price_amount, item.price_currency) }}
                </span>
              </div>

              <!-- Carrier -->
              <div class="flex items-center gap-3 mb-2 text-sm flex-wrap">
                <span v-if="item.carrier_org_name" class="flex items-center gap-1 font-medium">
                  <Building2 class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                  {{ item.carrier_org_name }}
                </span>
                <span v-if="item.carrier_member_name" class="flex items-center gap-1 text-muted-foreground">
                  <User class="h-3.5 w-3.5 shrink-0" />
                  {{ item.carrier_member_name }}
                </span>
              </div>

              <!-- Route + meta -->
              <div class="text-sm text-muted-foreground mb-2 break-words">
                {{ formatRoute(item.origin_address, item.destination_address) }}
              </div>

              <!-- Details row -->
              <div class="flex items-center gap-3 flex-wrap text-xs text-muted-foreground mb-3">
                <span v-if="item.payment_method || item.vat_type">
                  {{ formatPaymentInfo(item.payment_method, item.vat_type) }}
                </span>
                <span class="flex items-center gap-1">
                  <CalendarDays class="h-3.5 w-3.5 shrink-0" />
                  {{ formatDateShort(item.created_at) }}
                </span>
                <Badge v-if="item.comment" variant="secondary" class="text-xs">Есть комментарий</Badge>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-2 flex-wrap pt-2.5 border-t" @click.stop>
                <Button
                  v-if="item.status === 'pending'"
                  size="sm"
                  :disabled="actionLoading === item.id"
                  @click="openModal('select', item.freight_request_id, item.id)"
                >
                  <BadgeCheck class="mr-1 h-3.5 w-3.5" /> Выбрать
                </Button>
                <Button
                  v-if="item.status === 'pending'"
                  variant="outline" size="sm"
                  :disabled="actionLoading === item.id"
                  @click="openModal('reject', item.freight_request_id, item.id)"
                >
                  <XCircle class="mr-1 h-3.5 w-3.5" /> Отклонить
                </Button>
                <Button
                  v-if="item.status === 'selected'"
                  variant="secondary" size="sm"
                  :disabled="actionLoading === item.id"
                  @click="openModal('unselect', item.freight_request_id, item.id)"
                >
                  <Undo2 class="mr-1 h-3.5 w-3.5" /> Отменить выбор
                </Button>
                <Button
                  variant="ghost" size="sm" class="ml-auto"
                  @click="router.push(`/freight-requests/${item.freight_request_id}?tab=offers`)"
                >
                  К заявке <ArrowRight class="ml-1 h-3.5 w-3.5" />
                </Button>
              </div>
            </CardContent>
          </Card>

          <div ref="inSentinelRef" class="h-16 flex items-center justify-center">
            <LoadingSpinner v-if="isLoadingMore" text="Загрузка..." />
            <span v-else-if="!hasMore && itemsCount > 0" class="text-sm text-muted-foreground">
              Все предложения загружены ({{ itemsCount }})
            </span>
          </div>
        </div>

      </div>
    </div>

    <!-- Confirm modal -->
    <Dialog v-model:open="showModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ modalConfig?.title }}</DialogTitle>
          <DialogDescription>{{ modalConfig?.description }}</DialogDescription>
        </DialogHeader>

        <div v-if="modalConfig?.hasReason" class="space-y-2">
          <Label>Причина (необязательно)</Label>
          <Textarea v-model="modalReason" rows="2" placeholder="Укажите причину..." />
        </div>

        <div
          v-if="actionError"
          class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
        >
          <AlertCircle class="h-4 w-4 shrink-0" />
          {{ actionError }}
        </div>

        <DialogFooter>
          <Button variant="outline" :disabled="actionLoading !== null" @click="closeModal">
            Отмена
          </Button>
          <Button
            :variant="modalConfig?.variant ?? 'default'"
            :disabled="actionLoading !== null"
            @click="executeAction"
          >
            {{ actionLoading ? 'Выполнение...' : modalConfig?.btn }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
