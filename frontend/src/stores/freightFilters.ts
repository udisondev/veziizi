import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  FreightRequestStatusFilter,
  OwnershipFilter,
  VehicleType,
  VehicleSubType,
} from '@/types/freightRequest'

export interface RoutePointFilter {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
}

export const useFreightFiltersStore = defineStore('freightFilters', () => {
  // Filter state
  const ownershipFilter = ref<OwnershipFilter>('all')
  const statusFilter = ref<FreightRequestStatusFilter>('all')
  const orgNameFilter = ref('')
  const orgINNFilter = ref('')
  const routePoints = ref<RoutePointFilter[]>([])
  const minWeight = ref<number | undefined>()
  const maxWeight = ref<number | undefined>()
  const minPrice = ref<number | undefined>()
  const maxPrice = ref<number | undefined>()
  const vehicleTypes = ref<VehicleType[]>([])
  const vehicleSubTypes = ref<VehicleSubType[]>([])

  // Computed - subscription filters (everything except ownership and status)
  const hasSubscriptionFilters = computed(() =>
    routePoints.value.length > 0 ||
    minWeight.value !== undefined ||
    maxWeight.value !== undefined ||
    minPrice.value !== undefined ||
    maxPrice.value !== undefined ||
    vehicleTypes.value.length > 0 ||
    vehicleSubTypes.value.length > 0
  )

  const hasActiveFilters = computed(() =>
    ownershipFilter.value !== 'all' ||
    statusFilter.value !== 'all' ||
    orgNameFilter.value !== '' ||
    orgINNFilter.value !== '' ||
    hasSubscriptionFilters.value
  )

  const activeFiltersCount = computed(() => {
    let count = 0
    if (ownershipFilter.value !== 'all') count++
    if (statusFilter.value !== 'all') count++
    if (orgNameFilter.value !== '') count++
    if (orgINNFilter.value !== '') count++
    if (routePoints.value.length > 0) count++
    if (minWeight.value !== undefined || maxWeight.value !== undefined) count++
    if (minPrice.value !== undefined || maxPrice.value !== undefined) count++
    if (vehicleTypes.value.length > 0) count++
    if (vehicleSubTypes.value.length > 0) count++
    return count
  })

  // Actions
  function setFilters(filters: {
    ownership?: OwnershipFilter
    status?: FreightRequestStatusFilter
    orgName?: string
    orgINN?: string
    routePoints?: RoutePointFilter[]
    minWeight?: number
    maxWeight?: number
    minPrice?: number
    maxPrice?: number
    vehicleTypes?: VehicleType[]
    vehicleSubTypes?: VehicleSubType[]
  }) {
    if (filters.ownership !== undefined) ownershipFilter.value = filters.ownership
    if (filters.status !== undefined) statusFilter.value = filters.status
    if (filters.orgName !== undefined) orgNameFilter.value = filters.orgName
    if (filters.orgINN !== undefined) orgINNFilter.value = filters.orgINN
    if (filters.routePoints !== undefined) routePoints.value = filters.routePoints
    if (filters.minWeight !== undefined) minWeight.value = filters.minWeight
    if (filters.maxWeight !== undefined) maxWeight.value = filters.maxWeight
    if (filters.minPrice !== undefined) minPrice.value = filters.minPrice
    if (filters.maxPrice !== undefined) maxPrice.value = filters.maxPrice
    if (filters.vehicleTypes !== undefined) vehicleTypes.value = filters.vehicleTypes
    if (filters.vehicleSubTypes !== undefined) vehicleSubTypes.value = filters.vehicleSubTypes
  }

  function resetFilters() {
    ownershipFilter.value = 'all'
    statusFilter.value = 'all'
    orgNameFilter.value = ''
    orgINNFilter.value = ''
    routePoints.value = []
    minWeight.value = undefined
    maxWeight.value = undefined
    minPrice.value = undefined
    maxPrice.value = undefined
    vehicleTypes.value = []
    vehicleSubTypes.value = []
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

  function updateRoutePoint(id: string, updates: Partial<RoutePointFilter>) {
    const point = routePoints.value.find(rp => rp.id === id)
    if (point) {
      Object.assign(point, updates)
    }
  }

  function reorderRoutePoints(points: RoutePointFilter[]) {
    routePoints.value = points
  }

  return {
    // State
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
  }
})
