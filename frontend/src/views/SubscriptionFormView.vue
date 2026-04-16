<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'

// Shared Components
import { DetailPageHeader, LoadingSpinner } from '@/components/shared'

// Filter Components
import { FreightFiltersForm, type RoutePointFilter } from '@/components/filters'

const route = useRoute()
const router = useRouter()
const store = useSubscriptionsStore()
const { toast } = useToast()

const subscriptionId = computed(() => route.params.id as string | undefined)
const isEditing = computed(() => !!subscriptionId.value)

const isLoadingSubscription = ref(false)
const subscription = ref<FreightSubscription | null>(null)
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
const routePoints = ref<RoutePointFilter[]>([])

function loadSubscription(sub: FreightSubscription) {
  subscription.value = sub
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

// Route point management
function addRoutePoint() {
  routePoints.value.push({
    id: `rp-${Date.now()}`,
    countryId: undefined,
    cityId: undefined,
    order: routePoints.value.length,
  })
}

function removeRoutePoint(id: string) {
  routePoints.value = routePoints.value.filter(rp => rp.id !== id)
  routePoints.value.forEach((rp, idx) => { rp.order = idx })
}

function updateRoutePoint(id: string, updates: Partial<RoutePointFilter>) {
  const point = routePoints.value.find(rp => rp.id === id)
  if (point) Object.assign(point, updates)
}

function reorderRoutePoints(points: RoutePointFilter[]) {
  routePoints.value = points
}

const isValid = computed(() => {
  if (!name.value.trim()) return false
  for (const rp of routePoints.value) {
    if (!rp.countryId) return false
  }
  return true
})

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
    if (isEditing.value && subscription.value) {
      await store.updateSubscription(subscription.value.id, data)
      toast({ title: 'Подписка обновлена' })
    } else {
      await store.createSubscription(data)
      toast({ title: 'Подписка создана' })
    }
    router.push({ name: 'freight-subscriptions' })
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
  router.push({ name: 'freight-subscriptions' })
}

onMounted(async () => {
  if (isEditing.value && subscriptionId.value) {
    isLoadingSubscription.value = true
    try {
      const sub = await store.getSubscription(subscriptionId.value)
      if (sub) {
        loadSubscription(sub)
      } else {
        toast({ title: 'Подписка не найдена', variant: 'destructive' })
        router.push({ name: 'freight-subscriptions' })
      }
    } catch {
      toast({ title: 'Ошибка загрузки подписки', variant: 'destructive' })
      router.push({ name: 'freight-subscriptions' })
    } finally {
      isLoadingSubscription.value = false
    }
  }
})
</script>

<template>
  <div>
    <DetailPageHeader :back-to="{ name: 'freight-subscriptions' }" back-label="К подпискам" />

    <div class="max-w-2xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
      <div class="mb-6">
        <h1 class="text-2xl font-bold tracking-tight text-foreground mb-3">
          {{ isEditing ? 'Редактировать подписку' : 'Создать подписку' }}
        </h1>
        <p class="text-muted-foreground">
          Укажите критерии для получения уведомлений о заявках.
          Если параметр не указан — подходят любые значения.
        </p>
      </div>

      <LoadingSpinner v-if="isLoadingSubscription" text="Загрузка..." />

      <form v-else class="space-y-6" @submit.prevent="handleSubmit">
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

        <Separator />

        <div class="flex gap-3">
          <Button type="button" variant="outline" @click="handleCancel">
            Отмена
          </Button>
          <Button type="submit" :disabled="!isValid || isSaving">
            {{ isSaving ? 'Сохранение...' : (isEditing ? 'Сохранить' : 'Создать') }}
          </Button>
        </div>
      </form>
    </div>
  </div>
</template>
