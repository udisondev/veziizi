<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSubscriptionsStore } from '@/stores/subscriptions'
import { useToast } from '@/components/ui/toast/use-toast'
import type {
  FreightSubscription,
  FreightSubscriptionCreate,
  RoutePointCriteriaCreate,
} from '@/types/subscription'
import {
  vehicleTypeOptions,
  vehicleSubTypeLabels,
  vehicleTypeSubTypes,
  paymentMethodOptions,
  paymentTermsOptions,
  vatTypeOptions,
  type VehicleType,
  type VehicleSubType,
  type PaymentMethod,
  type PaymentTerms,
  type VatType,
} from '@/types/freightRequest'

// UI Components
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'

// Filter Components
import { ChipButtonGroup, RangeInput } from '@/components/filters'

// Subscription Components
import SubscriptionRouteStep from './SubscriptionRouteStep.vue'

interface Props {
  open: boolean
  subscription?: FreightSubscription | null
}

interface Emits {
  (e: 'update:open', value: boolean): void
  (e: 'success'): void
  (e: 'cancel'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const store = useSubscriptionsStore()
const { toast } = useToast()

const isEditing = computed(() => !!props.subscription)
const isSaving = ref(false)

// Form state
const name = ref('')
const minWeight = ref<number | undefined>()
const maxWeight = ref<number | undefined>()
const minPrice = ref<number | undefined>()
const maxPrice = ref<number | undefined>()
const minVolume = ref<number | undefined>()
const maxVolume = ref<number | undefined>()
const vehicleTypes = ref<VehicleType[]>([])
const vehicleSubTypes = ref<VehicleSubType[]>([])
const paymentMethods = ref<PaymentMethod[]>([])
const paymentTerms = ref<PaymentTerms[]>([])
const vatTypes = ref<VatType[]>([])
const isActive = ref(true)

// Computed для доступных подтипов транспорта
const availableSubTypes = computed(() => {
  if (vehicleTypes.value.length === 0) return []
  const subTypes = new Set<VehicleSubType>()
  for (const vt of vehicleTypes.value) {
    const subs = vehicleTypeSubTypes[vt] || []
    for (const sub of subs) {
      subTypes.add(sub)
    }
  }
  return Array.from(subTypes).map(value => ({
    value,
    label: vehicleSubTypeLabels[value],
  }))
})

// Route points
interface RoutePointLocal {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
}
const routePoints = ref<RoutePointLocal[]>([])

// Initialize form when dialog opens or subscription changes
watch(
  () => [props.open, props.subscription],
  () => {
    if (props.open) {
      if (props.subscription) {
        loadSubscription(props.subscription)
      } else {
        resetForm()
      }
    }
  },
  { immediate: true }
)

function loadSubscription(sub: FreightSubscription) {
  name.value = sub.name
  minWeight.value = sub.min_weight
  maxWeight.value = sub.max_weight
  minPrice.value = sub.min_price
  maxPrice.value = sub.max_price
  minVolume.value = sub.min_volume
  maxVolume.value = sub.max_volume
  vehicleTypes.value = (sub.vehicle_types || []) as VehicleType[]
  vehicleSubTypes.value = (sub.vehicle_subtypes || []) as VehicleSubType[]
  paymentMethods.value = (sub.payment_methods || []) as PaymentMethod[]
  paymentTerms.value = (sub.payment_terms || []) as PaymentTerms[]
  vatTypes.value = (sub.vat_types || []) as VatType[]
  isActive.value = sub.is_active

  routePoints.value = (sub.route_points || []).map((rp, idx) => ({
    id: `rp-${idx}-${Date.now()}`,
    countryId: rp.country_id,
    countryName: rp.country_name,
    cityId: rp.city_id,
    cityName: rp.city_name,
    order: rp.order,
  }))
}

function resetForm() {
  name.value = ''
  minWeight.value = undefined
  maxWeight.value = undefined
  minPrice.value = undefined
  maxPrice.value = undefined
  minVolume.value = undefined
  maxVolume.value = undefined
  vehicleTypes.value = []
  vehicleSubTypes.value = []
  paymentMethods.value = []
  paymentTerms.value = []
  vatTypes.value = []
  isActive.value = true
  routePoints.value = []
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
  // Reorder
  routePoints.value.forEach((rp, idx) => {
    rp.order = idx
  })
}

function updateRoutePoint(id: string, updates: Partial<RoutePointLocal>) {
  const point = routePoints.value.find(rp => rp.id === id)
  if (point) {
    Object.assign(point, updates)
  }
}

function reorderRoutePoints(points: RoutePointLocal[]) {
  routePoints.value = points
}

// Form validation
const isValid = computed(() => {
  if (!name.value.trim()) return false

  // Check route points validity
  for (const rp of routePoints.value) {
    if (!rp.countryId) return false
  }

  return true
})

// Submit
async function handleSubmit() {
  if (!isValid.value) return

  isSaving.value = true

  const routePointsData: RoutePointCriteriaCreate[] = routePoints.value
    .filter(rp => rp.countryId)
    .map((rp, idx) => ({
      country_id: rp.countryId!,
      city_id: rp.cityId,
      order: idx + 1,
    }))

  const data: FreightSubscriptionCreate = {
    name: name.value.trim(),
    min_weight: minWeight.value,
    max_weight: maxWeight.value,
    min_price: minPrice.value,
    max_price: maxPrice.value,
    min_volume: minVolume.value,
    max_volume: maxVolume.value,
    vehicle_types: vehicleTypes.value.length > 0 ? vehicleTypes.value : undefined,
    vehicle_subtypes: vehicleSubTypes.value.length > 0 ? vehicleSubTypes.value : undefined,
    payment_methods: paymentMethods.value.length > 0 ? paymentMethods.value : undefined,
    payment_terms: paymentTerms.value.length > 0 ? paymentTerms.value : undefined,
    vat_types: vatTypes.value.length > 0 ? vatTypes.value : undefined,
    route_points: routePointsData.length > 0 ? routePointsData : undefined,
    is_active: isActive.value,
  }

  try {
    if (isEditing.value && props.subscription) {
      await store.updateSubscription(props.subscription.id, data)
      toast({ title: 'Подписка обновлена' })
    } else {
      await store.createSubscription(data)
      toast({ title: 'Подписка создана' })
    }
    emit('success')
  } catch {
    toast({
      title: 'Ошибка',
      description: 'Не удалось сохранить подписку',
      variant: 'destructive',
    })
  } finally {
    isSaving.value = false
  }
}

function handleCancel() {
  emit('cancel')
}
</script>

<template>
  <Dialog :open="open" @update:open="(v: boolean) => emit('update:open', v)">
    <DialogContent class="max-w-2xl max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>
          {{ isEditing ? 'Редактировать подписку' : 'Создать подписку' }}
        </DialogTitle>
        <DialogDescription>
          Укажите критерии для получения уведомлений о заявках.
          Если параметр не указан — подходят любые значения.
        </DialogDescription>
      </DialogHeader>

      <form @submit.prevent="handleSubmit" class="space-y-6 py-4">
        <!-- Name -->
        <div>
          <Label for="name" class="text-base font-medium">Название подписки *</Label>
          <Input
            id="name"
            v-model="name"
            placeholder="Например: Россия → Казахстан, тент"
            class="mt-2"
          />
        </div>

        <Separator />

        <!-- Numeric Ranges -->
        <div class="space-y-4">
          <h4 class="font-medium">Числовые параметры</h4>

          <RangeInput
            v-model:min-value="minWeight"
            v-model:max-value="maxWeight"
            label="Вес груза, т"
            :min="0"
            step="0.1"
          />

          <RangeInput
            v-model:min-value="minPrice"
            v-model:max-value="maxPrice"
            label="Ставка, руб."
            :min="0"
            step="1000"
          />

          <RangeInput
            v-model:min-value="minVolume"
            v-model:max-value="maxVolume"
            label="Объём груза, м³"
            :min="0"
            step="1"
          />
        </div>

        <Separator />

        <!-- Route Points -->
        <SubscriptionRouteStep
          :route-points="routePoints"
          @add-point="addRoutePoint"
          @remove-point="removeRoutePoint"
          @update-point="updateRoutePoint"
          @reorder="reorderRoutePoints"
        />

        <Separator />

        <!-- Vehicle Types -->
        <ChipButtonGroup
          v-model="vehicleTypes"
          :options="vehicleTypeOptions"
          label="Типы транспорта"
          empty-text="Не выбрано — все типы транспорта"
        />

        <!-- Vehicle Sub Types -->
        <ChipButtonGroup
          v-if="availableSubTypes.length > 0"
          v-model="vehicleSubTypes"
          :options="availableSubTypes"
          label="Подтипы транспорта"
          empty-text="Не выбрано — все подтипы"
        />

        <!-- Payment Methods -->
        <ChipButtonGroup
          v-model="paymentMethods"
          :options="paymentMethodOptions"
          label="Способы оплаты"
          empty-text="Не выбрано — все способы"
        />

        <!-- Payment Terms -->
        <ChipButtonGroup
          v-model="paymentTerms"
          :options="paymentTermsOptions"
          label="Условия оплаты"
          empty-text="Не выбрано — все условия"
        />

        <!-- VAT Types -->
        <ChipButtonGroup
          v-model="vatTypes"
          :options="vatTypeOptions"
          label="НДС"
          empty-text="Не выбрано — все варианты"
        />
      </form>

      <DialogFooter>
        <Button type="button" variant="outline" @click="handleCancel">
          Отмена
        </Button>
        <Button
          type="submit"
          :disabled="!isValid || isSaving"
          @click="handleSubmit"
        >
          {{ isSaving ? 'Сохранение...' : (isEditing ? 'Сохранить' : 'Создать') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
