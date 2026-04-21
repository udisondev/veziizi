<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import { usePermissions } from '@/composables/usePermissions'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { freightRequestsApi, type FreightRequestListParams } from '@/api/freightRequests'
import type {
  FreightRequestListItem,
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
import { Tooltip } from '@/components/ui/tooltip'

// Shared Components
import {
  PageHeader,
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
} from '@/components/shared'

// Filter Components
import { FreightFiltersForm } from '@/components/filters'

// Icons
import { Plus, Clock, Building2, Package, Bell, SlidersHorizontal } from 'lucide-vue-next'

const PAGE_SIZE = 20

const router = useRouter()
const auth = useAuthStore()
const onboarding = useOnboardingStore()
const filtersStore = useFreightFiltersStore()
const { canCreateFreightRequest } = usePermissions()

const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const isLoadingMore = ref(false)
const error = ref<string | null>(null)
const mobileFiltersOpen = ref(false)

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
  cursor,
  hasMore,
} = storeToRefs(filtersStore)

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

// Load more items (infinite scroll)
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

// Infinite scroll setup
const canLoadMore = computed(() => hasMore.value && !isLoadingMore.value && cursor.value !== undefined)
const { sentinelRef, reset: resetInfiniteScroll } = useInfiniteScroll(loadMoreItems, {
  threshold: 300,
  enabled: canLoadMore,
})

// Template refs used in template via ref="..." (vue-tsc false positive workaround)
void sentinelRef

function goToDetail(id: string) {
  router.push(`/freight-requests/${id}`)
}

function goToCreate() {
  router.push('/freight-requests/new')
}

function handleBellClick() {
  router.push('/subscriptions')
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
    request_number: 1, // sandbox mock
    customer_org_id: 'sandbox-org-1',
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
// Debounce to prevent excessive API calls during rapid filter changes
let filterDebounceTimer: ReturnType<typeof setTimeout> | null = null
const FILTER_DEBOUNCE_MS = 300

watch(
  [
    ownershipFilter, orgINNFilter, requestNumber, statuses, routePoints,
    minWeight, maxWeight, minPrice, maxPrice, minVolume, maxVolume,
    vehicleSubTypes, paymentMethods, paymentTerms, vatTypes,
  ],
  () => {
    if (filterDebounceTimer) {
      clearTimeout(filterDebounceTimer)
    }
    filterDebounceTimer = setTimeout(async () => {
      await loadItems()
      await nextTick()
      resetInfiniteScroll()
    }, FILTER_DEBOUNCE_MS)
  },
  { deep: true }
)

onUnmounted(() => {
  if (filterDebounceTimer) {
    clearTimeout(filterDebounceTimer)
  }
})

onMounted(() => {
  loadItems()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <PageHeader title="Заявки на перевозку" class="mb-6">
      <template #actions>
        <Tooltip text="Рассылка уведомлений">
          <Button
            variant="outline"
            size="icon"
            @click="handleBellClick"
          >
            <Bell class="h-4 w-4" />
          </Button>
        </Tooltip>

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

    <!-- Mobile filter toggle -->
    <div class="lg:hidden mb-4">
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

    <!-- 2-column layout: filters left, list right -->
    <div class="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-6 items-start">

      <!-- Filters column -->
      <div :class="['lg:sticky lg:top-6 space-y-2', mobileFiltersOpen ? 'block' : 'hidden lg:block']">
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
          show-ownership
          :ownership="ownershipFilter"
          show-i-n-n
          :org-i-n-n="orgINNFilter"
          show-request-number
          :request-number="requestNumber"
          show-statuses
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

      <!-- List column -->
      <div>
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
            interactive
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
      </div><!-- end list column -->

    </div><!-- end 2-column grid -->
  </div>
</template>
