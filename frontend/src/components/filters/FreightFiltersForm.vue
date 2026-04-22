<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { vMaska } from 'maska/vue'
import type {
  VehicleSubType,
  PaymentMethod,
  PaymentTerms,
  VatType,
  OwnershipFilter,
  FreightRequestStatus,
} from '@/types/freightRequest'
import {
  allVehicleSubTypeOptions,
  paymentMethodOptions,
  paymentTermsOptions,
  vatTypeOptions,
  ownershipOptions,
  freightRequestStatusLabels,
} from '@/types/freightRequest'

// Опции статусов для MultiSelectField (без 'all')
const statusFilterOptions = Object.entries(freightRequestStatusLabels).map(([value, label]) => ({
  value: value as FreightRequestStatus,
  label,
}))

// UI Components
import { ChevronDown } from 'lucide-vue-next'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { SelectField } from '@/components/ui/select-field'

// Filter Components
import { RangeInput } from '@/components/filters'
import SubscriptionRouteStep from '@/components/subscriptions/SubscriptionRouteStep.vue'
import { MultiSelectField } from '@/components/ui/multi-select'

// Route point interface
export interface RoutePointFilter {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
}

interface Props {
  // Route
  routePoints: RoutePointFilter[]

  // Numeric ranges
  minWeight?: number
  maxWeight?: number
  minPrice?: number
  maxPrice?: number
  minVolume?: number
  maxVolume?: number

  // Vehicle
  vehicleSubTypes: VehicleSubType[]

  // Payment
  paymentMethods: PaymentMethod[]
  paymentTerms: PaymentTerms[]
  vatTypes: VatType[]

  // Optional fields (only for filters, not subscriptions)
  showOwnership?: boolean
  ownership?: OwnershipFilter
  showINN?: boolean
  orgINN?: string
  showRequestNumber?: boolean
  requestNumber?: number | null
  showStatuses?: boolean
  statuses?: FreightRequestStatus[]

  // Layout
  compact?: boolean
}

interface Emits {
  // Route
  (e: 'addRoutePoint'): void
  (e: 'removeRoutePoint', id: string): void
  (e: 'updateRoutePoint', id: string, updates: Partial<RoutePointFilter>): void
  (e: 'reorderRoutePoints', points: RoutePointFilter[]): void

  // Numeric ranges
  (e: 'update:minWeight', value: number | undefined): void
  (e: 'update:maxWeight', value: number | undefined): void
  (e: 'update:minPrice', value: number | undefined): void
  (e: 'update:maxPrice', value: number | undefined): void
  (e: 'update:minVolume', value: number | undefined): void
  (e: 'update:maxVolume', value: number | undefined): void

  // Vehicle
  (e: 'update:vehicleSubTypes', value: VehicleSubType[]): void

  // Payment
  (e: 'update:paymentMethods', value: PaymentMethod[]): void
  (e: 'update:paymentTerms', value: PaymentTerms[]): void
  (e: 'update:vatTypes', value: VatType[]): void

  // Optional fields
  (e: 'update:ownership', value: OwnershipFilter): void
  (e: 'update:orgINN', value: string): void
  (e: 'update:requestNumber', value: number | null): void
  (e: 'update:statuses', value: FreightRequestStatus[]): void
}

const props = withDefaults(defineProps<Props>(), {
  showOwnership: false,
  showINN: false,
  showRequestNumber: false,
  ownership: 'all',
  orgINN: '',
  requestNumber: null,
  showStatuses: false,
  statuses: () => ['published'],
  compact: false,
})

const emit = defineEmits<Emits>()

// Local computed for v-model bindings
const localVehicleSubTypes = computed({
  get: () => props.vehicleSubTypes,
  set: (v) => emit('update:vehicleSubTypes', v),
})

const localPaymentMethods = computed({
  get: () => props.paymentMethods,
  set: (v) => emit('update:paymentMethods', v),
})

const localPaymentTerms = computed({
  get: () => props.paymentTerms,
  set: (v) => emit('update:paymentTerms', v),
})

const localVatTypes = computed({
  get: () => props.vatTypes,
  set: (v) => emit('update:vatTypes', v),
})

const localOwnership = computed({
  get: () => props.ownership,
  set: (v) => emit('update:ownership', v as OwnershipFilter),
})

const localStatuses = computed({
  get: () => props.statuses,
  set: (v) => emit('update:statuses', v),
})

// Accordion open state — auto-open if data already present
const routeOpen = ref(props.routePoints.length > 0)
const cargoOpen = ref(
  props.minWeight !== undefined || props.maxWeight !== undefined ||
  props.minPrice !== undefined || props.maxPrice !== undefined ||
  props.minVolume !== undefined || props.maxVolume !== undefined
)
const vehicleOpen = ref(props.vehicleSubTypes.length > 0)
const paymentOpen = ref(
  props.paymentMethods.length > 0 || props.paymentTerms.length > 0 || props.vatTypes.length > 0
)

// Badge counts
const routeCount = computed(() => props.routePoints.filter(rp => rp.countryId).length)
const cargoCount = computed(() => [
  props.minWeight, props.maxWeight,
  props.minPrice, props.maxPrice,
  props.minVolume, props.maxVolume,
].filter(v => v !== undefined).length)
const vehicleCount = computed(() => props.vehicleSubTypes.length)
const paymentCount = computed(() =>
  props.paymentMethods.length + props.paymentTerms.length + props.vatTypes.length
)

// Auto-open when data arrives externally (e.g. reset then re-apply)
watch(() => props.routePoints.length, (v) => { if (v > 0) routeOpen.value = true })
watch(() => props.vehicleSubTypes.length, (v) => { if (v > 0) vehicleOpen.value = true })
watch(() => [props.paymentMethods.length, props.paymentTerms.length, props.vatTypes.length], (vals) => {
  if (vals.some(v => v > 0)) paymentOpen.value = true
})
watch(() => [props.minWeight, props.maxWeight, props.minPrice, props.maxPrice, props.minVolume, props.maxVolume], (vals) => {
  if (vals.some(v => v !== undefined)) cargoOpen.value = true
})

</script>

<template>
  <!-- ===== COMPACT MODE (filters sidebar — accordion) ===== -->
  <template v-if="compact">
    <div class="space-y-1">

      <!-- Top fields: Принадлежность, ИНН, Номер заявки, Статус -->
      <template v-if="showOwnership || showINN || showRequestNumber || showStatuses">
        <div v-if="showOwnership" class="space-y-1.5 pb-2">
          <Label class="text-sm font-semibold px-1">Принадлежность</Label>
          <SelectField v-model="localOwnership" :options="ownershipOptions" sheet-label="Принадлежность" />
        </div>
        <div v-if="showINN" class="space-y-1.5 pb-2">
          <Label class="text-sm font-semibold px-1">ИНН организации</Label>
          <Input
            v-maska
            :model-value="orgINN"
            data-maska="############"
            inputmode="numeric"
            maxlength="12"
            placeholder="10 или 12 цифр"
            @update:model-value="emit('update:orgINN', $event as string)"
          />
        </div>
        <div v-if="showRequestNumber" class="space-y-1.5 pb-2">
          <Label class="text-sm font-semibold px-1">Номер заявки</Label>
          <Input :model-value="requestNumber ?? ''" type="number" min="0" placeholder="Поиск по номеру" @update:model-value="emit('update:requestNumber', $event ? Number($event) : null)" />
        </div>
        <div v-if="showStatuses" class="space-y-1.5 pb-3">
          <Label class="text-sm font-semibold px-1">Статус заявки</Label>
          <MultiSelectField v-model="localStatuses" :options="statusFilterOptions" placeholder="Все статусы" sheet-label="Статус заявки" search-placeholder="Поиск статуса..." />
        </div>
      </template>

      <!-- Route -->
      <div>
        <button
          type="button"
          class="w-full flex items-center justify-between px-4 py-3 rounded-lg bg-muted text-left hover:bg-muted/80 transition-colors"
          @click="routeOpen = !routeOpen"
        >
          <span class="font-bold text-sm text-foreground">Маршрут</span>
          <div class="flex items-center gap-2">
            <span v-if="routeCount > 0" class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium">{{ routeCount }}</span>
            <ChevronDown class="h-4 w-4 text-muted-foreground transition-transform duration-200" :class="routeOpen ? 'rotate-180' : ''" />
          </div>
        </button>
        <div :class="['accordion-grid', routeOpen ? 'is-open' : '']">
          <div class="overflow-hidden">
            <div class="px-1 pt-3 pb-2">
              <SubscriptionRouteStep :route-points="routePoints" plain-cards @add-point="emit('addRoutePoint')" @remove-point="(id) => emit('removeRoutePoint', id)" @update-point="(id, updates) => emit('updateRoutePoint', id, updates)" @reorder="(points) => emit('reorderRoutePoints', points)" />
            </div>
          </div>
        </div>
      </div>

      <!-- Cargo -->
      <div>
        <button
          type="button"
          class="w-full flex items-center justify-between px-4 py-3 rounded-lg bg-muted text-left hover:bg-muted/80 transition-colors"
          @click="cargoOpen = !cargoOpen"
        >
          <span class="font-bold text-sm text-foreground">Параметры груза</span>
          <div class="flex items-center gap-2">
            <span v-if="cargoCount > 0" class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium">{{ cargoCount }}</span>
            <ChevronDown class="h-4 w-4 text-muted-foreground transition-transform duration-200" :class="cargoOpen ? 'rotate-180' : ''" />
          </div>
        </button>
        <div :class="['accordion-grid', cargoOpen ? 'is-open' : '']">
          <div class="overflow-hidden">
            <div class="px-1 pt-3 pb-2 grid grid-cols-1 gap-4">
              <RangeInput :min-value="minWeight" :max-value="maxWeight" label="Вес груза, т" :min="0" step="0.1" @update:min-value="emit('update:minWeight', $event)" @update:max-value="emit('update:maxWeight', $event)" />
              <RangeInput :min-value="minPrice" :max-value="maxPrice" label="Ставка, руб." :min="0" step="1000" @update:min-value="emit('update:minPrice', $event)" @update:max-value="emit('update:maxPrice', $event)" />
              <RangeInput :min-value="minVolume" :max-value="maxVolume" label="Объём груза, м³" :min="0" step="1" @update:min-value="emit('update:minVolume', $event)" @update:max-value="emit('update:maxVolume', $event)" />
            </div>
          </div>
        </div>
      </div>

      <!-- Vehicle -->
      <div>
        <button
          type="button"
          class="w-full flex items-center justify-between px-4 py-3 rounded-lg bg-muted text-left hover:bg-muted/80 transition-colors"
          @click="vehicleOpen = !vehicleOpen"
        >
          <span class="font-bold text-sm text-foreground">Тип кузова</span>
          <div class="flex items-center gap-2">
            <span v-if="vehicleCount > 0" class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium">{{ vehicleCount }}</span>
            <ChevronDown class="h-4 w-4 text-muted-foreground transition-transform duration-200" :class="vehicleOpen ? 'rotate-180' : ''" />
          </div>
        </button>
        <div :class="['accordion-grid', vehicleOpen ? 'is-open' : '']">
          <div class="overflow-hidden">
            <div class="px-1 pt-3 pb-2">
              <MultiSelectField v-model="localVehicleSubTypes" :options="allVehicleSubTypeOptions" placeholder="Все типы кузова" sheet-label="Тип кузова" search-placeholder="Поиск типа кузова..." />
            </div>
          </div>
        </div>
      </div>

      <!-- Payment -->
      <div>
        <button
          type="button"
          class="w-full flex items-center justify-between px-4 py-3 rounded-lg bg-muted text-left hover:bg-muted/80 transition-colors"
          @click="paymentOpen = !paymentOpen"
        >
          <span class="font-bold text-sm text-foreground">Оплата</span>
          <div class="flex items-center gap-2">
            <span v-if="paymentCount > 0" class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium">{{ paymentCount }}</span>
            <ChevronDown class="h-4 w-4 text-muted-foreground transition-transform duration-200" :class="paymentOpen ? 'rotate-180' : ''" />
          </div>
        </button>
        <div :class="['accordion-grid', paymentOpen ? 'is-open' : '']">
          <div class="overflow-hidden">
            <div class="px-1 pt-3 pb-2 grid grid-cols-1 gap-4">
              <div class="space-y-1.5">
                <Label class="text-sm font-semibold px-1">Способ оплаты</Label>
                <MultiSelectField v-model="localPaymentMethods" :options="paymentMethodOptions" placeholder="Все способы" sheet-label="Способ оплаты" search-placeholder="Поиск..." />
              </div>
              <div class="space-y-1.5">
                <Label class="text-sm font-semibold px-1">Условия оплаты</Label>
                <MultiSelectField v-model="localPaymentTerms" :options="paymentTermsOptions" placeholder="Все условия" sheet-label="Условия оплаты" search-placeholder="Поиск..." />
              </div>
              <div class="space-y-1.5">
                <Label class="text-sm font-semibold px-1">НДС</Label>
                <MultiSelectField v-model="localVatTypes" :options="vatTypeOptions" placeholder="Все варианты" sheet-label="НДС" search-placeholder="Поиск..." />
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </template>

  <!-- ===== FULL MODE (subscription form — original layout) ===== -->
  <template v-else>
    <div class="space-y-6">
      <!-- Route Points -->
      <SubscriptionRouteStep
        :route-points="routePoints"
        @add-point="emit('addRoutePoint')"
        @remove-point="(id) => emit('removeRoutePoint', id)"
        @update-point="(id, updates) => emit('updateRoutePoint', id, updates)"
        @reorder="(points) => emit('reorderRoutePoints', points)"
      />

      <Separator />

      <!-- Cargo Params -->
      <div class="space-y-4">
        <h4 class="font-semibold text-lg">Параметры груза</h4>
        <RangeInput :min-value="minWeight" :max-value="maxWeight" label="Вес груза, т" :min="0" step="0.1" @update:min-value="emit('update:minWeight', $event)" @update:max-value="emit('update:maxWeight', $event)" />
        <RangeInput :min-value="minPrice" :max-value="maxPrice" label="Ставка, руб." :min="0" step="1000" @update:min-value="emit('update:minPrice', $event)" @update:max-value="emit('update:maxPrice', $event)" />
        <RangeInput :min-value="minVolume" :max-value="maxVolume" label="Объём груза, м³" :min="0" step="1" @update:min-value="emit('update:minVolume', $event)" @update:max-value="emit('update:maxVolume', $event)" />
      </div>

      <Separator />

      <!-- Vehicle SubTypes -->
      <div class="space-y-2">
        <h4 class="font-semibold text-lg">Тип кузова</h4>
        <MultiSelectField v-model="localVehicleSubTypes" :options="allVehicleSubTypeOptions" placeholder="Все типы кузова" sheet-label="Тип кузова" search-placeholder="Поиск типа кузова..." />
      </div>

      <Separator />

      <!-- Payment -->
      <div class="space-y-4">
        <h4 class="font-semibold text-lg">Оплата</h4>
        <div class="space-y-2">
          <Label>Способ оплаты</Label>
          <MultiSelectField v-model="localPaymentMethods" :options="paymentMethodOptions" placeholder="Все способы" sheet-label="Способ оплаты" search-placeholder="Поиск..." />
        </div>
        <div class="space-y-2">
          <Label>Условия оплаты</Label>
          <MultiSelectField v-model="localPaymentTerms" :options="paymentTermsOptions" placeholder="Все условия" sheet-label="Условия оплаты" search-placeholder="Поиск..." />
        </div>
        <div class="space-y-2">
          <Label>НДС</Label>
          <MultiSelectField v-model="localVatTypes" :options="vatTypeOptions" placeholder="Все варианты" sheet-label="НДС" search-placeholder="Поиск..." />
        </div>
      </div>
    </div>
  </template>
</template>
