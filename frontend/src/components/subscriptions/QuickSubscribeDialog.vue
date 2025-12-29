<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useSubscriptionsStore } from '@/stores/subscriptions'
import { useToast } from '@/components/ui/toast/use-toast'
import type { VehicleType, VehicleSubType } from '@/types/freightRequest'
import { vehicleTypeLabels, vehicleSubTypeLabels } from '@/types/freightRequest'
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
import { FiltersSummary, type FiltersData } from '@/components/filters'

interface RoutePointLocal {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
}

interface Filters {
  routePoints?: RoutePointLocal[]
  minWeight?: number
  maxWeight?: number
  minPrice?: number
  maxPrice?: number
  vehicleTypes: VehicleType[]
  vehicleSubTypes: VehicleSubType[]
}

interface Props {
  open: boolean
  filters: Filters
}

interface Emits {
  (e: 'update:open', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const store = useSubscriptionsStore()
const { toast } = useToast()

const subscriptionName = ref('')
const isSaving = ref(false)

// Reset name when dialog opens
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    subscriptionName.value = generateDefaultName()
  }
})

function generateDefaultName(): string {
  const parts: string[] = []

  // Route points
  if (props.filters.routePoints?.length) {
    const validPoints = props.filters.routePoints.filter(rp => rp.countryName)
    if (validPoints.length > 0) {
      const routeStr = validPoints
        .map(rp => rp.cityName || rp.countryName)
        .join(' → ')
      parts.push(routeStr)
    }
  }

  // Weight range
  if (props.filters.minWeight || props.filters.maxWeight) {
    if (props.filters.minWeight && props.filters.maxWeight) {
      parts.push(`${props.filters.minWeight}-${props.filters.maxWeight}т`)
    } else if (props.filters.minWeight) {
      parts.push(`от ${props.filters.minWeight}т`)
    } else {
      parts.push(`до ${props.filters.maxWeight}т`)
    }
  }

  // Vehicle types (first one)
  const firstVehicleType = props.filters.vehicleTypes[0]
  if (firstVehicleType) {
    const firstTypeLabel = vehicleTypeLabels[firstVehicleType]
    if (props.filters.vehicleTypes.length > 1) {
      parts.push(`${firstTypeLabel} +${props.filters.vehicleTypes.length - 1}`)
    } else {
      parts.push(firstTypeLabel)
    }
  }

  // Vehicle subtypes (first one)
  const firstVehicleSubType = props.filters.vehicleSubTypes[0]
  if (firstVehicleSubType) {
    const firstSubTypeLabel = vehicleSubTypeLabels[firstVehicleSubType]
    if (props.filters.vehicleSubTypes.length > 1) {
      parts.push(`${firstSubTypeLabel} +${props.filters.vehicleSubTypes.length - 1}`)
    } else {
      parts.push(firstSubTypeLabel)
    }
  }

  return parts.length > 0 ? parts.join(', ') : 'Моя подписка'
}

// Convert props filters to FiltersSummary format
const filtersForSummary = computed<FiltersData>(() => ({
  routePoints: props.filters.routePoints?.map((rp, idx) => ({
    countryName: rp.countryName,
    cityName: rp.cityName,
    order: idx,
  })),
  minWeight: props.filters.minWeight,
  maxWeight: props.filters.maxWeight,
  minPrice: props.filters.minPrice,
  maxPrice: props.filters.maxPrice,
  vehicleTypes: props.filters.vehicleTypes,
  vehicleSubTypes: props.filters.vehicleSubTypes,
}))

const isValid = computed(() => subscriptionName.value.trim().length > 0)

async function handleSubmit() {
  if (!isValid.value) return

  isSaving.value = true

  try {
    await store.createSubscription({
      name: subscriptionName.value.trim(),
      route_points: props.filters.routePoints?.length
        ? props.filters.routePoints
            .filter(rp => rp.countryId)
            .map((rp, idx) => ({
              country_id: rp.countryId!,
              city_id: rp.cityId,
              order: idx + 1,
            }))
        : undefined,
      min_weight: props.filters.minWeight,
      max_weight: props.filters.maxWeight,
      min_price: props.filters.minPrice,
      max_price: props.filters.maxPrice,
      vehicle_types: props.filters.vehicleTypes.length > 0 ? props.filters.vehicleTypes : undefined,
      vehicle_subtypes: props.filters.vehicleSubTypes.length > 0 ? props.filters.vehicleSubTypes : undefined,
      is_active: true,
    })

    toast({
      title: 'Подписка создана',
      description: 'Вы будете получать уведомления о новых заявках',
    })

    emit('success')
    emit('update:open', false)
  } catch {
    toast({
      title: 'Ошибка',
      description: 'Не удалось создать подписку',
      variant: 'destructive',
    })
  } finally {
    isSaving.value = false
  }
}

function handleCancel() {
  emit('update:open', false)
}
</script>

<template>
  <Dialog :open="open" @update:open="(v: boolean) => emit('update:open', v)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Подписаться на заявки?</DialogTitle>
        <DialogDescription>
          Вы будете получать уведомления о новых заявках, соответствующих выбранным фильтрам.
        </DialogDescription>
      </DialogHeader>

      <form @submit.prevent="handleSubmit" class="space-y-4 py-4">
        <!-- Subscription Name -->
        <div>
          <Label for="sub-name">Название подписки *</Label>
          <Input
            id="sub-name"
            v-model="subscriptionName"
            placeholder="Например: Тент 10-20т"
            class="mt-1.5"
          />
        </div>

        <!-- Filters Summary -->
        <div class="p-3 bg-muted/50 rounded-lg">
          <Label class="text-xs text-muted-foreground mb-2 block">Выбранные фильтры</Label>
          <FiltersSummary :filters="filtersForSummary" :compact="true" />
        </div>
      </form>

      <DialogFooter class="gap-2 sm:gap-0">
        <Button type="button" variant="outline" @click="handleCancel">
          Отмена
        </Button>
        <Button
          type="submit"
          :disabled="!isValid || isSaving"
          @click="handleSubmit"
        >
          {{ isSaving ? 'Создание...' : 'Создать подписку' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
