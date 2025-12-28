<script setup lang="ts">
import { computed } from 'vue'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  paymentMethodLabels,
  paymentTermsLabels,
  vatTypeLabels,
  type VehicleType,
  type VehicleSubType,
  type PaymentMethod,
  type PaymentTerms,
  type VatType,
} from '@/types/freightRequest'
import { MapPin, Truck, CreditCard, Scale, Box } from 'lucide-vue-next'

export interface RoutePointDisplay {
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order?: number
}

export interface FiltersData {
  routePoints?: RoutePointDisplay[]
  minWeight?: number
  maxWeight?: number
  minPrice?: number
  maxPrice?: number
  minVolume?: number
  maxVolume?: number
  vehicleTypes?: VehicleType[]
  vehicleSubTypes?: VehicleSubType[]
  paymentMethods?: PaymentMethod[]
  paymentTerms?: PaymentTerms[]
  vatTypes?: VatType[]
}

interface Props {
  filters: FiltersData
  compact?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  compact: false,
})

const hasFilters = computed(() => {
  const f = props.filters
  return !!(
    f.minWeight || f.maxWeight ||
    f.minPrice || f.maxPrice ||
    f.minVolume || f.maxVolume ||
    (f.vehicleTypes && f.vehicleTypes.length > 0) ||
    (f.vehicleSubTypes && f.vehicleSubTypes.length > 0) ||
    (f.paymentMethods && f.paymentMethods.length > 0) ||
    (f.paymentTerms && f.paymentTerms.length > 0) ||
    (f.vatTypes && f.vatTypes.length > 0) ||
    (f.routePoints && f.routePoints.length > 0)
  )
})

const weightRange = computed(() => {
  const f = props.filters
  if (!f.minWeight && !f.maxWeight) return null
  if (f.minWeight && f.maxWeight) return `${f.minWeight} - ${f.maxWeight} т`
  if (f.minWeight) return `от ${f.minWeight} т`
  return `до ${f.maxWeight} т`
})

const priceRange = computed(() => {
  const f = props.filters
  if (!f.minPrice && !f.maxPrice) return null
  const formatPrice = (p: number) => p.toLocaleString('ru-RU')
  if (f.minPrice && f.maxPrice) return `${formatPrice(f.minPrice)} - ${formatPrice(f.maxPrice)} руб.`
  if (f.minPrice) return `от ${formatPrice(f.minPrice)} руб.`
  return `до ${formatPrice(f.maxPrice!)} руб.`
})

const volumeRange = computed(() => {
  const f = props.filters
  if (!f.minVolume && !f.maxVolume) return null
  if (f.minVolume && f.maxVolume) return `${f.minVolume} - ${f.maxVolume} м³`
  if (f.minVolume) return `от ${f.minVolume} м³`
  return `до ${f.maxVolume} м³`
})

const vehicleTypesDisplay = computed(() => {
  if (!props.filters.vehicleTypes?.length) return null
  return props.filters.vehicleTypes.map(t => vehicleTypeLabels[t]).join(', ')
})

const vehicleSubTypesDisplay = computed(() => {
  if (!props.filters.vehicleSubTypes?.length) return null
  return props.filters.vehicleSubTypes.map(t => vehicleSubTypeLabels[t]).join(', ')
})

const paymentMethodsDisplay = computed(() => {
  if (!props.filters.paymentMethods?.length) return null
  return props.filters.paymentMethods.map(t => paymentMethodLabels[t]).join(', ')
})

const paymentTermsDisplay = computed(() => {
  if (!props.filters.paymentTerms?.length) return null
  return props.filters.paymentTerms.map(t => paymentTermsLabels[t]).join(', ')
})

const vatTypesDisplay = computed(() => {
  if (!props.filters.vatTypes?.length) return null
  return props.filters.vatTypes.map(t => vatTypeLabels[t]).join(', ')
})

const routeDisplay = computed(() => {
  if (!props.filters.routePoints?.length) return null
  return props.filters.routePoints
    .slice()
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
    .map(rp => {
      if (rp.cityName) return rp.cityName
      return rp.countryName || `Страна #${rp.countryId}`
    })
    .join(' → ')
})
</script>

<template>
  <div v-if="hasFilters" :class="['space-y-2 text-sm', { 'text-xs': compact }]">
    <!-- Route -->
    <div v-if="routeDisplay" class="flex items-start gap-2">
      <MapPin :class="['text-muted-foreground mt-0.5 flex-shrink-0', compact ? 'h-3 w-3' : 'h-4 w-4']" />
      <span>{{ routeDisplay }}</span>
    </div>

    <!-- Weight -->
    <div v-if="weightRange" class="flex items-center gap-2">
      <Scale :class="['text-muted-foreground flex-shrink-0', compact ? 'h-3 w-3' : 'h-4 w-4']" />
      <span>{{ weightRange }}</span>
    </div>

    <!-- Volume -->
    <div v-if="volumeRange" class="flex items-center gap-2">
      <Box :class="['text-muted-foreground flex-shrink-0', compact ? 'h-3 w-3' : 'h-4 w-4']" />
      <span>{{ volumeRange }}</span>
    </div>

    <!-- Price -->
    <div v-if="priceRange" class="flex items-center gap-2">
      <CreditCard :class="['text-muted-foreground flex-shrink-0', compact ? 'h-3 w-3' : 'h-4 w-4']" />
      <span>{{ priceRange }}</span>
    </div>

    <!-- Vehicle Types -->
    <div v-if="vehicleTypesDisplay" class="flex items-start gap-2">
      <Truck :class="['text-muted-foreground mt-0.5 flex-shrink-0', compact ? 'h-3 w-3' : 'h-4 w-4']" />
      <span class="text-muted-foreground">Транспорт:</span>
      <span>{{ vehicleTypesDisplay }}</span>
    </div>

    <!-- Vehicle Sub Types -->
    <div v-if="vehicleSubTypesDisplay" class="flex items-start gap-2">
      <Truck :class="['text-muted-foreground mt-0.5 flex-shrink-0', compact ? 'h-3 w-3' : 'h-4 w-4']" />
      <span class="text-muted-foreground">Тип кузова:</span>
      <span>{{ vehicleSubTypesDisplay }}</span>
    </div>

    <!-- Payment Methods -->
    <div v-if="paymentMethodsDisplay" class="flex items-start gap-2">
      <CreditCard :class="['text-muted-foreground mt-0.5 flex-shrink-0', compact ? 'h-3 w-3' : 'h-4 w-4']" />
      <span class="text-muted-foreground">Оплата:</span>
      <span>{{ paymentMethodsDisplay }}</span>
    </div>

    <!-- Payment Terms -->
    <div v-if="paymentTermsDisplay" :class="['text-muted-foreground', compact ? 'pl-5' : 'pl-6']">
      Условия: {{ paymentTermsDisplay }}
    </div>

    <!-- VAT Types -->
    <div v-if="vatTypesDisplay" :class="['text-muted-foreground', compact ? 'pl-5' : 'pl-6']">
      НДС: {{ vatTypesDisplay }}
    </div>
  </div>
  <div v-else class="text-sm text-muted-foreground">
    Все заявки
  </div>
</template>
