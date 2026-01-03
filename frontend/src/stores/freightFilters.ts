import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  OwnershipFilter,
  VehicleSubType,
  PaymentMethod,
  PaymentTerms,
  VatType,
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
  const orgINNFilter = ref('')
  const routePoints = ref<RoutePointFilter[]>([])
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

  // Computed - subscription filters (everything except ownership and INN)
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
    hasSubscriptionFilters.value
  )

  const activeFiltersCount = computed(() => {
    let count = 0
    if (ownershipFilter.value !== 'all') count++
    if (orgINNFilter.value !== '') count++
    if (routePoints.value.length > 0) count++
    if (minWeight.value !== undefined || maxWeight.value !== undefined) count++
    if (minPrice.value !== undefined || maxPrice.value !== undefined) count++
    if (minVolume.value !== undefined || maxVolume.value !== undefined) count++
    if (vehicleSubTypes.value.length > 0) count++
    if (paymentMethods.value.length > 0) count++
    if (paymentTerms.value.length > 0) count++
    if (vatTypes.value.length > 0) count++
    return count
  })

  // Actions
  function setFilters(filters: {
    ownership?: OwnershipFilter
    orgINN?: string
    routePoints?: RoutePointFilter[]
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
  }) {
    if (filters.ownership !== undefined) ownershipFilter.value = filters.ownership
    if (filters.orgINN !== undefined) orgINNFilter.value = filters.orgINN
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
  }

  function resetFilters() {
    ownershipFilter.value = 'all'
    orgINNFilter.value = ''
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
    orgINNFilter,
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
