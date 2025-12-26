<script setup lang="ts">
import { computed } from 'vue'
import type { FreightSubscription } from '@/types/subscription'
import {
  cargoTypeLabels,
  bodyTypeLabels,
  paymentMethodLabels,
  paymentTermsLabels,
  vatTypeLabels,
  type CargoType,
  type BodyType,
  type PaymentMethod,
  type PaymentTerms,
  type VatType,
} from '@/types/freightRequest'
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
import { Pencil, Trash2, MapPin, Package, Truck, CreditCard, Scale, Box } from 'lucide-vue-next'
import { ref } from 'vue'

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

const hasFilters = computed(() => {
  const s = props.subscription
  return !!(
    s.min_weight || s.max_weight ||
    s.min_price || s.max_price ||
    s.min_volume || s.max_volume ||
    (s.cargo_types && s.cargo_types.length > 0) ||
    (s.body_types && s.body_types.length > 0) ||
    (s.payment_methods && s.payment_methods.length > 0) ||
    (s.payment_terms && s.payment_terms.length > 0) ||
    (s.vat_types && s.vat_types.length > 0) ||
    (s.route_points && s.route_points.length > 0)
  )
})

const weightRange = computed(() => {
  const s = props.subscription
  if (!s.min_weight && !s.max_weight) return null
  if (s.min_weight && s.max_weight) return `${s.min_weight} - ${s.max_weight} т`
  if (s.min_weight) return `от ${s.min_weight} т`
  return `до ${s.max_weight} т`
})

const priceRange = computed(() => {
  const s = props.subscription
  if (!s.min_price && !s.max_price) return null
  const formatPrice = (p: number) => p.toLocaleString('ru-RU')
  if (s.min_price && s.max_price) return `${formatPrice(s.min_price)} - ${formatPrice(s.max_price)} руб.`
  if (s.min_price) return `от ${formatPrice(s.min_price)} руб.`
  return `до ${formatPrice(s.max_price!)} руб.`
})

const volumeRange = computed(() => {
  const s = props.subscription
  if (!s.min_volume && !s.max_volume) return null
  if (s.min_volume && s.max_volume) return `${s.min_volume} - ${s.max_volume} м³`
  if (s.min_volume) return `от ${s.min_volume} м³`
  return `до ${s.max_volume} м³`
})

const cargoTypesDisplay = computed(() => {
  if (!props.subscription.cargo_types?.length) return null
  return props.subscription.cargo_types.map(t => cargoTypeLabels[t as CargoType]).join(', ')
})

const bodyTypesDisplay = computed(() => {
  if (!props.subscription.body_types?.length) return null
  return props.subscription.body_types.map(t => bodyTypeLabels[t as BodyType]).join(', ')
})

const paymentMethodsDisplay = computed(() => {
  if (!props.subscription.payment_methods?.length) return null
  return props.subscription.payment_methods.map(t => paymentMethodLabels[t as PaymentMethod]).join(', ')
})

const paymentTermsDisplay = computed(() => {
  if (!props.subscription.payment_terms?.length) return null
  return props.subscription.payment_terms.map(t => paymentTermsLabels[t as PaymentTerms]).join(', ')
})

const vatTypesDisplay = computed(() => {
  if (!props.subscription.vat_types?.length) return null
  return props.subscription.vat_types.map(t => vatTypeLabels[t as VatType]).join(', ')
})

const routeDisplay = computed(() => {
  if (!props.subscription.route_points?.length) return null
  return props.subscription.route_points
    .sort((a, b) => a.order - b.order)
    .map(rp => {
      if (rp.city_name) return rp.city_name
      return rp.country_name || `Страна #${rp.country_id}`
    })
    .join(' → ')
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
  <Card :class="['transition-opacity', { 'opacity-60': !subscription.is_active }]">
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
      <div v-if="hasFilters" class="space-y-2 text-sm">
        <!-- Route -->
        <div v-if="routeDisplay" class="flex items-start gap-2">
          <MapPin class="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          <span>{{ routeDisplay }}</span>
        </div>

        <!-- Weight -->
        <div v-if="weightRange" class="flex items-center gap-2">
          <Scale class="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <span>{{ weightRange }}</span>
        </div>

        <!-- Volume -->
        <div v-if="volumeRange" class="flex items-center gap-2">
          <Box class="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <span>{{ volumeRange }}</span>
        </div>

        <!-- Price -->
        <div v-if="priceRange" class="flex items-center gap-2">
          <CreditCard class="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <span>{{ priceRange }}</span>
        </div>

        <!-- Cargo Types -->
        <div v-if="cargoTypesDisplay" class="flex items-start gap-2">
          <Package class="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          <span class="text-muted-foreground">Груз:</span>
          <span>{{ cargoTypesDisplay }}</span>
        </div>

        <!-- Body Types -->
        <div v-if="bodyTypesDisplay" class="flex items-start gap-2">
          <Truck class="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          <span class="text-muted-foreground">Кузов:</span>
          <span>{{ bodyTypesDisplay }}</span>
        </div>

        <!-- Payment Methods -->
        <div v-if="paymentMethodsDisplay" class="flex items-start gap-2">
          <CreditCard class="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          <span class="text-muted-foreground">Оплата:</span>
          <span>{{ paymentMethodsDisplay }}</span>
        </div>

        <!-- Payment Terms -->
        <div v-if="paymentTermsDisplay" class="text-xs text-muted-foreground pl-6">
          Условия: {{ paymentTermsDisplay }}
        </div>

        <!-- VAT Types -->
        <div v-if="vatTypesDisplay" class="text-xs text-muted-foreground pl-6">
          НДС: {{ vatTypesDisplay }}
        </div>
      </div>

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
