<script setup lang="ts">
import { ref, watch } from 'vue'
import type { VehicleRequirements, BodyType, LoadingType } from '@/types/freightRequest'
import { bodyTypeOptions, loadingTypeOptions, bodyTypeLabels, loadingTypeLabels } from '@/types/freightRequest'

interface Props {
  vehicle: VehicleRequirements
  errors: Record<string, string | null>
}

interface Emits {
  (e: 'update:vehicle', value: VehicleRequirements): void
  (e: 'validateField', field: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// Галочка для температурного режима
const showTemperature = ref(!!props.vehicle.temperature)

// Следим за галочкой: инициализируем температуру при включении, очищаем при выключении
watch(showTemperature, (show) => {
  if (show && !props.vehicle.temperature) {
    // Инициализируем пустой объект температуры для валидации
    updateField('temperature', { min: undefined as unknown as number, max: undefined as unknown as number })
  } else if (!show && props.vehicle.temperature) {
    updateField('temperature', undefined)
  }
})

function updateField<K extends keyof VehicleRequirements>(
  field: K,
  value: VehicleRequirements[K]
) {
  emit('update:vehicle', { ...props.vehicle, [field]: value })
}

function toggleBodyType(type: BodyType) {
  const current = props.vehicle.body_types || []
  const updated = current.includes(type)
    ? current.filter((t) => t !== type)
    : [...current, type]
  updateField('body_types', updated)
}

function toggleLoadingType(type: LoadingType) {
  const current = props.vehicle.loading_types || []
  const updated = current.includes(type)
    ? current.filter((t) => t !== type)
    : [...current, type]
  updateField('loading_types', updated)
}

function handleCapacityInput(event: Event) {
  const value = parseFloat((event.target as HTMLInputElement).value) || undefined
  updateField('capacity', value)
}

function handleVolumeInput(event: Event) {
  const value = parseFloat((event.target as HTMLInputElement).value) || undefined
  updateField('volume', value)
}

function handleDimensionInput(dimension: 'length' | 'width' | 'height', event: Event) {
  const value = parseFloat((event.target as HTMLInputElement).value) || undefined
  updateField(dimension, value)
}

function handleRequiresAdrChange(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  updateField('requires_adr', checked)
}

function handleTemperatureInput(field: 'min' | 'max', event: Event) {
  const inputValue = (event.target as HTMLInputElement).value
  // Не обрабатываем если только минус (пользователь ещё вводит)
  if (inputValue === '' || inputValue === '-') {
    return
  }
  const value = parseFloat(inputValue)
  const current = props.vehicle.temperature || { min: 0, max: 0 }
  const updated = { ...current, [field]: isNaN(value) ? 0 : value }
  updateField('temperature', updated)
}

</script>

<template>
  <div class="space-y-6">
    <!-- Body types -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">
        Тип кузова <span class="text-red-500">*</span>
      </label>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="option in bodyTypeOptions"
          :key="option.value"
          type="button"
          :class="[
            'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
            vehicle.body_types?.includes(option.value)
              ? 'bg-blue-100 border-blue-500 text-blue-700'
              : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50',
          ]"
          @click="toggleBodyType(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
      <p v-if="errors.body_types" class="mt-1 text-sm text-red-600">
        {{ errors.body_types }}
      </p>
      <p v-if="vehicle.body_types?.length" class="mt-2 text-sm text-gray-500">
        Выбрано: {{ vehicle.body_types.map(t => bodyTypeLabels[t]).join(', ') }}
      </p>
    </div>

    <!-- Loading types -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">
        Тип погрузки
      </label>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="option in loadingTypeOptions"
          :key="option.value"
          type="button"
          :class="[
            'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
            vehicle.loading_types?.includes(option.value)
              ? 'bg-blue-100 border-blue-500 text-blue-700'
              : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50',
          ]"
          @click="toggleLoadingType(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
      <p v-if="vehicle.loading_types?.length" class="mt-2 text-sm text-gray-500">
        Выбрано: {{ vehicle.loading_types.map(t => loadingTypeLabels[t]).join(', ') }}
      </p>
    </div>

    <!-- Capacity -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Грузоподъёмность, кг
      </label>
      <input
        type="number"
        :value="vehicle.capacity || ''"
        placeholder="20000"
        min="0"
        step="100"
        class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        @input="handleCapacityInput"
      />
    </div>

    <!-- Volume -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Объём кузова, м³
      </label>
      <input
        type="number"
        :value="vehicle.volume || ''"
        placeholder="82"
        min="0"
        step="1"
        class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        @input="handleVolumeInput"
      />
    </div>

    <!-- Dimensions -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Размеры кузова (Д × Ш × В), м
      </label>
      <div class="grid grid-cols-3 gap-3">
        <div>
          <input
            type="number"
            :value="vehicle.length || ''"
            placeholder="13.6"
            min="0"
            step="0.1"
            class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            @input="handleDimensionInput('length', $event)"
          />
        </div>
        <div>
          <input
            type="number"
            :value="vehicle.width || ''"
            placeholder="2.45"
            min="0"
            step="0.1"
            class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            @input="handleDimensionInput('width', $event)"
          />
        </div>
        <div>
          <input
            type="number"
            :value="vehicle.height || ''"
            placeholder="2.7"
            min="0"
            step="0.1"
            class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            @input="handleDimensionInput('height', $event)"
          />
        </div>
      </div>
    </div>

    <!-- Temperature checkbox -->
    <div class="space-y-3">
      <div class="flex items-center gap-3">
        <input
          id="show_temperature"
          v-model="showTemperature"
          type="checkbox"
          class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
        />
        <label for="show_temperature" class="text-sm text-gray-700">
          Температурный режим
        </label>
      </div>

      <!-- Temperature fields (показываются по галочке) -->
      <div v-if="showTemperature" class="pl-7">
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Диапазон температуры, °C <span class="text-red-500">*</span>
        </label>
        <div class="flex items-center gap-3">
          <div class="flex-1">
            <input
              type="number"
              :value="vehicle.temperature?.min ?? ''"
              placeholder="от"
              step="1"
              :class="[
                'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500',
                errors.temperature_min ? 'border-red-300' : 'border-gray-300'
              ]"
              @input="handleTemperatureInput('min', $event)"
            />
            <p v-if="errors.temperature_min" class="mt-1 text-sm text-red-600">
              {{ errors.temperature_min }}
            </p>
          </div>
          <span class="text-gray-500">—</span>
          <div class="flex-1">
            <input
              type="number"
              :value="vehicle.temperature?.max ?? ''"
              placeholder="до"
              step="1"
              :class="[
                'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500',
                errors.temperature_max ? 'border-red-300' : 'border-gray-300'
              ]"
              @input="handleTemperatureInput('max', $event)"
            />
            <p v-if="errors.temperature_max" class="mt-1 text-sm text-red-600">
              {{ errors.temperature_max }}
            </p>
          </div>
        </div>
        <p v-if="errors.temperature" class="mt-1 text-sm text-red-600">
          {{ errors.temperature }}
        </p>
      </div>
    </div>

    <!-- Requires ADR -->
    <div class="flex items-center gap-3">
      <input
        id="requires_adr"
        type="checkbox"
        :checked="vehicle.requires_adr"
        class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
        @change="handleRequiresAdrChange"
      />
      <label for="requires_adr" class="text-sm text-gray-700">
        Требуется сертификация ADR (перевозка опасных грузов)
      </label>
    </div>
  </div>
</template>
