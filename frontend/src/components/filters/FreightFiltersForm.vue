<script setup lang="ts">
import { computed } from 'vue'
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

// Опции статусов для ChipButtonGroup (без 'all')
const statusFilterOptions = Object.entries(freightRequestStatusLabels).map(([value, label]) => ({
  value: value as FreightRequestStatus,
  label,
}))

// UI Components
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'

// Filter Components
import { ChipButtonGroup, RangeInput } from '@/components/filters'
import SubscriptionRouteStep from '@/components/subscriptions/SubscriptionRouteStep.vue'

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
</script>

<template>
  <div class="space-y-6">
    <!-- Ownership (optional) -->
    <div v-if="showOwnership" class="space-y-2">
      <Label>Принадлежность</Label>
      <Select v-model="localOwnership">
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

    <!-- INN (optional) -->
    <div v-if="showINN" class="space-y-2">
      <Label>ИНН организации</Label>
      <Input
        :model-value="orgINN"
        placeholder="Поиск по ИНН"
        @update:model-value="emit('update:orgINN', $event as string)"
      />
    </div>

    <!-- Request Number (optional) -->
    <div v-if="showRequestNumber" class="space-y-2">
      <Label>Номер заявки</Label>
      <Input
        :model-value="requestNumber ?? ''"
        type="number"
        placeholder="Поиск по номеру"
        @update:model-value="emit('update:requestNumber', $event ? Number($event) : null)"
      />
    </div>

    <!-- Statuses (optional) -->
    <ChipButtonGroup
      v-if="showStatuses"
      v-model="localStatuses"
      :options="statusFilterOptions"
      label="Статус заявки"
      empty-text="Не выбрано — все статусы"
    />

    <Separator v-if="showOwnership || showINN || showRequestNumber || showStatuses" />

    <!-- Route Points -->
    <SubscriptionRouteStep
      :route-points="routePoints"
      @add-point="emit('addRoutePoint')"
      @remove-point="(id) => emit('removeRoutePoint', id)"
      @update-point="(id, updates) => emit('updateRoutePoint', id, updates)"
      @reorder="(points) => emit('reorderRoutePoints', points)"
    />

    <Separator />

    <!-- Numeric Ranges -->
    <div class="space-y-4">
      <h4 class="font-medium text-sm">Параметры груза</h4>

      <RangeInput
        :min-value="minWeight"
        :max-value="maxWeight"
        label="Вес груза, т"
        :min="0"
        step="0.1"
        @update:min-value="emit('update:minWeight', $event)"
        @update:max-value="emit('update:maxWeight', $event)"
      />

      <RangeInput
        :min-value="minPrice"
        :max-value="maxPrice"
        label="Ставка, руб."
        :min="0"
        step="1000"
        @update:min-value="emit('update:minPrice', $event)"
        @update:max-value="emit('update:maxPrice', $event)"
      />

      <RangeInput
        :min-value="minVolume"
        :max-value="maxVolume"
        label="Объём груза, м³"
        :min="0"
        step="1"
        @update:min-value="emit('update:minVolume', $event)"
        @update:max-value="emit('update:maxVolume', $event)"
      />
    </div>

    <Separator />

    <!-- Vehicle SubTypes (flat list of all 35 body types) -->
    <ChipButtonGroup
      v-model="localVehicleSubTypes"
      :options="allVehicleSubTypeOptions"
      label="Тип кузова"
      empty-text="Не выбрано — все типы кузова"
    />

    <Separator />

    <!-- Payment -->
    <div class="space-y-4">
      <h4 class="font-medium text-sm">Оплата</h4>

      <ChipButtonGroup
        v-model="localPaymentMethods"
        :options="paymentMethodOptions"
        label="Способ оплаты"
        empty-text="Не выбрано — все способы"
      />

      <ChipButtonGroup
        v-model="localPaymentTerms"
        :options="paymentTermsOptions"
        label="Условия оплаты"
        empty-text="Не выбрано — все условия"
      />

      <ChipButtonGroup
        v-model="localVatTypes"
        :options="vatTypeOptions"
        label="НДС"
        empty-text="Не выбрано — все варианты"
      />
    </div>
  </div>
</template>
