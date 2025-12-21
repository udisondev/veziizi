<script setup lang="ts">
import { computed } from 'vue'
import type { CargoInfo, CargoType, ADRClass } from '@/types/freightRequest'
import { cargoTypeOptions, adrClassOptions } from '@/types/freightRequest'

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

function handleTypeChange(event: Event) {
  updateField('type', (event.target as HTMLSelectElement).value as CargoType)
}

function handleAdrClassChange(event: Event) {
  updateField('adr_class', (event.target as HTMLSelectElement).value as ADRClass)
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

const showAdrClass = computed(() => props.cargo.type === 'dangerous')
</script>

<template>
  <div class="space-y-6">
    <!-- Description -->
    <div>
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

    <!-- Cargo type -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Тип груза <span class="text-red-500">*</span>
      </label>
      <select
        :value="cargo.type"
        :class="inputClass('cargo_type')"
        @change="handleTypeChange"
      >
        <option v-for="option in cargoTypeOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
      <p v-if="errors.cargo_type" class="mt-1 text-sm text-red-600">
        {{ errors.cargo_type }}
      </p>
    </div>

    <!-- ADR class (only for dangerous cargo) -->
    <div v-if="showAdrClass">
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Класс опасности (ADR)
      </label>
      <select
        :value="cargo.adr_class || 'none'"
        class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        @change="handleAdrClassChange"
      >
        <option v-for="option in adrClassOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <!-- Weight and Volume -->
    <div class="grid grid-cols-2 gap-4">
      <div>
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

      <div>
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
    <div>
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
    <div>
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
  </div>
</template>
