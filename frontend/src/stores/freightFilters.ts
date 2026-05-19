import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  OwnershipFilter,
  VehicleSubType,
  PaymentMethod,
  PaymentTerms,
  VatType,
  FreightRequestStatus,
} from '@/types/freightRequest'
import type { RoutePointRole, RoutePointData } from '@/types/routePoint'

// Дефолтные статусы для фильтра (пустой = все статусы)
const DEFAULT_STATUSES: FreightRequestStatus[] = []

type RouteParams = {
  route_city_ids?: string
  route_country_ids?: string
  origin_city_ids?: string
  origin_country_ids?: string
  destination_city_ids?: string
  destination_country_ids?: string
}

// normalizeRoleByPosition приводит роль точки к допустимой по её позиции
// в списке. Та же логика, что в UI карточки (SubscriptionRoutePointCard):
//   - одна точка        → любая роль допустима;
//   - первая (≥2 точки) → only 'origin' | 'any';
//   - последняя         → only 'destination' | 'any';
//   - средняя           → 'any'.
// Это defensive-нормализация для случаев drag-and-drop и старых сохранённых
// фильтров, где состояние могло разойтись с позицией.
function normalizeRoleByPosition(role: RoutePointRole | undefined, index: number, total: number): RoutePointRole {
  const r = role ?? 'any'
  if (total <= 1) return r
  if (index === 0) return r === 'destination' ? 'any' : r
  if (index === total - 1) return r === 'origin' ? 'any' : r
  return 'any'
}

export function buildRouteParams(points: RoutePointData[]): RouteParams {
  const result: RouteParams = {}
  if (!points.length) return result

  const total = points.length
  const normalized = points.map((p, i) => ({ ...p, role: normalizeRoleByPosition(p.role, i, total) }))

  const byRole = (role: RoutePointRole) => normalized.filter(p => p.role === role)
  const cityIds  = (pts: RoutePointData[]) => pts.filter(p => p.cityId).map(p => p.cityId!.toString())
  const countryIds = (pts: RoutePointData[]) => pts.filter(p => p.countryId && !p.cityId).map(p => p.countryId!.toString())

  const any = byRole('any')
  const ac = cityIds(any);    if (ac.length) result.route_city_ids = ac.join(',')
  const ak = countryIds(any); if (ak.length) result.route_country_ids = ak.join(',')

  const origin = byRole('origin')
  const oc = cityIds(origin);    if (oc.length) result.origin_city_ids = oc.join(',')
  const ok = countryIds(origin); if (ok.length) result.origin_country_ids = ok.join(',')

  const dest = byRole('destination')
  const dc = cityIds(dest);    if (dc.length) result.destination_city_ids = dc.join(',')
  const dk = countryIds(dest); if (dk.length) result.destination_country_ids = dk.join(',')

  return result
}


export const useFreightFiltersStore = defineStore('freightFilters', () => {
  // Filter state
  const ownershipFilter = ref<OwnershipFilter>('all')

  // Pagination state
  const cursor = ref<string | undefined>(undefined)
  const hasMore = ref(true)
  const orgINNFilter = ref('')
  const requestNumber = ref<number | null>(null)
  const statuses = ref<FreightRequestStatus[]>([...DEFAULT_STATUSES])
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
  const hasPendingOffers = ref(false)

  // Helper для проверки что статусы изменены относительно дефолта
  const isStatusesChanged = computed(() => {
    if (statuses.value.length !== DEFAULT_STATUSES.length) return true
    const sorted = [...statuses.value].sort()
    const defaultSorted = [...DEFAULT_STATUSES].sort()
    return sorted.some((s, i) => s !== defaultSorted[i])
  })

  // Computed - subscription filters (everything except ownership, INN and statuses)
  const hasSubscriptionFilters = computed(() =>
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
    vatTypes.value.length > 0
  )

  const hasActiveFilters = computed(() =>
    ownershipFilter.value !== 'all' ||
    orgINNFilter.value !== '' ||
    requestNumber.value !== null ||
    isStatusesChanged.value ||
    hasSubscriptionFilters.value
  )

  const activeFiltersCount = computed(() => {
    let count = 0
    if (ownershipFilter.value !== 'all') count++
    if (orgINNFilter.value !== '') count++
    if (requestNumber.value !== null) count++
    if (isStatusesChanged.value) count++
    if (routePoints.value.length > 0) count++
    if (minWeight.value !== undefined || maxWeight.value !== undefined) count++
    if (minPrice.value !== undefined || maxPrice.value !== undefined) count++
    if (minVolume.value !== undefined || maxVolume.value !== undefined) count++
    if (vehicleSubTypes.value.length > 0) count++
    if (paymentMethods.value.length > 0) count++
    if (paymentTerms.value.length > 0) count++
    if (vatTypes.value.length > 0) count++
    if (hasPendingOffers.value) count++
    return count
  })

  // Actions
  function setFilters(filters: {
    ownership?: OwnershipFilter
    orgINN?: string
    requestNumber?: number | null
    statuses?: FreightRequestStatus[]
    routePoints?: RoutePointData[]
    minWeight?: number
    maxWeight?: number
    minPrice?: number
    maxPrice?: number
    minVolume?: number
    maxVolume?: number
    vehicleSubTypes?: VehicleSubType[]
    paymentMethods?: PaymentMethod[]
    paymentTerms?: PaymentTerms[]
    vatTypes?: VatType[]
    hasPendingOffers?: boolean
  }) {
    if (filters.ownership !== undefined) ownershipFilter.value = filters.ownership
    if (filters.orgINN !== undefined) orgINNFilter.value = filters.orgINN
    if (filters.requestNumber !== undefined) requestNumber.value = filters.requestNumber
    if (filters.statuses !== undefined) statuses.value = filters.statuses
    if (filters.routePoints !== undefined) routePoints.value = filters.routePoints
    if (filters.minWeight !== undefined) minWeight.value = filters.minWeight
    if (filters.maxWeight !== undefined) maxWeight.value = filters.maxWeight
    if (filters.minPrice !== undefined) minPrice.value = filters.minPrice
    if (filters.maxPrice !== undefined) maxPrice.value = filters.maxPrice
    if (filters.minVolume !== undefined) minVolume.value = filters.minVolume
    if (filters.maxVolume !== undefined) maxVolume.value = filters.maxVolume
    if (filters.vehicleSubTypes !== undefined) vehicleSubTypes.value = filters.vehicleSubTypes
    if (filters.paymentMethods !== undefined) paymentMethods.value = filters.paymentMethods
    if (filters.paymentTerms !== undefined) paymentTerms.value = filters.paymentTerms
    if (filters.vatTypes !== undefined) vatTypes.value = filters.vatTypes
    if (filters.hasPendingOffers !== undefined) hasPendingOffers.value = filters.hasPendingOffers
  }

  // Pagination actions
  function resetPagination() {
    cursor.value = undefined
    hasMore.value = true
  }

  function setCursor(newCursor: string | undefined) {
    cursor.value = newCursor
  }

  function setHasMore(value: boolean) {
    hasMore.value = value
  }

  function resetFilters() {
    ownershipFilter.value = 'all'
    orgINNFilter.value = ''
    requestNumber.value = null
    statuses.value = [...DEFAULT_STATUSES]
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
    hasPendingOffers.value = false
    // Сброс пагинации при сбросе фильтров
    resetPagination()
  }

  // Route point management
  function addRoutePoint() {
    const newId = `rp-${Date.now()}`
    const order = routePoints.value.length
    routePoints.value.push({
      id: newId,
      countryId: undefined,
      cityId: undefined,
      order,
    })
  }

  function removeRoutePoint(id: string) {
    routePoints.value = routePoints.value.filter(rp => rp.id !== id)
    routePoints.value.forEach((rp, idx) => {
      rp.order = idx
    })
  }

  function updateRoutePoint(id: string, updates: Partial<RoutePointData>) {
    const point = routePoints.value.find(rp => rp.id === id)
    if (point) {
      Object.assign(point, updates)
    }
  }

  function reorderRoutePoints(points: RoutePointData[]) {
    routePoints.value = points
  }

  return {
    // State
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

    // Pagination state
    cursor,
    hasMore,

    // Computed
    hasSubscriptionFilters,
    hasActiveFilters,
    activeFiltersCount,

    // Actions
    setFilters,
    resetFilters,
    addRoutePoint,
    removeRoutePoint,
    updateRoutePoint,
    reorderRoutePoints,

    // Pagination actions
    resetPagination,
    setCursor,
    setHasMore,
  }
})
