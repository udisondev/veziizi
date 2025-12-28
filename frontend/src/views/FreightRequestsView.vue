<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useFreightFiltersStore, type RoutePointFilter } from '@/stores/freightFilters'
import { usePermissions } from '@/composables/usePermissions'
import { useToast } from '@/components/ui/toast/use-toast'
import { freightRequestsApi } from '@/api/freightRequests'
import type {
  FreightRequestListItem,
  FreightRequestStatus,
  FreightRequestStatusFilter,
  OwnershipFilter,
  VehicleType,
  VehicleSubType,
} from '@/types/freightRequest'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  vehicleTypeOptions,
  currencyLabels,
  countryLabels,
  ownershipOptions,
  statusOptions,
  type Country,
} from '@/types/freightRequest'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// Shared Components
import {
  PageHeader,
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
  FilterSheet,
} from '@/components/shared'
import { Separator } from '@/components/ui/separator'

// Filter Components
import { ChipButtonGroup, RangeInput } from '@/components/filters'
import QuickSubscribeDialog from '@/components/subscriptions/QuickSubscribeDialog.vue'
import SubscriptionRouteStep from '@/components/subscriptions/SubscriptionRouteStep.vue'

// Icons
import { Plus, Clock, Building2, Package, Bell } from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()
const filtersStore = useFreightFiltersStore()
const { toast } = useToast()
const { canCreateFreightRequest } = usePermissions()

const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const showFilters = ref(false)
const showSubscribeDialog = ref(false)

// Get reactive refs from store
const {
  ownershipFilter,
  statusFilter,
  orgNameFilter,
  orgINNFilter,
  routePoints,
  minWeight,
  maxWeight,
  minPrice,
  maxPrice,
  vehicleTypes,
  vehicleSubTypes,
  hasSubscriptionFilters,
  hasActiveFilters,
  activeFiltersCount,
} = storeToRefs(filtersStore)

// Temp filters for sheet
const tempOwnership = ref<OwnershipFilter>('all')
const tempStatus = ref<FreightRequestStatusFilter>('all')
const tempOrgName = ref('')
const tempOrgINN = ref('')
const tempRoutePoints = ref<RoutePointFilter[]>([])
const tempMinWeight = ref<number | undefined>()
const tempMaxWeight = ref<number | undefined>()
const tempMinPrice = ref<number | undefined>()
const tempMaxPrice = ref<number | undefined>()
const tempVehicleTypes = ref<VehicleType[]>([])
const tempVehicleSubTypes = ref<VehicleSubType[]>([])

// Status map for StatusBadge
const freightStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  published: { label: 'Опубликована', variant: 'success' },
  selected: { label: 'Выбран исполнитель', variant: 'warning' },
  confirmed: { label: 'Подтверждена', variant: 'info' },
  cancelled: { label: 'Отменена', variant: 'destructive' },
  expired: { label: 'Истекла', variant: 'secondary' },
}

// Sheet functions
function openFilters() {
  tempOwnership.value = ownershipFilter.value
  tempStatus.value = statusFilter.value
  tempOrgName.value = orgNameFilter.value
  tempOrgINN.value = orgINNFilter.value
  tempRoutePoints.value = routePoints.value.map(rp => ({ ...rp }))
  tempMinWeight.value = minWeight.value
  tempMaxWeight.value = maxWeight.value
  tempMinPrice.value = minPrice.value
  tempMaxPrice.value = maxPrice.value
  tempVehicleTypes.value = [...vehicleTypes.value]
  tempVehicleSubTypes.value = [...vehicleSubTypes.value]
  showFilters.value = true
}

function applyFilters() {
  filtersStore.setFilters({
    ownership: tempOwnership.value,
    status: tempStatus.value,
    orgName: tempOrgName.value,
    orgINN: tempOrgINN.value,
    routePoints: tempRoutePoints.value.map(rp => ({ ...rp })),
    minWeight: tempMinWeight.value,
    maxWeight: tempMaxWeight.value,
    minPrice: tempMinPrice.value,
    maxPrice: tempMaxPrice.value,
    vehicleTypes: [...tempVehicleTypes.value],
    vehicleSubTypes: [...tempVehicleSubTypes.value],
  })
  showFilters.value = false
}

function resetTempFilters() {
  tempOwnership.value = 'all'
  tempStatus.value = 'all'
  tempOrgName.value = ''
  tempOrgINN.value = ''
  tempRoutePoints.value = []
  tempMinWeight.value = undefined
  tempMaxWeight.value = undefined
  tempMinPrice.value = undefined
  tempMaxPrice.value = undefined
  tempVehicleTypes.value = []
  tempVehicleSubTypes.value = []
}

// Load data with filters
async function loadItems() {
  isLoading.value = true
  error.value = null

  try {
    const params: Parameters<typeof freightRequestsApi.list>[0] = {}

    // Basic filters
    if (ownershipFilter.value === 'my_org' && auth.organizationId) {
      params.customer_org_id = auth.organizationId
    } else if (ownershipFilter.value === 'my' && auth.memberId) {
      params.member_id = auth.memberId
    }

    if (statusFilter.value !== 'all') params.status = statusFilter.value as FreightRequestStatus
    if (orgNameFilter.value) params.org_name = orgNameFilter.value
    if (orgINNFilter.value) params.org_inn = orgINNFilter.value

    // Extended filters
    if (minWeight.value !== undefined) params.min_weight = minWeight.value
    if (maxWeight.value !== undefined) params.max_weight = maxWeight.value
    if (minPrice.value !== undefined) params.min_price = minPrice.value
    if (maxPrice.value !== undefined) params.max_price = maxPrice.value
    if (vehicleTypes.value.length > 0) params.vehicle_types = vehicleTypes.value.join(',')
    if (vehicleSubTypes.value.length > 0) params.vehicle_subtypes = vehicleSubTypes.value.join(',')

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

function handleBellClick() {
  if (hasSubscriptionFilters.value) {
    showSubscribeDialog.value = true
  } else {
    toast({
      title: 'Настройте фильтры',
      description: 'Откройте панель фильтров и задайте параметры для подписки',
    })
  }
}

// Get current filters for subscription dialog
const currentSubscriptionFilters = computed(() => ({
  routePoints: routePoints.value,
  minWeight: minWeight.value,
  maxWeight: maxWeight.value,
  minPrice: minPrice.value,
  maxPrice: maxPrice.value,
  vehicleTypes: vehicleTypes.value,
  vehicleSubTypes: vehicleSubTypes.value,
}))

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

function formatWeight(weight?: number): string {
  if (!weight) return '—'
  return `${weight.toLocaleString('ru-RU')} т`
}

function getTransitPointsCount(item: FreightRequestListItem): number {
  if (!item.route?.points || item.route.points.length <= 2) return 0
  return item.route.points.length - 2
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
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

// Watch filters and reload
watch(
  [ownershipFilter, statusFilter, orgNameFilter, orgINNFilter, routePoints,
   minWeight, maxWeight, minPrice, maxPrice, vehicleTypes, vehicleSubTypes],
  () => loadItems(),
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
          :class="hasSubscriptionFilters ? 'text-primary border-primary' : ''"
          @click="handleBellClick"
          title="Подписаться на заявки по текущим фильтрам"
        >
          <Bell class="h-4 w-4" />
        </Button>

        <!-- Filters Sheet -->
        <FilterSheet
          v-model:open="showFilters"
          :active-filters-count="activeFiltersCount"
          description="Настройте параметры поиска заявок"
          @open="openFilters"
          @apply="applyFilters"
          @reset="resetTempFilters"
        >
          <!-- Ownership -->
          <div class="space-y-2">
            <Label>Принадлежность</Label>
            <Select v-model="tempOwnership">
              <SelectTrigger>
                <SelectValue placeholder="Выберите..." />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in ownershipOptions"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <!-- Status -->
          <div class="space-y-2">
            <Label>Статус</Label>
            <Select v-model="tempStatus">
              <SelectTrigger>
                <SelectValue placeholder="Все статусы" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in statusOptions"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <!-- Organization name -->
          <div class="space-y-2">
            <Label>Название организации</Label>
            <Input
              v-model="tempOrgName"
              placeholder="Поиск по названию"
            />
          </div>

          <!-- INN -->
          <div class="space-y-2">
            <Label>ИНН</Label>
            <Input
              v-model="tempOrgINN"
              placeholder="Поиск по ИНН"
            />
          </div>

          <!-- Route Points -->
          <SubscriptionRouteStep
            :route-points="tempRoutePoints"
            @add-point="addTempRoutePoint"
            @remove-point="removeTempRoutePoint"
            @update-point="updateTempRoutePoint"
            @reorder="reorderTempRoutePoints"
          />

          <Separator />

          <!-- Weight Range -->
          <RangeInput
            :min-value="tempMinWeight"
            :max-value="tempMaxWeight"
            label="Вес груза, т"
            :step="0.1"
            @update:min-value="tempMinWeight = $event"
            @update:max-value="tempMaxWeight = $event"
          />

          <!-- Price Range -->
          <RangeInput
            :min-value="tempMinPrice"
            :max-value="tempMaxPrice"
            label="Ставка, руб."
            :step="1000"
            @update:min-value="tempMinPrice = $event"
            @update:max-value="tempMaxPrice = $event"
          />

          <!-- Vehicle Types -->
          <ChipButtonGroup
            v-model="tempVehicleTypes"
            :options="vehicleTypeOptions"
            label="Тип транспорта"
            empty-text="Не выбрано — все типы транспорта"
          />
        </FilterSheet>

        <Button v-if="canCreateFreightRequest" @click="goToCreate">
          <Plus class="mr-2 h-4 w-4" />
          Новая заявка
        </Button>
      </template>
    </PageHeader>

    <!-- Active filters indicator -->
    <Card v-if="hasActiveFilters" class="mb-6 border-primary/20 bg-primary/5">
      <CardContent class="flex items-center justify-between py-3">
        <div class="text-sm text-primary flex flex-wrap gap-x-2 gap-y-1">
          <span v-if="ownershipFilter !== 'all'">
            {{ ownershipOptions.find(o => o.value === ownershipFilter)?.label }}
          </span>
          <span v-if="statusFilter !== 'all'">
            <span v-if="ownershipFilter !== 'all'">, </span>
            Статус: {{ statusOptions.find(o => o.value === statusFilter)?.label }}
          </span>
          <span v-if="orgNameFilter">
            <span v-if="ownershipFilter !== 'all' || statusFilter !== 'all'">, </span>
            Организация: "{{ orgNameFilter }}"
          </span>
          <span v-if="orgINNFilter">
            <span v-if="ownershipFilter !== 'all' || statusFilter !== 'all' || orgNameFilter">, </span>
            ИНН: "{{ orgINNFilter }}"
          </span>
          <span v-if="routePoints.length > 0">
            <span v-if="ownershipFilter !== 'all' || statusFilter !== 'all' || orgNameFilter || orgINNFilter">, </span>
            Точек маршрута: {{ routePoints.length }}
          </span>
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
      v-else-if="items.length === 0"
      :icon="Package"
      title="Заявок пока нет"
      :description="hasActiveFilters ? 'Нет заявок по заданным фильтрам' : 'Создайте первую заявку на перевозку'"
      :action-label="canCreateFreightRequest && !hasActiveFilters ? 'Создать заявку' : undefined"
      @action="goToCreate"
    />

    <!-- List -->
    <div v-else class="space-y-4">
      <Card
        v-for="item in items"
        :key="item.id"
        class="hover:shadow-md transition-shadow cursor-pointer"
        @click="goToDetail(item.id)"
      >
        <CardContent class="p-4">
          <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <!-- Route -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <StatusBadge :status="item.status" :status-map="freightStatusMap" />
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
                <div class="font-medium">{{ formatWeight(item.cargo_weight) }}</div>
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
                <div class="font-medium">{{ formatDate(item.created_at) }}</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Quick Subscribe Dialog -->
    <QuickSubscribeDialog
      v-model:open="showSubscribeDialog"
      :filters="currentSubscriptionFilters"
      @success="showSubscribeDialog = false"
    />
  </div>
</template>
