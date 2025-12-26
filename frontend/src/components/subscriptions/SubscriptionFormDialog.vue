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
  cargoTypeOptions,
  bodyTypeOptions,
  paymentMethodOptions,
  paymentTermsOptions,
  vatTypeOptions,
  type CargoType,
  type BodyType,
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
const cargoTypes = ref<CargoType[]>([])
const bodyTypes = ref<BodyType[]>([])
const paymentMethods = ref<PaymentMethod[]>([])
const paymentTerms = ref<PaymentTerms[]>([])
const vatTypes = ref<VatType[]>([])
const isActive = ref(true)

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
  cargoTypes.value = (sub.cargo_types || []) as CargoType[]
  bodyTypes.value = (sub.body_types || []) as BodyType[]
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
  cargoTypes.value = []
  bodyTypes.value = []
  paymentMethods.value = []
  paymentTerms.value = []
  vatTypes.value = []
  isActive.value = true
  routePoints.value = []
}

// Toggle functions for chip buttons
function toggleItem<T>(arr: T[], item: T) {
  const index = arr.indexOf(item)
  if (index === -1) {
    arr.push(item)
  } else {
    arr.splice(index, 1)
  }
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
      order: idx,
    }))

  const data: FreightSubscriptionCreate = {
    name: name.value.trim(),
    min_weight: minWeight.value,
    max_weight: maxWeight.value,
    min_price: minPrice.value,
    max_price: maxPrice.value,
    min_volume: minVolume.value,
    max_volume: maxVolume.value,
    cargo_types: cargoTypes.value.length > 0 ? cargoTypes.value : undefined,
    body_types: bodyTypes.value.length > 0 ? bodyTypes.value : undefined,
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

          <!-- Weight -->
          <div>
            <Label class="text-sm">Вес груза, т</Label>
            <div class="flex items-center gap-3 mt-1">
              <Input
                type="number"
                v-model.number="minWeight"
                placeholder="от"
                min="0"
                step="0.1"
                class="flex-1"
              />
              <span class="text-muted-foreground">—</span>
              <Input
                type="number"
                v-model.number="maxWeight"
                placeholder="до"
                min="0"
                step="0.1"
                class="flex-1"
              />
            </div>
          </div>

          <!-- Price -->
          <div>
            <Label class="text-sm">Ставка, руб.</Label>
            <div class="flex items-center gap-3 mt-1">
              <Input
                type="number"
                v-model.number="minPrice"
                placeholder="от"
                min="0"
                step="1000"
                class="flex-1"
              />
              <span class="text-muted-foreground">—</span>
              <Input
                type="number"
                v-model.number="maxPrice"
                placeholder="до"
                min="0"
                step="1000"
                class="flex-1"
              />
            </div>
          </div>

          <!-- Volume -->
          <div>
            <Label class="text-sm">Объём груза, м³</Label>
            <div class="flex items-center gap-3 mt-1">
              <Input
                type="number"
                v-model.number="minVolume"
                placeholder="от"
                min="0"
                step="1"
                class="flex-1"
              />
              <span class="text-muted-foreground">—</span>
              <Input
                type="number"
                v-model.number="maxVolume"
                placeholder="до"
                min="0"
                step="1"
                class="flex-1"
              />
            </div>
          </div>
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

        <!-- Cargo Types -->
        <div>
          <Label class="text-sm font-medium mb-2 block">Типы груза</Label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in cargoTypeOptions"
              :key="option.value"
              type="button"
              :class="[
                'px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
                cargoTypes.includes(option.value)
                  ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                  : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300',
              ]"
              @click="toggleItem(cargoTypes, option.value)"
            >
              {{ option.label }}
            </button>
          </div>
          <p v-if="!cargoTypes.length" class="text-xs text-muted-foreground mt-1">
            Не выбрано — все типы груза
          </p>
        </div>

        <!-- Body Types -->
        <div>
          <Label class="text-sm font-medium mb-2 block">Типы кузова</Label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in bodyTypeOptions"
              :key="option.value"
              type="button"
              :class="[
                'px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
                bodyTypes.includes(option.value)
                  ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                  : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300',
              ]"
              @click="toggleItem(bodyTypes, option.value)"
            >
              {{ option.label }}
            </button>
          </div>
          <p v-if="!bodyTypes.length" class="text-xs text-muted-foreground mt-1">
            Не выбрано — все типы кузова
          </p>
        </div>

        <!-- Payment Methods -->
        <div>
          <Label class="text-sm font-medium mb-2 block">Способы оплаты</Label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in paymentMethodOptions"
              :key="option.value"
              type="button"
              :class="[
                'px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
                paymentMethods.includes(option.value)
                  ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                  : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300',
              ]"
              @click="toggleItem(paymentMethods, option.value)"
            >
              {{ option.label }}
            </button>
          </div>
          <p v-if="!paymentMethods.length" class="text-xs text-muted-foreground mt-1">
            Не выбрано — все способы
          </p>
        </div>

        <!-- Payment Terms -->
        <div>
          <Label class="text-sm font-medium mb-2 block">Условия оплаты</Label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in paymentTermsOptions"
              :key="option.value"
              type="button"
              :class="[
                'px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
                paymentTerms.includes(option.value)
                  ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                  : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300',
              ]"
              @click="toggleItem(paymentTerms, option.value)"
            >
              {{ option.label }}
            </button>
          </div>
          <p v-if="!paymentTerms.length" class="text-xs text-muted-foreground mt-1">
            Не выбрано — все условия
          </p>
        </div>

        <!-- VAT Types -->
        <div>
          <Label class="text-sm font-medium mb-2 block">НДС</Label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in vatTypeOptions"
              :key="option.value"
              type="button"
              :class="[
                'px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
                vatTypes.includes(option.value)
                  ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                  : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300',
              ]"
              @click="toggleItem(vatTypes, option.value)"
            >
              {{ option.label }}
            </button>
          </div>
          <p v-if="!vatTypes.length" class="text-xs text-muted-foreground mt-1">
            Не выбрано — все варианты
          </p>
        </div>
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
