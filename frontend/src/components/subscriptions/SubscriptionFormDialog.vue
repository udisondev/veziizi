<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSubscriptionsStore } from '@/stores/subscriptions'
import { useToast } from '@/components/ui/toast/use-toast'
import type {
  FreightSubscription,
  FreightSubscriptionCreate,
  RoutePointCriteriaCreate,
} from '@/types/subscription'
import type {
  VehicleSubType,
  PaymentMethod,
  PaymentTerms,
  VatType,
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
import { FreightFiltersForm, type RoutePointFilter } from '@/components/filters'

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
const vehicleSubTypes = ref<VehicleSubType[]>([])
const paymentMethods = ref<PaymentMethod[]>([])
const paymentTerms = ref<PaymentTerms[]>([])
const vatTypes = ref<VatType[]>([])
const isActive = ref(true)

// Route points
const routePoints = ref<RoutePointFilter[]>([])

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

function updateRoutePoint(id: string, updates: Partial<RoutePointFilter>) {
  const point = routePoints.value.find(rp => rp.id === id)
  if (point) {
    Object.assign(point, updates)
  }
}

function reorderRoutePoints(points: RoutePointFilter[]) {
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

        <!-- Filters Form -->
        <FreightFiltersForm
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
