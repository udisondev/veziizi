import { useRouter } from 'vue-router'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import type { FreightSubscription } from '@/types/subscription'
import type { FreightRequestListParams } from '@/api/freightRequests'
import { vehicleTypeSubTypes } from '@/types/freightRequest'

export function subscriptionToParams(sub: FreightSubscription, limit = 5): FreightRequestListParams {
  const params: FreightRequestListParams = { statuses: 'published', limit }
  if (sub.vehicle_types?.length) params.vehicle_types = sub.vehicle_types.join(',')
  if (sub.vehicle_subtypes?.length) params.vehicle_subtypes = sub.vehicle_subtypes.join(',')
  if (sub.payment_methods?.length) params.payment_methods = sub.payment_methods.join(',')
  if (sub.payment_terms?.length) params.payment_terms = sub.payment_terms.join(',')
  if (sub.vat_types?.length) params.vat_types = sub.vat_types.join(',')
  if (sub.min_weight !== undefined) params.min_weight = sub.min_weight
  if (sub.max_weight !== undefined) params.max_weight = sub.max_weight
  if (sub.min_price !== undefined) params.min_price = sub.min_price
  if (sub.max_price !== undefined) params.max_price = sub.max_price
  if (sub.min_volume !== undefined) params.min_volume = sub.min_volume
  if (sub.max_volume !== undefined) params.max_volume = sub.max_volume
  if (sub.route_points?.length) {
    const cityIds = sub.route_points.filter(p => p.city_id).map(p => p.city_id!)
    const countryIds = sub.route_points.filter(p => !p.city_id).map(p => p.country_id)
    if (cityIds.length) params.route_city_ids = cityIds.join(',')
    if (countryIds.length) params.route_country_ids = [...new Set(countryIds)].join(',')
  }
  return params
}

export function useSubscriptionNavigation() {
  const router = useRouter()
  const filtersStore = useFreightFiltersStore()

  function applyFilters(sub: FreightSubscription) {
    filtersStore.resetFilters()
    const resolvedSubTypes = sub.vehicle_subtypes?.length
      ? sub.vehicle_subtypes
      : (sub.vehicle_types ?? []).flatMap(t => vehicleTypeSubTypes[t] ?? [])
    filtersStore.setFilters({
      vehicleSubTypes: resolvedSubTypes,
      paymentMethods: sub.payment_methods ?? [],
      paymentTerms: sub.payment_terms ?? [],
      vatTypes: sub.vat_types ?? [],
      minWeight: sub.min_weight,
      maxWeight: sub.max_weight,
      minPrice: sub.min_price,
      maxPrice: sub.max_price,
      minVolume: sub.min_volume,
      maxVolume: sub.max_volume,
      routePoints: (sub.route_points ?? []).map(rp => ({
        id: `rp-${rp.order}`,
        countryId: rp.country_id,
        countryName: rp.country_name,
        cityId: rp.city_id,
        cityName: rp.city_name,
        order: rp.order,
      })),
    })
  }

  function goToSubscriptionResults(sub: FreightSubscription) {
    applyFilters(sub)
    router.push({ name: 'freight-requests' })
  }

  return { applyFilters, goToSubscriptionResults }
}
