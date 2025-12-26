<script setup lang="ts">
import { computed } from 'vue'
import draggable from 'vuedraggable'
import SubscriptionRoutePointCard from './SubscriptionRoutePointCard.vue'
import { Plus } from 'lucide-vue-next'

interface RoutePointData {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
}

interface Props {
  routePoints: RoutePointData[]
}

interface Emits {
  (e: 'addPoint'): void
  (e: 'removePoint', id: string): void
  (e: 'updatePoint', id: string, updates: Partial<RoutePointData>): void
  (e: 'reorder', points: RoutePointData[]): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const localPoints = computed({
  get: () => props.routePoints,
  set: (value) => {
    // Update order after drag-n-drop
    const reordered = value.map((p, idx) => ({ ...p, order: idx }))
    emit('reorder', reordered)
  },
})

function handleUpdate(id: string, updates: Partial<RoutePointData>) {
  emit('updatePoint', id, updates)
}

function handleRemove(id: string) {
  emit('removePoint', id)
}
</script>

<template>
  <div class="space-y-3">
    <!-- Route points list with drag-n-drop -->
    <draggable
      v-if="routePoints.length > 0"
      v-model="localPoints"
      item-key="id"
      handle=".drag-handle"
      ghost-class="opacity-50"
      animation="200"
      class="space-y-3"
    >
      <template #item="{ element, index }">
        <SubscriptionRoutePointCard
          :point="element"
          :index="index"
          :can-remove="true"
          :can-move="routePoints.length > 1"
          @update="(updates) => handleUpdate(element.id, updates)"
          @remove="handleRemove(element.id)"
        />
      </template>
    </draggable>

    <!-- Add point button (always visible) -->
    <div
      class="w-full border-2 border-dashed border-muted rounded-lg text-center"
      :class="routePoints.length === 0 ? 'p-6' : 'p-4'"
    >
      <p v-if="routePoints.length === 0" class="text-sm text-muted-foreground mb-3">
        Точки маршрута не добавлены — подходят любые маршруты
      </p>
      <button
        type="button"
        class="inline-flex items-center justify-center gap-2 px-4 py-2 text-muted-foreground hover:text-primary transition-colors"
        @click="emit('addPoint')"
      >
        <Plus class="h-5 w-5" />
        <span>Добавить точку</span>
      </button>
    </div>

    <!-- Help text -->
    <p v-if="routePoints.length > 1" class="text-xs text-muted-foreground">
      Перетаскивайте карточки за иконку слева для изменения порядка точек.
    </p>
  </div>
</template>
