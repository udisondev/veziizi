<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { useFreightFiltersStore, type RoutePointFilter } from '@/stores/freightFilters'
import { usePermissions } from '@/composables/usePermissions'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { useToast } from '@/components/ui/toast/use-toast'
import { freightRequestsApi, type FreightRequestListParams } from '@/api/freightRequests'
import type {
  FreightRequestListItem,
  OwnershipFilter,
  VehicleSubType,
  PaymentMethod,
  PaymentTerms,
  VatType,
  FreightRequestStatus,
} from '@/types/freightRequest'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  currencyLabels,
  countryLabels,
  type Country,
} from '@/types/freightRequest'
import { freightRequestStatusMap } from '@/constants/statusMaps'
import { formatDateShort, formatWeight } from '@/utils/formatters'
import { logger } from '@/utils/logger'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

// Shared Components
import {
  PageHeader,
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
  FilterSheet,
} from '@/components/shared'

// Filter Components
import { FreightFiltersForm } from '@/components/filters'

// Icons
import { Plus, Clock, Building2, Package, Bell } from 'lucide-vue-next'

const PAGE_SIZE = 20

const router = useRouter()
const auth = useAuthStore()
const onboarding = useOnboardingStore()
const filtersStore = useFreightFiltersStore()
const { toast } = useToast()
const { canCreateFreightRequest } = usePermissions()

const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const isLoadingMore = ref(false)
const error = ref<string | null>(null)
const showFilters = ref(false)

// Get reactive refs from store
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
  hasActiveFilters,
  activeFiltersCount,
  cursor,
  hasMore,
} = storeToRefs(filtersStore)

// Temp filters for sheet
const tempOwnership = ref<OwnershipFilter>('all')
const tempOrgINN = ref('')
const tempRequestNumber = ref<number | null>(null)
const tempStatuses = ref<FreightRequestStatus[]>([])
const tempRoutePoints = ref<RoutePointFilter[]>([])
const tempMinWeight = ref<number | undefined>()
const tempMaxWeight = ref<number | undefined>()
const tempMinPrice = ref<number | undefined>()
const tempMaxPrice = ref<number | undefined>()
const tempMinVolume = ref<number | undefined>()
const tempMaxVolume = ref<number | undefined>()
const tempVehicleSubTypes = ref<VehicleSubType[]>([])
const tempPaymentMethods = ref<PaymentMethod[]>([])
const tempPaymentTerms = ref<PaymentTerms[]>([])
const tempVatTypes = ref<VatType[]>([])


// Sheet functions
function openFilters() {
  tempOwnership.value = ownershipFilter.value
  tempOrgINN.value = orgINNFilter.value
  tempRequestNumber.value = requestNumber.value
  tempStatuses.value = [...statuses.value]
  tempRoutePoints.value = routePoints.value.map(rp => ({ ...rp }))
  tempMinWeight.value = minWeight.value
  tempMaxWeight.value = maxWeight.value
  tempMinPrice.value = minPrice.value
  tempMaxPrice.value = maxPrice.value
  tempMinVolume.value = minVolume.value
  tempMaxVolume.value = maxVolume.value
  tempVehicleSubTypes.value = [...vehicleSubTypes.value]
  tempPaymentMethods.value = [...paymentMethods.value]
  tempPaymentTerms.value = [...paymentTerms.value]
  tempVatTypes.value = [...vatTypes.value]
  showFilters.value = true
}

function applyFilters() {
  filtersStore.setFilters({
    ownership: tempOwnership.value,
    orgINN: tempOrgINN.value,
    requestNumber: tempRequestNumber.value,
    statuses: [...tempStatuses.value],
    routePoints: tempRoutePoints.value.map(rp => ({ ...rp })),
    minWeight: tempMinWeight.value,
    maxWeight: tempMaxWeight.value,
    minPrice: tempMinPrice.value,
    maxPrice: tempMaxPrice.value,
    minVolume: tempMinVolume.value,
    maxVolume: tempMaxVolume.value,
    vehicleSubTypes: [...tempVehicleSubTypes.value],
    paymentMethods: [...tempPaymentMethods.value],
    paymentTerms: [...tempPaymentTerms.value],
    vatTypes: [...tempVatTypes.value],
  })
  showFilters.value = false
}

function resetTempFilters() {
  tempOwnership.value = 'all'
  tempOrgINN.value = ''
  tempRequestNumber.value = null
  tempStatuses.value = []
  tempRoutePoints.value = []
  tempMinWeight.value = undefined
  tempMaxWeight.value = undefined
  tempMinPrice.value = undefined
  tempMaxPrice.value = undefined
  tempMinVolume.value = undefined
  tempMaxVolume.value = undefined
  tempVehicleSubTypes.value = []
  tempPaymentMethods.value = []
  tempPaymentTerms.value = []
  tempVatTypes.value = []
}

// Build params for API request
function buildParams(): FreightRequestListParams {
  const params: FreightRequestListParams = {
    limit: PAGE_SIZE,
  }

  // Status filter
  if (statuses.value.length > 0) {
    params.statuses = statuses.value.join(',')
  }

  // Ownership filter
  if (ownershipFilter.value === 'my_org' && auth.organizationId) {
    params.customer_org_id = auth.organizationId
  } else if (ownershipFilter.value === 'my' && auth.memberId) {
    params.member_id = auth.memberId
  }

  if (orgINNFilter.value) params.org_inn = orgINNFilter.value
  if (requestNumber.value) params.request_number = requestNumber.value

  // Numeric filters
  if (minWeight.value !== undefined) params.min_weight = minWeight.value
  if (maxWeight.value !== undefined) params.max_weight = maxWeight.value
  if (minPrice.value !== undefined) params.min_price = minPrice.value
  if (maxPrice.value !== undefined) params.max_price = maxPrice.value
  if (minVolume.value !== undefined) params.min_volume = minVolume.value
  if (maxVolume.value !== undefined) params.max_volume = maxVolume.value

  // Vehicle filter
  if (vehicleSubTypes.value.length > 0) params.vehicle_subtypes = vehicleSubTypes.value.join(',')

  // Payment filters
  if (paymentMethods.value.length > 0) params.payment_methods = paymentMethods.value.join(',')
  if (paymentTerms.value.length > 0) params.payment_terms = paymentTerms.value.join(',')
  if (vatTypes.value.length > 0) params.vat_types = vatTypes.value.join(',')

  // Route filter - extract city IDs and country IDs from route points
  if (routePoints.value.length > 0) {
    // Points with city selected -> filter by city
    const cityIds = routePoints.value
      .filter(rp => rp.cityId !== undefined)
      .map(rp => rp.cityId)
    if (cityIds.length > 0) {
      params.route_city_ids = cityIds.join(',')
    }

    // Points with only country (no city) -> filter by country
    const countryIds = routePoints.value
      .filter(rp => rp.countryId !== undefined && rp.cityId === undefined)
      .map(rp => rp.countryId)
    if (countryIds.length > 0) {
      params.route_country_ids = countryIds.join(',')
    }
  }

  return params
}

// Initial load (with reset)
async function loadItems() {
  isLoading.value = true
  error.value = null
  filtersStore.resetPagination()
  items.value = []

  try {
    const params = buildParams()
    const response = await freightRequestsApi.list(params)

    items.value = response.items
    filtersStore.setCursor(response.next_cursor)
    filtersStore.setHasMore(response.has_more)
  } catch (e) {
    error.value = 'Не удалось загрузить заявки'
    logger.error('Failed to load freight requests', e)
  } finally {
    isLoading.value = false
  }
}

// Load more items (infinite scroll)
async function loadMoreItems() {
  if (!hasMore.value || isLoadingMore.value || !cursor.value) return

  isLoadingMore.value = true
  try {
    const params = buildParams()
    params.cursor = cursor.value

    const response = await freightRequestsApi.list(params)

    items.value.push(...response.items)
    filtersStore.setCursor(response.next_cursor)
    filtersStore.setHasMore(response.has_more)
  } catch (e) {
    logger.error('Failed to load more freight requests', e)
  } finally {
    isLoadingMore.value = false
  }
}

// Infinite scroll setup
const canLoadMore = computed(() => hasMore.value && !isLoadingMore.value && cursor.value !== undefined)
const { sentinelRef, reset: resetInfiniteScroll } = useInfiniteScroll(loadMoreItems, {
  threshold: 300,
  enabled: canLoadMore,
})

function goToDetail(id: string) {
  router.push(`/freight-requests/${id}`)
}

function goToCreate() {
  router.push('/freight-requests/new')
}

function handleBellClick() {
  router.push('/subscriptions')
}

// Route point management functions for temp state
function addTempRoutePoint() {
  const newId = `rp-${Date.now()}`
  const order = tempRoutePoints.value.length
  tempRoutePoints.value.push({
    id: newId,
    countryId: undefined,
    cityId: undefined,
    order,
  })
}

function removeTempRoutePoint(id: string) {
  tempRoutePoints.value = tempRoutePoints.value.filter(rp => rp.id !== id)
  tempRoutePoints.value.forEach((rp, idx) => {
    rp.order = idx
  })
}

function updateTempRoutePoint(id: string, updates: Partial<RoutePointFilter>) {
  const point = tempRoutePoints.value.find(rp => rp.id === id)
  if (point) {
    Object.assign(point, updates)
  }
}

function reorderTempRoutePoints(points: RoutePointFilter[]) {
  tempRoutePoints.value = points
}

function formatPrice(amount?: number, currency?: string): string {
  if (!amount || !currency) return '—'
  const value = amount / 100
  const symbol = currencyLabels[currency as keyof typeof currencyLabels] || currency
  return `${value.toLocaleString('ru-RU')} ${symbol}`
}

function formatWeightDisplay(weight?: number): string {
  if (!weight) return '—'
  return formatWeight(weight * 1000) // конвертация из тонн в кг для formatWeight
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

// Преобразование sandbox заявки в формат списка
function sandboxRequestToListItem(req: NonNullable<typeof onboarding.sandboxCreatedRequest>): FreightRequestListItem {
  return {
    id: req.id,
    status: req.status as 'published',
    origin_address: req.origin_address,
    destination_address: req.destination_address,
    cargo_weight: req.cargo_weight / 1000, // кг в тонны
    vehicle_type: req.vehicle_type,
    vehicle_subtype: req.vehicle_subtype,
    price_amount: req.price_amount,
    price_currency: req.price_currency,
    created_at: req.created_at,
    expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(), // +30 дней
    customer_org_name: 'Моя организация',
  }
}

// Computed для списка с учётом sandbox режима
const displayItems = computed<FreightRequestListItem[]>(() => {
  if (onboarding.isSandboxMode && onboarding.sandboxCreatedRequest) {
    return [sandboxRequestToListItem(onboarding.sandboxCreatedRequest)]
  }
  return items.value
})

// Watch filters and reload (reset pagination on filter change)
watch(
  [
    ownershipFilter, orgINNFilter, requestNumber, statuses, routePoints,
    minWeight, maxWeight, minPrice, maxPrice, minVolume, maxVolume,
    vehicleSubTypes, paymentMethods, paymentTerms, vatTypes,
  ],
  async () => {
    await loadItems()
    await nextTick()
    resetInfiniteScroll()
  },
  { deep: true }
)

onMounted(() => {
  loadItems()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <PageHeader title="Заявки на перевозку" class="mb-6">
      <template #actions>
        <!-- Subscription Bell -->
        <Button
          variant="outline"
          size="icon"
          @click="handleBellClick"
          title="Рассылка"
        >
          <Bell class="h-4 w-4" />
        </Button>

        <!-- Filters Sheet -->
        <FilterSheet
          v-model:open="showFilters"
          :active-filters-count="activeFiltersCount"
          description="Настройте параметры поиска заявок"
          data-tutorial="filters-btn"
          @open="openFilters"
          @apply="applyFilters"
          @reset="resetTempFilters"
        >
          <FreightFiltersForm
            :route-points="tempRoutePoints"
            :min-weight="tempMinWeight"
            :max-weight="tempMaxWeight"
            :min-price="tempMinPrice"
            :max-price="tempMaxPrice"
            :min-volume="tempMinVolume"
            :max-volume="tempMaxVolume"
            :vehicle-sub-types="tempVehicleSubTypes"
            :payment-methods="tempPaymentMethods"
            :payment-terms="tempPaymentTerms"
            :vat-types="tempVatTypes"
            show-ownership
            :ownership="tempOwnership"
            show-i-n-n
            :org-i-n-n="tempOrgINN"
            show-request-number
            :request-number="tempRequestNumber"
            show-statuses
            :statuses="tempStatuses"
            @add-route-point="addTempRoutePoint"
            @remove-route-point="removeTempRoutePoint"
            @update-route-point="updateTempRoutePoint"
            @reorder-route-points="reorderTempRoutePoints"
            @update:min-weight="tempMinWeight = $event"
            @update:max-weight="tempMaxWeight = $event"
            @update:min-price="tempMinPrice = $event"
            @update:max-price="tempMaxPrice = $event"
            @update:min-volume="tempMinVolume = $event"
            @update:max-volume="tempMaxVolume = $event"
            @update:vehicle-sub-types="tempVehicleSubTypes = $event"
            @update:payment-methods="tempPaymentMethods = $event"
            @update:payment-terms="tempPaymentTerms = $event"
            @update:vat-types="tempVatTypes = $event"
            @update:ownership="tempOwnership = $event"
            @update:org-i-n-n="tempOrgINN = $event"
            @update:request-number="tempRequestNumber = $event"
            @update:statuses="tempStatuses = $event"
          />
        </FilterSheet>

        <Button
          v-if="canCreateFreightRequest"
          data-tutorial="create-request-btn"
          @click="goToCreate"
        >
          <Plus class="mr-2 h-4 w-4" />
          Новая заявка
        </Button>
      </template>
    </PageHeader>

    <!-- Active filters indicator -->
    <Card v-if="hasActiveFilters" class="mb-6 border-primary/20 bg-primary/5">
      <CardContent class="flex items-center justify-between py-3">
        <div class="text-sm text-primary">
          Активные фильтры: {{ activeFiltersCount }}
        </div>
        <Button variant="ghost" size="sm" @click="filtersStore.resetFilters">
          Сбросить
        </Button>
      </CardContent>
    </Card>

    <!-- Loading -->
    <LoadingSpinner v-if="isLoading" text="Загрузка заявок..." />

    <!-- Error -->
    <ErrorBanner
      v-else-if="error"
      :message="error"
      @retry="loadItems"
    />

    <!-- Empty state -->
    <EmptyState
      v-else-if="displayItems.length === 0"
      :icon="Package"
      title="Заявок пока нет"
      :description="hasActiveFilters ? 'Нет заявок по заданным фильтрам' : 'Создайте первую заявку на перевозку'"
      :action-label="canCreateFreightRequest && !hasActiveFilters ? 'Создать заявку' : undefined"
      @action="goToCreate"
    />

    <!-- List -->
    <div v-else class="space-y-4">
      <Card
        v-for="item in displayItems"
        :key="item.id"
        data-tutorial="freight-request-card"
        class="hover:shadow-md transition-shadow cursor-pointer"
        @click="goToDetail(item.id)"
      >
        <CardContent class="p-4">
          <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <!-- Route -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <span class="text-sm font-medium text-muted-foreground">#{{ item.request_number }}</span>
                <StatusBadge :status="item.status" :status-map="freightRequestStatusMap" />
                <span
                  v-if="item.status === 'published' && isExpiringSoon(item.expires_at)"
                  class="inline-flex items-center gap-1 text-xs text-warning"
                >
                  <Clock class="h-3 w-3" />
                  Истекает скоро
                </span>
              </div>

              <!-- Route with vertical dashed line -->
              <div class="flex items-stretch gap-3">
                <!-- Vertical line with dots -->
                <div class="flex flex-col items-center py-1">
                  <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
                  <div class="w-px flex-1 border-l border-dashed border-muted-foreground/40 min-h-2" />
                  <div
                    v-if="getTransitPointsCount(item) > 0"
                    class="text-xs text-muted-foreground bg-background px-1 shrink-0"
                  >
                    +{{ getTransitPointsCount(item) }}
                  </div>
                  <div
                    v-if="getTransitPointsCount(item) > 0"
                    class="w-px flex-1 border-l border-dashed border-muted-foreground/40 min-h-2"
                  />
                  <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
                </div>
                <!-- Addresses -->
                <div class="flex flex-col justify-between flex-1 min-w-0 gap-1">
                  <div class="text-lg font-medium text-foreground truncate">
                    {{ item.origin_address || 'Не указан' }}
                  </div>
                  <div class="text-lg font-medium text-foreground truncate">
                    {{ item.destination_address || 'Не указан' }}
                  </div>
                </div>
              </div>

              <!-- Organization info -->
              <div v-if="item.customer_org_name" class="flex items-center gap-1 text-sm text-muted-foreground mt-2">
                <Building2 class="h-4 w-4" />
                {{ item.customer_org_name }}
                <span v-if="item.customer_org_country" class="text-muted-foreground/70">
                  ({{ countryLabels[item.customer_org_country as Country] || item.customer_org_country }})
                </span>
              </div>
            </div>

            <!-- Details -->
            <div class="flex flex-wrap gap-4 lg:gap-6 text-sm">
              <!-- Weight -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Вес</div>
                <div class="font-medium">{{ formatWeightDisplay(item.cargo_weight) }}</div>
              </div>

              <!-- Vehicle -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Транспорт</div>
                <div class="font-medium truncate max-w-32">
                  {{ formatVehicleType(item.vehicle_type, item.vehicle_subtype) }}
                </div>
              </div>

              <!-- Price -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Ставка</div>
                <div class="font-medium text-success">
                  {{ formatPrice(item.price_amount, item.price_currency) }}
                </div>
              </div>

              <!-- Date -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Создана</div>
                <div class="font-medium">{{ formatDateShort(item.created_at) }}</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Infinite scroll sentinel -->
      <div
        ref="sentinelRef"
        class="h-16 flex items-center justify-center"
      >
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
  </div>
</template>
