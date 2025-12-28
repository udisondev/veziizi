<script setup lang="ts">
import { computed, ref } from 'vue'
import type { FreightSubscription } from '@/types/subscription'
import type { CargoType, BodyType, PaymentMethod, PaymentTerms, VatType } from '@/types/freightRequest'
import { useToast } from '@/components/ui/toast/use-toast'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { FiltersSummary, type FiltersData, type RoutePointDisplay } from '@/components/filters'
import { Pencil, Trash2 } from 'lucide-vue-next'

interface Props {
  subscription: FreightSubscription
}

interface Emits {
  (e: 'edit', subscription: FreightSubscription): void
  (e: 'delete', id: string): void
  (e: 'toggle-active', id: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { toast } = useToast()
const isDeleteDialogOpen = ref(false)
const isTogglingActive = ref(false)

// Convert subscription to FiltersData format
const filtersData = computed<FiltersData>(() => {
  const s = props.subscription

  // Convert route points to display format
  const routePoints: RoutePointDisplay[] | undefined = s.route_points?.length
    ? s.route_points
        .sort((a, b) => a.order - b.order)
        .map(rp => ({
          countryName: rp.country_name,
          cityName: rp.city_name,
        }))
    : undefined

  return {
    minWeight: s.min_weight,
    maxWeight: s.max_weight,
    minPrice: s.min_price,
    maxPrice: s.max_price,
    minVolume: s.min_volume,
    maxVolume: s.max_volume,
    cargoTypes: (s.cargo_types || []) as CargoType[],
    bodyTypes: (s.body_types || []) as BodyType[],
    paymentMethods: (s.payment_methods || []) as PaymentMethod[],
    paymentTerms: (s.payment_terms || []) as PaymentTerms[],
    vatTypes: (s.vat_types || []) as VatType[],
    routePoints,
  }
})

const hasFilters = computed(() => {
  const f = filtersData.value
  return !!(
    f.minWeight || f.maxWeight ||
    f.minPrice || f.maxPrice ||
    f.minVolume || f.maxVolume ||
    (f.cargoTypes?.length ?? 0) > 0 ||
    (f.bodyTypes?.length ?? 0) > 0 ||
    (f.paymentMethods?.length ?? 0) > 0 ||
    (f.paymentTerms?.length ?? 0) > 0 ||
    (f.vatTypes?.length ?? 0) > 0 ||
    (f.routePoints?.length ?? 0) > 0
  )
})

async function handleToggleActive() {
  isTogglingActive.value = true
  emit('toggle-active', props.subscription.id)
  isTogglingActive.value = false
}

function handleEdit() {
  emit('edit', props.subscription)
}

function confirmDelete() {
  isDeleteDialogOpen.value = true
}

function handleDelete() {
  emit('delete', props.subscription.id)
  isDeleteDialogOpen.value = false
  toast({
    title: 'Подписка удалена',
  })
}
</script>

<template>
  <Card :class="`transition-opacity ${!subscription.is_active ? 'opacity-60' : ''}`">
    <CardHeader class="pb-3">
      <div class="flex items-start justify-between gap-2">
        <div class="flex-1 min-w-0">
          <CardTitle class="text-base truncate">{{ subscription.name }}</CardTitle>
          <div class="flex items-center gap-2 mt-1">
            <Badge :variant="subscription.is_active ? 'default' : 'secondary'">
              {{ subscription.is_active ? 'Активна' : 'Отключена' }}
            </Badge>
            <span v-if="!hasFilters" class="text-xs text-muted-foreground">
              Все заявки
            </span>
          </div>
        </div>
        <div class="flex items-center gap-1">
          <Switch
            :checked="subscription.is_active"
            :disabled="isTogglingActive"
            @update:checked="handleToggleActive"
          />
        </div>
      </div>
    </CardHeader>
    <CardContent class="pt-0 space-y-3">
      <!-- Filters summary -->
      <FiltersSummary v-if="hasFilters" :filters="filtersData" />

      <!-- Actions -->
      <div class="flex items-center gap-2 pt-2 border-t">
        <Button variant="outline" size="sm" @click="handleEdit">
          <Pencil class="h-4 w-4 mr-1" />
          Изменить
        </Button>
        <Button variant="ghost" size="sm" class="text-destructive" @click="confirmDelete">
          <Trash2 class="h-4 w-4 mr-1" />
          Удалить
        </Button>
      </div>
    </CardContent>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model:open="isDeleteDialogOpen"
      title="Удалить подписку?"
      description="Это действие нельзя отменить. Вы больше не будете получать уведомления по этой подписке."
      confirm-text="Удалить"
      :danger="true"
      @confirm="handleDelete"
    />
  </Card>
</template>
