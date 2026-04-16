<script setup lang="ts">
import { computed } from 'vue'
import draggable from 'vuedraggable'
import type { RoutePoint } from '@/types/freightRequest'
import RoutePointCard from '../shared/RoutePointCard.vue'
import LeafletMap from '../shared/LeafletMap.vue'
import { useTutorialEvent } from '@/composables/useTutorialEvent'

const { emit: emitTutorial } = useTutorialEvent()

interface Props {
  routePoints: RoutePoint[]
  errors: Record<string, string | null>
}

interface Emits {
  (e: 'addPoint'): void
  (e: 'removePoint', index: number): void
  (e: 'updatePoint', index: number, updates: Partial<RoutePoint>): void
  (e: 'reorder', points: RoutePoint[]): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const localPoints = computed({
  get: () => props.routePoints,
  set: (value) => {
    // При drag-n-drop уведомляем родителя о новом порядке
    emit('reorder', value)
    emitTutorial('route:pointsReordered')
  },
})

function handleAddPoint() {
  emitTutorial('route:pointAdded', { newIndex: props.routePoints.length })
  emit('addPoint')
}

const hasValidCoordinates = computed(() =>
  props.routePoints.some((p) => p.coordinates)
)

// Примечание: логика автоматической установки is_loading/is_unloading
// теперь централизована в ensureRouteConstraints() в useFreightRequestForm.ts
// и вызывается в validateStep1() и loadFromRequest()
</script>

<template>
  <div class="space-y-6">
    <!-- Route error -->
    <div
      v-if="errors.route"
      class="bg-destructive/10 border border-destructive/30 text-destructive px-4 py-3 rounded-lg"
    >
      {{ errors.route }}
    </div>

    <!-- Route points list -->
    <draggable
      v-model="localPoints"
      item-key="_uid"
      handle=".drag-handle"
      ghost-class="opacity-50"
      animation="200"
      class="space-y-4"
      data-tutorial="route-points-list"
    >
      <template #item="{ element, index }">
        <RoutePointCard
          :point="element"
          :index="index"
          :total-points="routePoints.length"
          :errors="errors"
          :can-remove="routePoints.length > 2"
          :can-move="routePoints.length > 1"
          @update="emit('updatePoint', index, $event)"
          @remove="emit('removePoint', index)"
        />
      </template>
    </draggable>

    <!-- Add point button (одна кнопка) -->
    <button
      type="button"
      class="w-full flex items-center justify-center gap-2 px-4 py-3 border-2 border-dashed border-border text-muted-foreground rounded-lg hover:border-primary hover:text-primary hover:bg-primary/5 transition-colors"
      data-tutorial="route-add-point-btn"
      @click="handleAddPoint"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
        <path
          fill-rule="evenodd"
          d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
          clip-rule="evenodd"
        />
      </svg>
      Добавить точку
    </button>

    <!-- Map preview -->
    <div v-if="hasValidCoordinates">
      <h4 class="text-sm font-medium text-foreground mb-2">Маршрут на карте</h4>
      <LeafletMap :points="routePoints" />
    </div>

    <div v-else class="bg-muted/50 border border-border rounded-lg p-4 text-center text-muted-foreground">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 mx-auto mb-2 text-muted-foreground/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
      </svg>
      <p class="text-sm">Карта появится после выбора адресов с координатами</p>
    </div>

    <!-- Help text -->
    <p class="text-sm text-muted-foreground">
      Перетаскивайте карточки за иконку слева для изменения порядка точек маршрута.
      Первая точка всегда погрузка, последняя — разгрузка.
    </p>
  </div>
</template>
