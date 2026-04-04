<script setup lang="ts">
import { ref, computed } from 'vue'
import type { CargoInfo, ADRClass } from '@/types/freightRequest'
import { adrClassOptions } from '@/types/freightRequest'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BottomSheet from '@/components/shared/BottomSheet.vue'

interface Props {
  cargo: CargoInfo
  errors: Record<string, string | null>
}

interface Emits {
  (e: 'update:cargo', value: CargoInfo): void
  (e: 'validateField', field: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { isMobile } = useBreakpoint()
const adrSheetOpen = ref(false)

const selectedAdrLabel = computed(() =>
  adrClassOptions.find(o => o.value === (props.cargo.adr_class || 'none'))?.label ?? null
)

function handleAdrSheetSelect(value: ADRClass) {
  handleAdrClassChange(value)
  adrSheetOpen.value = false
}

function updateField<K extends keyof CargoInfo>(field: K, value: CargoInfo[K]) {
  emit('update:cargo', { ...props.cargo, [field]: value })
}

function handleDescriptionInput(event: Event) {
  updateField('description', (event.target as HTMLTextAreaElement).value)
}

function handleWeightInput(event: Event) {
  const value = parseFloat((event.target as HTMLInputElement).value) || 0
  updateField('weight', value)
}

function handleVolumeInput(event: Event) {
  const value = parseFloat((event.target as HTMLInputElement).value) || undefined
  updateField('volume', value)
}

function handleQuantityInput(event: Event) {
  const value = parseInt((event.target as HTMLInputElement).value) || undefined
  updateField('quantity', value)
}

function handleAdrClassChange(value: ADRClass) {
  updateField('adr_class', value === 'none' ? undefined : value)
}

function handleDimensionInput(dimension: 'length' | 'width' | 'height', event: Event) {
  const value = parseFloat((event.target as HTMLInputElement).value) || 0
  const current = props.cargo.dimensions || { length: 0, width: 0, height: 0 }
  const updated = { ...current, [dimension]: value }

  // Если все размеры 0, очищаем dimensions
  if (updated.length === 0 && updated.width === 0 && updated.height === 0) {
    updateField('dimensions', undefined)
  } else {
    updateField('dimensions', updated)
  }
}

const inputClass = (field: string) => [
  'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500',
  props.errors[field] ? 'border-red-300' : 'border-gray-300',
]
</script>

<template>
  <div class="space-y-6">
    <!-- Description -->
    <div data-tutorial="cargo-description">
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Описание груза <span class="text-red-500">*</span>
      </label>
      <textarea
        :value="cargo.description"
        placeholder="Что перевозим? Например: Мебель, ДСП, упакованная"
        rows="3"
        :class="inputClass('description')"
        @input="handleDescriptionInput"
        @blur="emit('validateField', 'description')"
      />
      <p v-if="errors.description" class="mt-1 text-sm text-red-600">
        {{ errors.description }}
      </p>
    </div>

    <!-- Weight and Volume -->
    <div class="grid grid-cols-2 gap-4">
      <div data-tutorial="cargo-weight">
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Вес, кг <span class="text-red-500">*</span>
        </label>
        <input
          type="number"
          :value="cargo.weight || ''"
          placeholder="0"
          min="0"
          step="0.1"
          :class="inputClass('weight')"
          @input="handleWeightInput"
          @blur="emit('validateField', 'weight')"
        />
        <p v-if="errors.weight" class="mt-1 text-sm text-red-600">
          {{ errors.weight }}
        </p>
      </div>

      <div data-tutorial="cargo-volume">
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Объём, м³
        </label>
        <input
          type="number"
          :value="cargo.volume || ''"
          placeholder="0"
          min="0"
          step="0.1"
          class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          @input="handleVolumeInput"
        />
      </div>
    </div>

    <!-- Dimensions -->
    <div data-tutorial="cargo-dimensions">
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Габариты (Д × Ш × В), м
      </label>
      <div class="grid grid-cols-3 gap-3">
        <div>
          <input
            type="number"
            :value="cargo.dimensions?.length || ''"
            placeholder="Длина"
            min="0"
            step="0.01"
            class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            @input="handleDimensionInput('length', $event)"
          />
        </div>
        <div>
          <input
            type="number"
            :value="cargo.dimensions?.width || ''"
            placeholder="Ширина"
            min="0"
            step="0.01"
            class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            @input="handleDimensionInput('width', $event)"
          />
        </div>
        <div>
          <input
            type="number"
            :value="cargo.dimensions?.height || ''"
            placeholder="Высота"
            min="0"
            step="0.01"
            class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            @input="handleDimensionInput('height', $event)"
          />
        </div>
      </div>
    </div>

    <!-- Quantity -->
    <div data-tutorial="cargo-quantity">
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Количество мест <span class="text-red-500">*</span>
      </label>
      <input
        type="number"
        :value="cargo.quantity || ''"
        placeholder="1"
        min="1"
        step="1"
        :class="inputClass('quantity')"
        @input="handleQuantityInput"
        @blur="emit('validateField', 'quantity')"
      />
      <p v-if="errors.quantity" class="mt-1 text-sm text-red-600">
        {{ errors.quantity }}
      </p>
    </div>

    <!-- ADR Class -->
    <div data-tutorial="cargo-adr">
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Класс опасности груза
      </label>

      <!-- Desktop -->
      <template v-if="!isMobile()">
        <Select
          :model-value="cargo.adr_class || 'none'"
          @update:model-value="handleAdrClassChange($event as ADRClass)"
        >
          <SelectTrigger>
            <SelectValue placeholder="Не требуется" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="option in adrClassOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </template>

      <!-- Mobile -->
      <template v-else>
        <button
          type="button"
          class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-left bg-white"
          @click="adrSheetOpen = true"
        >
          <span class="text-gray-900">{{ selectedAdrLabel }}</span>
        </button>

        <BottomSheet v-model="adrSheetOpen" label="Класс опасности груза">
          <div class="overflow-y-auto flex-1">
            <button
              v-for="option in adrClassOptions"
              :key="option.value"
              type="button"
              :class="[
                'w-full px-4 py-3 text-left text-sm border-b border-gray-50 active:bg-gray-100',
                option.value === (cargo.adr_class || 'none') ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-900',
              ]"
              @click="handleAdrSheetSelect(option.value)"
            >
              {{ option.label }}
            </button>
          </div>
        </BottomSheet>
      </template>

      <p class="mt-1 text-sm text-gray-500">
        Выберите класс, если груз относится к опасным
      </p>
    </div>
  </div>
</template>
