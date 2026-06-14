<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { useFreightFiltersStore, buildRouteParams } from '@/stores/freightFilters'
import { usePermissions } from '@/composables/usePermissions'
import { useBreakpoint } from '@/composables/useBreakpoint'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { freightRequestsApi, type FreightRequestListParams } from '@/api/freightRequests'
import { eventStream } from '@/services/eventStream'
import type { FreightRequestListItem, CarrierInviteItem } from '@/types/freightRequest'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  currencyLabels,
} from '@/types/freightRequest'
import { freightRequestStatusMap } from '@/constants/statusMaps'
import { formatDateShort, formatWeight } from '@/utils/formatters'
import { logger } from '@/utils/logger'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import {
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
} from '@/components/shared'
import { FreightFiltersForm } from '@/components/filters'
import { Tooltip } from '@/components/ui/tooltip'
import SelectField from '@/components/ui/select-field/SelectField.vue'
import { Clock, Building2, Package, SlidersHorizontal, Weight, Truck, CalendarDays, Timer, X, ArrowDownWideNarrow, CalendarClock, Users, ArrowDownNarrowWide, Mail } from 'lucide-vue-next'
import type { Component } from 'vue'

type SortBy = 'created_at_desc' | 'expires_at_asc' | 'price_desc' | 'weight_desc' | 'loading_date_asc' | 'offers_count_asc'

interface SortOption { value: SortBy; label: string; icon: Component }

const SORT_OPTIONS: SortOption[] = [
  { value: 'created_at_desc', label: 'Новые', icon: CalendarDays },
  { value: 'expires_at_asc', label: 'Истекают', icon: Timer },
  { value: 'price_desc', label: 'Цена', icon: ArrowDownWideNarrow },
  { value: 'weight_desc', label: 'Вес', icon: Weight },
  { value: 'loading_date_asc', label: 'Дата загрузки', icon: CalendarClock },
  { value: 'offers_count_asc', label: 'Меньше офферов', icon: Users },
]

const SORT_SELECT_OPTIONS = SORT_OPTIONS.map(({ value, label }) => ({ value, label }))

const PAGE_SIZE = 20

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const onboarding = useOnboardingStore()
const filtersStore = useFreightFiltersStore()
const { canCreateFreightRequest, canCreateOffer } = usePermissions()
const { isBelowLg } = useBreakpoint()

const emit = defineEmits<{
  'go-to-new': []
}>()

const sortBy = ref<SortBy>('created_at_desc')

const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const isLoadingMore = ref(false)
const error = ref<string | null>(null)
const mobileFiltersOpen = ref(false)
const hasUpdates = ref(false)

const {
  ownershipFilter,
  orgINNFilter,
  requestNumber,
  statuses,
  routePoints,
  minWeight,
  maxWeight,
  minPrice,
  maxPrice,
  minVolume,
  maxVolume,
  vehicleSubTypes,
  paymentMethods,
  paymentTerms,
  vatTypes,
  hasPendingOffers,
  hasActiveFilters,
  cursor,
  hasMore,
} = storeToRefs(filtersStore)

function buildParams(): FreightRequestListParams {
  const params: FreightRequestListParams = { limit: PAGE_SIZE }

  if (statuses.value.length > 0) {
    params.statuses = statuses.value.join(',')
  }

  if (ownershipFilter.value === 'my_org' && auth.organizationId) {
    params.customer_org_id = auth.organizationId
  } else if (ownershipFilter.value === 'my_as_carrier' && auth.organizationId) {
    params.carrier_org_id = auth.organizationId
  } else if (ownershipFilter.value === 'my' && auth.memberId) {
    params.member_id = auth.memberId
  }

  if (orgINNFilter.value) params.org_inn = orgINNFilter.value
  if (requestNumber.value) params.request_number = requestNumber.value

  // 0/NaN/undefined → фильтр выключен; иначе нижняя граница 0 исключит NULL-значения.
  const positive = (v: number | undefined): v is number => typeof v === 'number' && Number.isFinite(v) && v > 0
  if (positive(minWeight.value)) params.min_weight = minWeight.value
  if (positive(maxWeight.value)) params.max_weight = maxWeight.value
  if (positive(minPrice.value)) params.min_price = minPrice.value
  if (positive(maxPrice.value)) params.max_price = maxPrice.value
  if (positive(minVolume.value)) params.min_volume = minVolume.value
  if (positive(maxVolume.value)) params.max_volume = maxVolume.value

  if (vehicleSubTypes.value.length > 0) params.vehicle_subtypes = vehicleSubTypes.value.join(',')
  if (paymentMethods.value.length > 0) params.payment_methods = paymentMethods.value.join(',')
  if (paymentTerms.value.length > 0) params.payment_terms = paymentTerms.value.join(',')
  if (vatTypes.value.length > 0) params.vat_types = vatTypes.value.join(',')
  if (hasPendingOffers.value) params.has_pending_offers = true

  Object.assign(params, buildRouteParams(routePoints.value))

  params.sort_by = sortBy.value

  return params
}

async function loadItems() {
  hasUpdates.value = false
  isLoading.value = true
  error.value = null
  filtersStore.resetPagination()
  items.value = []

  try {
    const params = buildParams()
    const response = await freightRequestsApi.list(params)
    items.value = response.items ?? []
    filtersStore.setCursor(response.next_cursor)
    filtersStore.setHasMore(response.has_more)
  } catch (e) {
    error.value = 'Не удалось загрузить заявки'
    logger.error('Failed to load freight requests', e)
  } finally {
    isLoading.value = false
  }
}

async function loadMoreItems() {
  if (!hasMore.value || isLoadingMore.value || !cursor.value) return

  isLoadingMore.value = true
  try {
    const params = buildParams()
    params.cursor = cursor.value
    const response = await freightRequestsApi.list(params)
    items.value.push(...(response.items ?? []))
    filtersStore.setCursor(response.next_cursor)
    filtersStore.setHasMore(response.has_more)
  } catch (e) {
    logger.error('Failed to load more freight requests', e)
  } finally {
    isLoadingMore.value = false
  }
}

const canLoadMore = computed(() => hasMore.value && !isLoadingMore.value && cursor.value !== undefined)
const { sentinelRef, reset: resetInfiniteScroll } = useInfiniteScroll(loadMoreItems, {
  threshold: 300,
  enabled: canLoadMore,
})

void sentinelRef

// ─── Invitations ─────────────────────────────────────────────────────────────

const showInvitations = ref(false)
const invItems = ref<CarrierInviteItem[]>([])
const invLoading = ref(false)
const invLoadingMore = ref(false)
const invError = ref<string | null>(null)
const invCursor = ref<string | undefined>(undefined)
const invHasMore = ref(false)
let invLoadSeq = 0

async function loadInvitations() {
  const seq = ++invLoadSeq
  invLoading.value = true
  invError.value = null
  invItems.value = []
  invCursor.value = undefined
  invHasMore.value = false
  try {
    const res = await freightRequestsApi.listCarrierInvitations({ limit: PAGE_SIZE })
    if (seq !== invLoadSeq) return
    invItems.value = res.items ?? []
    invCursor.value = res.next_cursor
    invHasMore.value = res.has_more
  } catch (e) {
    if (seq !== invLoadSeq) return
    invError.value = 'Не удалось загрузить приглашения'
    logger.error('Failed to load invitations', e)
  } finally {
    if (seq === invLoadSeq) invLoading.value = false
  }
}

async function loadMoreInvitations() {
  if (!invHasMore.value || invLoadingMore.value || !invCursor.value) return
  invLoadingMore.value = true
  try {
    const res = await freightRequestsApi.listCarrierInvitations({ limit: PAGE_SIZE, cursor: invCursor.value })
    invItems.value.push(...(res.items ?? []))
    invCursor.value = res.next_cursor
    invHasMore.value = res.has_more
  } catch (e) {
    logger.error('Failed to load more invitations', e)
  } finally {
    invLoadingMore.value = false
  }
}

const canLoadMoreInv = computed(() => invHasMore.value && !invLoadingMore.value && invCursor.value !== undefined)
const { sentinelRef: invSentinelRef, reset: resetInvScroll } = useInfiniteScroll(loadMoreInvitations, {
  threshold: 300,
  enabled: canLoadMoreInv,
})

void invSentinelRef

watch(showInvitations, (val) => {
  if (val) {
    loadInvitations()
    resetInvScroll()
  } else {
    invItems.value = []
    invCursor.value = undefined
    invHasMore.value = false
  }
})

const filteredInvItems = computed(() => {
  let result = invItems.value

  if (requestNumber.value) {
    result = result.filter(item => item.request_number === requestNumber.value)
  }

  const activePoints = routePoints.value.filter(rp => rp.cityName)
  if (activePoints.length > 0) {
    result = result.filter(item => {
      const haystack = [item.origin_address, item.destination_address]
        .filter(Boolean).join(' ').toLowerCase()
      return activePoints.every(rp => {
        const needle = (rp.cityName ?? '').toLowerCase()
        return !needle || haystack.includes(needle)
      })
    })
  }

  return result
})

function goToDetail(id: string) {
  router.push(`/freight-requests/${id}`)
}

function formatPrice(amount?: number, currency?: string): string {
  if (!amount || !currency) return '—'
  const value = amount / 100
  const symbol = currencyLabels[currency as keyof typeof currencyLabels] || currency
  return `${value.toLocaleString('ru-RU')} ${symbol}`
}

function formatWeightDisplay(weight?: number): string {
  if (!weight) return '—'
  return formatWeight(weight * 1000)
}

function getTransitPointsCount(item: FreightRequestListItem): number {
  if (!item.route?.points || item.route.points.length <= 2) return 0
  return item.route.points.length - 2
}

function formatVehicleType(type?: string, subtype?: string): string {
  if (!type) return '—'
  const typeLabel = vehicleTypeLabels[type as keyof typeof vehicleTypeLabels] || type
  if (subtype) {
    const subtypeLabel = vehicleSubTypeLabels[subtype as keyof typeof vehicleSubTypeLabels] || subtype
    return `${typeLabel} (${subtypeLabel})`
  }
  return typeLabel
}

function isExpiringSoon(expiresAt: string): boolean {
  const expires = new Date(expiresAt)
  const now = new Date()
  const diffDays = (expires.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  return diffDays > 0 && diffDays <= 3
}

function sandboxRequestToListItem(req: NonNullable<typeof onboarding.sandboxCreatedRequest>): FreightRequestListItem {
  return {
    id: req.id,
    request_number: 1,
    customer_org_id: 'sandbox-org-1',
    status: req.status as 'published',
    origin_address: req.origin_address,
    destination_address: req.destination_address,
    cargo_weight: req.cargo_weight / 1000,
    vehicle_type: req.vehicle_type,
    vehicle_subtype: req.vehicle_subtype,
    price_amount: req.price_amount,
    price_currency: req.price_currency,
    created_at: req.created_at,
    expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
    customer_org_name: 'Моя организация',
  }
}

const displayItems = computed<FreightRequestListItem[]>(() => {
  if (onboarding.isSandboxMode && onboarding.sandboxCreatedRequest) {
    return [sandboxRequestToListItem(onboarding.sandboxCreatedRequest)]
  }
  return items.value
})

let filterDebounceTimer: ReturnType<typeof setTimeout> | null = null
const FILTER_DEBOUNCE_MS = 300

watch(
  [
    ownershipFilter, orgINNFilter, requestNumber, statuses, routePoints,
    minWeight, maxWeight, minPrice, maxPrice, minVolume, maxVolume,
    vehicleSubTypes, paymentMethods, paymentTerms, vatTypes, hasPendingOffers,
    sortBy,
  ],
  () => {
    if (filterDebounceTimer) clearTimeout(filterDebounceTimer)
    filterDebounceTimer = setTimeout(async () => {
      await loadItems()
      await nextTick()
      resetInfiniteScroll()
    }, FILTER_DEBOUNCE_MS)
  },
  { deep: true }
)

function handleFreightRequestEvent() {
  if (showInvitations.value) {
    loadInvitations()
    return
  }
  if (items.value.length > 0) hasUpdates.value = true
}

async function applyUpdates() {
  hasUpdates.value = false
  await loadItems()
  await nextTick()
  resetInfiniteScroll()
}

onUnmounted(() => {
  if (filterDebounceTimer) clearTimeout(filterDebounceTimer)
  eventStream.off('freight_request', handleFreightRequestEvent)
  eventStream.offConnected(handleFreightRequestEvent)
})

onMounted(() => {
  if (route.query.invitations === '1') {
    showInvitations.value = true
  }
  loadItems()
  eventStream.on('freight_request', handleFreightRequestEvent)
  eventStream.onConnected(handleFreightRequestEvent)
})
</script>

<template>
  <div>
    <!-- Toolbar -->
    <div class="flex items-center justify-between mb-4 gap-2">
      <div class="flex items-center gap-2 flex-wrap">
        <Badge
          v-if="hasPendingOffers"
          variant="secondary"
          class="cursor-pointer gap-1 pr-1.5"
          @click="filtersStore.setFilters({ hasPendingOffers: false })"
        >
          С предложениями
          <X class="h-3 w-3" />
        </Badge>
        <Button
          v-if="hasActiveFilters"
          variant="ghost"
          size="sm"
          @click="filtersStore.resetFilters"
        >
          Сбросить всё
        </Button>
      </div>
    </div>

    <!-- 2-column layout: filters | list -->
    <div class="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-6 items-start">

      <!-- Filters -->
      <div class="lg:sticky lg:top-16">
        <slot name="sidebar-header" />

        <!-- Чекбокс: показать приглашения -->
        <div class="flex items-center gap-2 pb-3 mb-2 border-b">
          <Checkbox
            id="show-invitations"
            :checked="showInvitations"
            @update:checked="showInvitations = ($event as boolean)"
          />
          <Label for="show-invitations" class="text-sm cursor-pointer mb-0 select-none">
            Приглашения
          </Label>
        </div>

        <!-- Mobile filter toggle (под табами) -->
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

        <div :class="['accordion-grid filters-accordion', mobileFiltersOpen ? 'is-open' : '']">
          <div class="overflow-hidden lg:overflow-visible">
            <div class="space-y-2 pb-2">
              <FreightFiltersForm
                compact
                :route-points="routePoints"
                :min-weight="minWeight"
                :max-weight="maxWeight"
                :min-price="minPrice"
                :max-price="maxPrice"
                :min-volume="minVolume"
                :max-volume="maxVolume"
                :vehicle-sub-types="vehicleSubTypes"
                :payment-methods="paymentMethods"
                :payment-terms="paymentTerms"
                :vat-types="vatTypes"
                :show-cargo="!showInvitations"
                :show-vehicle="!showInvitations"
                :show-payment="!showInvitations"
                :show-ownership="!showInvitations"
                :ownership="ownershipFilter"
                :show-i-n-n="!showInvitations"
                :org-i-n-n="orgINNFilter"
                show-request-number
                :request-number="requestNumber"
                :show-statuses="!showInvitations"
                :statuses="statuses"
                data-tutorial="filters-btn"
                @add-route-point="filtersStore.addRoutePoint"
                @remove-route-point="filtersStore.removeRoutePoint"
                @update-route-point="filtersStore.updateRoutePoint"
                @reorder-route-points="filtersStore.reorderRoutePoints"
                @update:min-weight="filtersStore.setFilters({ minWeight: $event })"
                @update:max-weight="filtersStore.setFilters({ maxWeight: $event })"
                @update:min-price="filtersStore.setFilters({ minPrice: $event })"
                @update:max-price="filtersStore.setFilters({ maxPrice: $event })"
                @update:min-volume="filtersStore.setFilters({ minVolume: $event })"
                @update:max-volume="filtersStore.setFilters({ maxVolume: $event })"
                @update:vehicle-sub-types="filtersStore.setFilters({ vehicleSubTypes: $event })"
                @update:payment-methods="filtersStore.setFilters({ paymentMethods: $event })"
                @update:payment-terms="filtersStore.setFilters({ paymentTerms: $event })"
                @update:vat-types="filtersStore.setFilters({ vatTypes: $event })"
                @update:ownership="filtersStore.setFilters({ ownership: $event })"
                @update:org-i-n-n="filtersStore.setFilters({ orgINN: $event })"
                @update:request-number="filtersStore.setFilters({ requestNumber: $event })"
                @update:statuses="filtersStore.setFilters({ statuses: $event })"
              />
              <div v-if="hasActiveFilters" class="flex justify-center pt-1">
                <Button variant="ghost" size="sm" @click="filtersStore.resetFilters">
                  Сбросить фильтры
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- List -->
      <div>

        <!-- ── Режим приглашений ───────────────────────────────────────────── -->
        <template v-if="showInvitations">
          <LoadingSpinner v-if="invLoading" text="Загрузка приглашений..." />
          <ErrorBanner v-else-if="invError" :message="invError" @retry="loadInvitations" />
          <template v-else>
          <EmptyState
            v-if="filteredInvItems.length === 0 && !invHasMore"
            :icon="Mail"
            title="Приглашений нет"
            :description="invItems.length > 0 ? 'Нет приглашений по заданным фильтрам' : 'Никто ещё не приглашал вас сделать предложение'"
          />
          <div v-else-if="filteredInvItems.length > 0" class="space-y-3">
            <Card
              v-for="item in filteredInvItems"
              :key="item.id"
              interactive
              @click="router.push(`/freight-requests/${item.freight_request_id}`)"
            >
              <CardContent class="p-4">
                <!-- Header: number -->
                <div class="flex items-center gap-2 mb-3">
                  <span class="text-sm font-medium text-muted-foreground">#{{ item.request_number }}</span>
                </div>

                <!-- Route -->
                <div class="flex items-stretch gap-3 mb-3">
                  <div class="flex flex-col items-center py-2">
                    <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
                    <div class="w-px flex-1 min-h-[18px] border-l border-dashed border-muted-foreground/40" />
                    <div class="w-2 h-2 rounded-full bg-muted-foreground/50 shrink-0" />
                  </div>
                  <div class="flex flex-col justify-between flex-1 min-w-0">
                    <span class="text-base font-semibold text-foreground truncate">{{ item.origin_address || '—' }}</span>
                    <span class="text-base text-muted-foreground truncate">{{ item.destination_address || '—' }}</span>
                  </div>
                </div>

                <!-- Vehicle -->
                <div class="flex items-center gap-1.5 text-xs text-muted-foreground mb-3">
                  <Truck class="h-3.5 w-3.5 shrink-0" />
                  <span class="font-medium text-foreground">
                    {{ [item.vehicle_brand, item.vehicle_model].filter(Boolean).join(' ') || 'Транспорт' }}
                  </span>
                  <template v-if="item.vehicle_registration">
                    <span>·</span>
                    <span class="font-mono">{{ item.vehicle_registration }}</span>
                  </template>
                </div>

                <!-- Footer: meta + offer button -->
                <div class="flex items-center justify-between gap-3 pt-2.5 border-t">
                  <div class="flex items-center gap-3 flex-wrap text-xs text-muted-foreground min-w-0">
                    <span v-if="item.customer_org_name" class="flex items-center gap-1 truncate max-w-40">
                      <Building2 class="h-3.5 w-3.5 shrink-0" />
                      {{ item.customer_org_name }}
                    </span>
                    <span class="flex items-center gap-1">
                      <CalendarDays class="h-3.5 w-3.5 shrink-0" />
                      {{ formatDateShort(item.invited_at) }}
                    </span>
                  </div>
                  <Button
                    v-if="item.freight_status === 'published' && canCreateOffer(item.customer_org_id ?? '')"
                    size="sm"
                    variant="outline"
                    class="shrink-0 text-xs h-7"
                    @click.stop="router.push({ name: 'freight-request-make-offer', params: { id: item.freight_request_id } })"
                  >
                    Сделать оффер
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

            <!-- Infinite scroll sentinel для приглашений -->
            <div ref="invSentinelRef" class="h-16 flex items-center justify-center">
              <template v-if="invLoadingMore">
                <LoadingSpinner text="Загрузка..." />
              </template>
              <template v-else-if="!invHasMore && invItems.length > 0">
                <span class="text-sm text-muted-foreground">
                  Все приглашения загружены ({{ invItems.length }})
                </span>
              </template>
            </div>
          </template>
        </template>

        <!-- ── Режим заявок ───────────────────────────────────────────────── -->
        <template v-else>
          <!-- Sort bar: скрываем во время загрузки и при ошибке -->
          <div v-if="!isLoading && !error" class="mb-4 pb-4 border-b">
            <!-- Desktop: иконки + тултипы -->
            <div v-if="!isBelowLg" class="flex gap-1">
              <Tooltip v-for="opt in SORT_OPTIONS" :key="opt.value" :text="opt.label" side="top">
                <button
                  type="button"
                  class="flex justify-center rounded-md border p-1.5 transition-colors"
                  :class="sortBy === opt.value ? 'bg-primary border-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-muted/50'"
                  @click="sortBy = opt.value as SortBy"
                >
                  <component :is="opt.icon" class="h-3.5 w-3.5" />
                </button>
              </Tooltip>
            </div>
            <!-- Mobile/tablet: селект -->
            <SelectField
              v-else
              v-model="sortBy"
              :options="SORT_SELECT_OPTIONS"
              :icon="ArrowDownNarrowWide"
              sheet-label="Сортировка"
            />
          </div>

          <!-- SSE: баннер новых данных -->
          <div
            v-if="hasUpdates && !isLoading"
            class="mb-4 flex items-center justify-between rounded-lg border border-primary/30 bg-primary/5 px-4 py-2.5 text-sm"
          >
            <span class="text-foreground">Список обновился — есть новые данные</span>
            <Button size="sm" variant="outline" @click="applyUpdates">Обновить</Button>
          </div>

          <LoadingSpinner v-if="isLoading" text="Загрузка заявок..." />

          <ErrorBanner
            v-else-if="error"
            :message="error"
            @retry="loadItems"
          />

          <EmptyState
            v-else-if="displayItems.length === 0"
            :icon="Package"
            title="Заявок пока нет"
            :description="hasActiveFilters ? 'Нет заявок по заданным фильтрам' : 'Создайте первую заявку на перевозку'"
            :action-label="canCreateFreightRequest && !hasActiveFilters ? 'Создать заявку' : undefined"
            @action="emit('go-to-new')"
          />

          <div v-else class="space-y-3">
            <Card
              v-for="item in displayItems"
              :key="item.id"
              data-tutorial="freight-request-card"
              interactive
              @click="goToDetail(item.id)"
            >
              <CardContent class="p-4">

                <!-- Header: number + status | price -->
                <div class="flex items-start justify-between gap-3 mb-3">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="text-sm font-medium text-muted-foreground">#{{ item.request_number }}</span>
                    <StatusBadge :status="item.status" :status-map="freightRequestStatusMap" />
                    <span
                      v-if="item.status === 'published' && isExpiringSoon(item.expires_at)"
                      class="inline-flex items-center gap-1 text-xs text-destructive"
                    >
                      <Clock class="h-3 w-3" />
                      Истекает {{ formatDateShort(item.expires_at) }}
                    </span>
                  </div>
                  <div v-if="item.price_amount" class="text-base font-bold text-emerald-600 dark:text-emerald-400 shrink-0">
                    {{ formatPrice(item.price_amount, item.price_currency) }}
                  </div>
                </div>

                <!-- Route -->
                <div class="flex items-stretch gap-3 mb-3">
                  <div class="flex flex-col items-center py-2">
                    <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
                    <div class="w-px flex-1 min-h-[18px] border-l border-dashed border-muted-foreground/40" />
                    <div class="w-2 h-2 rounded-full bg-muted-foreground/50 shrink-0" />
                  </div>
                  <div class="flex flex-col justify-between flex-1 min-w-0">
                    <div class="flex items-baseline gap-1.5 min-w-0">
                      <span class="text-base font-semibold text-foreground truncate">{{ item.origin_address || '—' }}</span>
                      <span
                        v-if="getTransitPointsCount(item) > 0"
                        class="text-xs text-primary font-medium shrink-0"
                      >+{{ getTransitPointsCount(item) }}</span>
                    </div>
                    <div class="text-base text-muted-foreground truncate">
                      {{ item.destination_address || '—' }}
                    </div>
                  </div>
                </div>

                <!-- Footer: meta + offer button -->
                <div class="flex items-center justify-between gap-3 pt-2.5 border-t">
                  <div class="flex items-center gap-3 flex-wrap text-xs text-muted-foreground min-w-0">
                    <span v-if="item.cargo_weight" class="flex items-center gap-1">
                      <Weight class="h-3.5 w-3.5 shrink-0" />
                      {{ formatWeightDisplay(item.cargo_weight) }}
                    </span>
                    <span v-if="item.vehicle_type" class="flex items-center gap-1 shrink-0">
                      <Truck class="h-3.5 w-3.5 shrink-0" />
                      {{ formatVehicleType(item.vehicle_type, item.vehicle_subtype) }}
                    </span>
                    <span v-if="item.customer_org_name" class="flex items-center gap-1 truncate max-w-40">
                      <Building2 class="h-3.5 w-3.5 shrink-0" />
                      {{ item.customer_org_name }}
                    </span>
                    <span class="flex items-center gap-1">
                      <CalendarDays class="h-3.5 w-3.5 shrink-0" />
                      {{ formatDateShort(item.created_at) }}
                    </span>
                    <span v-if="item.status === 'published' && item.expires_at && !isExpiringSoon(item.expires_at)" class="flex items-center gap-1">
                      <Timer class="h-3.5 w-3.5 shrink-0" />
                      до {{ formatDateShort(item.expires_at) }}
                    </span>
                  </div>
                  <Button
                    v-if="item.status === 'published' && canCreateOffer(item.customer_org_id)"
                    size="sm"
                    variant="outline"
                    class="shrink-0 text-xs h-7"
                    @click.stop="router.push({ name: 'freight-request-make-offer', params: { id: item.id } })"
                  >
                    Сделать оффер
                  </Button>
                </div>

              </CardContent>
            </Card>

            <!-- Infinite scroll sentinel -->
            <div ref="sentinelRef" class="h-16 flex items-center justify-center">
              <template v-if="isLoadingMore">
                <LoadingSpinner text="Загрузка..." />
              </template>
              <template v-else-if="!hasMore && items.length > 0">
                <span class="text-sm text-muted-foreground">
                  Все заявки загружены ({{ items.length }})
                </span>
              </template>
            </div>
          </div>
        </template>

      </div>
    </div>
  </div>
</template>
