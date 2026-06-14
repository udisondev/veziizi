<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { freightRequestsApi, type FreightRequestListParams } from '@/api/freightRequests'
import { vehiclesApi } from '@/api/vehicles'
import { buildRouteParams } from '@/stores/freightFilters'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { useToast } from '@/components/ui/toast/use-toast'
import { logger } from '@/utils/logger'
import { formatDateShort, formatWeight, pluralizeRu } from '@/utils/formatters'
import type { FreightRequestListItem, VehicleSubType, PaymentMethod, PaymentTerms, VatType, LoadingType } from '@/types/freightRequest'
import { currencyLabels, vehicleTypeLabels, vehicleSubTypeLabels, loadingTypeLabels } from '@/types/freightRequest'
import type { Vehicle } from '@/types/vehicle'
import type { RoutePointData } from '@/types/routePoint'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { LoadingSpinner, EmptyState, ErrorBanner, DetailPageHeader } from '@/components/shared'
import { FreightFiltersForm } from '@/components/filters'
import { StatusBadge } from '@/components/shared'
import { Truck, SlidersHorizontal, Package, Weight, CalendarDays, Gauge, Box } from 'lucide-vue-next'

const PAGE_SIZE = 20

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { toast } = useToast()

const vehicleId = computed(() => route.params.vehicleId as string)

// Vehicle
const vehicle = ref<Vehicle | null>(null)
const vehicleLoading = ref(false)

async function loadVehicle() {
  vehicleLoading.value = true
  try {
    vehicle.value = await vehiclesApi.get(vehicleId.value)
  } catch (e) {
    logger.error('Failed to load vehicle', e)
  } finally {
    vehicleLoading.value = false
  }
}

// Filter state
const routePoints = ref<RoutePointData[]>([])
const minWeight = ref<number | undefined>()
const maxWeight = ref<number | undefined>()
const minPrice = ref<number | undefined>()
const maxPrice = ref<number | undefined>()
const minVolume = ref<number | undefined>()
const maxVolume = ref<number | undefined>()
const vehicleSubTypes = ref<VehicleSubType[]>([])
const paymentMethods = ref<PaymentMethod[]>([])
const paymentTerms = ref<PaymentTerms[]>([])
const vatTypes = ref<VatType[]>([])
const requestNumber = ref<number | null>(null)

const mobileFiltersOpen = ref(false)

const hasActiveFilters = computed(() =>
  routePoints.value.length > 0 ||
  minWeight.value !== undefined ||
  maxWeight.value !== undefined ||
  minPrice.value !== undefined ||
  maxPrice.value !== undefined ||
  minVolume.value !== undefined ||
  maxVolume.value !== undefined ||
  vehicleSubTypes.value.length > 0 ||
  paymentMethods.value.length > 0 ||
  paymentTerms.value.length > 0 ||
  vatTypes.value.length > 0 ||
  requestNumber.value !== null
)

function resetFilters() {
  routePoints.value = []
  minWeight.value = undefined
  maxWeight.value = undefined
  minPrice.value = undefined
  maxPrice.value = undefined
  minVolume.value = undefined
  maxVolume.value = undefined
  vehicleSubTypes.value = []
  paymentMethods.value = []
  paymentTerms.value = []
  vatTypes.value = []
  requestNumber.value = null
}

// Freight requests list
const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const isLoadingMore = ref(false)
const error = ref<string | null>(null)
const cursor = ref<string | undefined>()
const hasMore = ref(false)

const positive = (v: number | undefined): v is number =>
  typeof v === 'number' && Number.isFinite(v) && v > 0

function buildParams(): FreightRequestListParams {
  const params: FreightRequestListParams = {
    limit: PAGE_SIZE,
    statuses: 'published',
    not_expired: true,
  }

  const role = auth.role
  if (role === 'owner' || role === 'administrator') {
    if (auth.organizationId) params.customer_org_id = auth.organizationId
  } else {
    if (auth.memberId) params.member_id = auth.memberId
  }

  if (requestNumber.value) params.request_number = requestNumber.value
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

  Object.assign(params, buildRouteParams(routePoints.value))

  return params
}

async function loadItems() {
  isLoading.value = true
  error.value = null
  cursor.value = undefined
  items.value = []
  selected.value.clear()
  try {
    const res = await freightRequestsApi.list(buildParams())
    items.value = res.items ?? []
    cursor.value = res.next_cursor
    hasMore.value = res.has_more
  } catch (e) {
    error.value = 'Не удалось загрузить заявки'
    logger.error('Failed to load freight requests for selection', e)
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
    const res = await freightRequestsApi.list(params)
    items.value.push(...(res.items ?? []))
    cursor.value = res.next_cursor
    hasMore.value = res.has_more
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

// Selection
const selected = ref(new Set<string>())

const selectedCount = computed(() => selected.value.size)

function toggleSelect(id: string) {
  if (selected.value.has(id)) {
    selected.value.delete(id)
  } else {
    selected.value.add(id)
  }
  // Trigger reactivity — Set mutations aren't tracked automatically
  selected.value = new Set(selected.value)
}

function isSelected(id: string) {
  return selected.value.has(id)
}

// Submit invitations
const isSubmitting = ref(false)

async function submitInvitations() {
  if (selectedCount.value === 0 || isSubmitting.value) return
  isSubmitting.value = true

  const ids = Array.from(selected.value)
  let successCount = 0
  let alreadyCount = 0
  let errorCount = 0

  await Promise.all(
    ids.map(async (frId) => {
      try {
        await freightRequestsApi.inviteCarrier(frId, vehicleId.value)
        successCount++
      } catch (e: any) {
        if (e?.status === 409) {
          alreadyCount++
        } else {
          errorCount++
          logger.error('Failed to invite carrier for request', frId, e)
        }
      }
    })
  )

  isSubmitting.value = false

  if (errorCount > 0) {
    toast({
      title: 'Частичная ошибка',
      description: `Отправлено: ${successCount}, уже приглашены: ${alreadyCount}, ошибок: ${errorCount}`,
      variant: 'destructive',
    })
  } else if (successCount > 0) {
    const parts: string[] = [`Направлено ${successCount} ${pluralizeRu(successCount, 'заявка', 'заявки', 'заявок')}`]
    if (alreadyCount > 0) parts.push(`${alreadyCount} ${pluralizeRu(alreadyCount, 'заявка уже была направлена ранее', 'заявки уже были направлены ранее', 'заявок уже были направлены ранее')}`)
    toast({ title: 'Приглашения отправлены', description: parts.join(', '), variant: 'success', duration: 10000 })
    router.push('/transport')
  } else if (alreadyCount > 0) {
    toast({
      title: 'Уже приглашены',
      description: 'Выбранные заявки уже были направлены этому перевозчику',
      variant: 'destructive',
    })
  }
}

// Formatters
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

function formatVehicleType(type?: string, subtype?: string): string {
  if (!type) return '—'
  const typeLabel = vehicleTypeLabels[type as keyof typeof vehicleTypeLabels] || type
  if (subtype) {
    const subtypeLabel = vehicleSubTypeLabels[subtype as keyof typeof vehicleSubTypeLabels] || subtype
    return `${typeLabel} · ${subtypeLabel}`
  }
  return typeLabel
}

function openRequestInNewTab(id: string) {
  window.open(`/freight-requests/${id}`, '_blank')
}

// Route point helpers
function addRoutePoint() {
  routePoints.value.push({ id: window.crypto.randomUUID(), order: routePoints.value.length, role: 'any' })
}
function removeRoutePoint(id: string) {
  routePoints.value = routePoints.value.filter(p => p.id !== id)
}
function updateRoutePoint(id: string, upd: Partial<RoutePointData>) {
  const p = routePoints.value.find(p => p.id === id)
  if (p) Object.assign(p, upd)
}
function reorderRoutePoints(pts: RoutePointData[]) {
  routePoints.value = pts
}

// Debounce filter reload
let debounceTimer: ReturnType<typeof setTimeout> | null = null

watch(
  [routePoints, minWeight, maxWeight, minPrice, maxPrice, minVolume, maxVolume,
   vehicleSubTypes, paymentMethods, paymentTerms, vatTypes, requestNumber],
  () => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      await loadItems()
      await nextTick()
      resetInfiniteScroll()
    }, 300)
  },
  { deep: true }
)

onUnmounted(() => { if (debounceTimer) clearTimeout(debounceTimer) })

onMounted(() => {
  loadVehicle()
  loadItems()
})
</script>

<template>
  <div class="min-h-screen bg-background">

    <!-- Sticky back navigation -->
    <DetailPageHeader :back-to="{ path: '/transport' }" back-label="К транспорту" use-history wide />

    <div class="max-w-7xl mx-auto pt-4 pb-28 px-4 sm:px-6 lg:px-8">

      <!-- Page title block -->
      <div class="mb-4">
        <h1 class="text-xl font-semibold">Выбор заявок для приглашения</h1>

        <!-- Vehicle info block -->
        <div v-if="vehicle" class="mt-3 p-3 rounded-lg border bg-card flex items-start gap-3">
          <Truck class="h-5 w-5 mt-0.5 shrink-0 text-muted-foreground" />
          <div class="min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="font-semibold text-sm">
                {{ [vehicle.brand, vehicle.model].filter(Boolean).join(' ') || 'Транспортное средство' }}
              </span>
              <span class="font-mono text-sm text-muted-foreground">{{ vehicle.registration_number }}</span>
            </div>
            <p class="text-xs text-muted-foreground mt-0.5">
              {{ vehicleTypeLabels[vehicle.vehicle_type] }} · {{ vehicleSubTypeLabels[vehicle.vehicle_subtype] }}
            </p>
            <div class="flex items-center gap-2 mt-1.5 flex-wrap">
              <span v-if="vehicle.capacity" class="flex items-center gap-1 text-xs text-muted-foreground">
                <Gauge class="h-3.5 w-3.5 shrink-0" />{{ vehicle.capacity }} т
              </span>
              <span v-if="vehicle.volume" class="flex items-center gap-1 text-xs text-muted-foreground">
                <Box class="h-3.5 w-3.5 shrink-0" />{{ vehicle.volume }} м³
              </span>
              <Badge
                v-for="lt in (vehicle.loading_types ?? [])"
                :key="lt"
                variant="secondary"
                class="text-xs h-5 font-normal"
              >{{ loadingTypeLabels[lt as LoadingType] }}</Badge>
              <Badge v-if="vehicle.requires_adr" variant="destructive" class="text-xs h-5">ADR</Badge>
              <Badge v-if="vehicle.thermograph" variant="outline" class="text-xs h-5">Термописец</Badge>
            </div>
          </div>
        </div>

        <p v-else-if="vehicleLoading" class="text-sm text-muted-foreground mt-2">Загрузка...</p>
      </div>

    <!-- Mobile filter toggle -->
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

    <!-- 2-column layout -->
    <div class="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-6 items-start">

      <!-- Filters sidebar -->
      <div class="lg:sticky lg:top-16">
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
                show-request-number
                :request-number="requestNumber"
                @add-route-point="addRoutePoint"
                @remove-route-point="removeRoutePoint"
                @update-route-point="updateRoutePoint"
                @reorder-route-points="reorderRoutePoints"
                @update:min-weight="minWeight = $event"
                @update:max-weight="maxWeight = $event"
                @update:min-price="minPrice = $event"
                @update:max-price="maxPrice = $event"
                @update:min-volume="minVolume = $event"
                @update:max-volume="maxVolume = $event"
                @update:vehicle-sub-types="vehicleSubTypes = $event"
                @update:payment-methods="paymentMethods = $event"
                @update:payment-terms="paymentTerms = $event"
                @update:vat-types="vatTypes = $event"
                @update:request-number="requestNumber = $event"
              />
              <div v-if="hasActiveFilters" class="flex justify-center pt-1">
                <Button variant="ghost" size="sm" @click="resetFilters">Сбросить фильтры</Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- List -->
      <div>
        <LoadingSpinner v-if="isLoading" text="Загрузка заявок..." />

        <ErrorBanner v-else-if="error" :message="error" @retry="loadItems" />

        <EmptyState
          v-else-if="items.length === 0"
          :icon="Truck"
          title="Нет активных заявок"
          :description="hasActiveFilters ? 'Нет заявок по заданным фильтрам' : 'У вас нет активных опубликованных заявок'"
        />

        <div v-else class="space-y-2">
          <Card
            v-for="item in items"
            :key="item.id"
            class="cursor-pointer transition-colors"
            :class="isSelected(item.id) ? 'bg-muted/50' : ''"
            @click="toggleSelect(item.id)"
          >
            <CardContent class="p-4">
              <div class="flex items-start gap-2">
                <div class="flex-1 min-w-0">
                  <!-- Number + status -->
                  <div class="flex items-center gap-2 mb-1.5 flex-wrap">
                    <span class="font-semibold text-sm">№{{ item.request_number }}</span>
                    <StatusBadge :status="item.status" />
                  </div>

                  <!-- Route -->
                  <div v-if="item.origin_address || item.destination_address" class="text-sm mb-1.5">
                    <span class="text-muted-foreground">
                      {{ item.origin_address || '—' }}
                      <span class="mx-1">→</span>
                      {{ item.destination_address || '—' }}
                    </span>
                  </div>

                  <!-- Meta row -->
                  <div class="flex items-center gap-3 flex-wrap text-xs text-muted-foreground">
                    <span v-if="item.cargo_weight" class="flex items-center gap-1">
                      <Weight class="h-3.5 w-3.5 shrink-0" />
                      {{ formatWeightDisplay(item.cargo_weight) }}
                    </span>
                    <span v-if="item.price_amount" class="flex items-center gap-1">
                      <Package class="h-3.5 w-3.5 shrink-0" />
                      {{ formatPrice(item.price_amount, item.price_currency) }}
                    </span>
                    <span v-if="item.vehicle_type" class="flex items-center gap-1">
                      <Truck class="h-3.5 w-3.5 shrink-0" />
                      {{ formatVehicleType(item.vehicle_type, item.vehicle_subtype) }}
                    </span>
                    <span class="flex items-center gap-1">
                      <CalendarDays class="h-3.5 w-3.5 shrink-0" />
                      до {{ formatDateShort(item.expires_at) }}
                    </span>
                  </div>
                </div>

                <!-- Open in new tab -->
                <button
                  class="shrink-0 text-xs text-primary hover:underline whitespace-nowrap"
                  @click.stop="openRequestInNewTab(item.id)"
                >
                  Открыть заявку
                </button>
              </div>
            </CardContent>
          </Card>

          <!-- Infinite scroll sentinel -->
          <div ref="sentinelRef" class="h-12 flex items-center justify-center">
            <template v-if="isLoadingMore">
              <LoadingSpinner text="Загрузка..." />
            </template>
            <template v-else-if="!hasMore && items.length > 0">
              <span class="text-xs text-muted-foreground">Все заявки загружены ({{ items.length }})</span>
            </template>
          </div>
        </div>
      </div>

    </div>

    </div>

    <!-- Fixed bottom panel -->
    <Transition name="slide-up">
      <div
        v-if="selectedCount > 0"
        class="fixed bottom-0 left-0 right-0 z-50 bg-background border-t shadow-lg px-4 py-3"
      >
        <div class="max-w-7xl mx-auto flex items-center justify-between gap-4">
          <span class="text-sm font-medium">
            Выбрано {{ selectedCount }} {{ pluralizeRu(selectedCount, 'заявка', 'заявки', 'заявок') }}
          </span>
          <div class="flex gap-2">
            <Button variant="outline" size="sm" @click="selected = new Set()">
              Отменить
            </Button>
            <Button size="sm" :disabled="isSubmitting" @click="submitInvitations">
              <span v-if="isSubmitting">Отправляем...</span>
              <span v-else>Направить заявки</span>
            </Button>
          </div>
        </div>
      </div>
    </Transition>

  </div>
</template>

<style scoped>
.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.2s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(100%);
}
</style>
