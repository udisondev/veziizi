<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { FormField } from '@/components/ui/form-field'
import { ref, watch, computed } from 'vue'
import { parseInputFloatOrUndefined } from '@/utils/inputParsers'
import type { VehicleRequirementsForm, VehicleType, VehicleSubType, LoadingType } from '@/types/freightRequest'
import {
  vehicleTypeOptions,
  getVehicleSubTypeOptions,
  allVehicleSubTypeOptions,
  getVehicleTypeForSubType,
  isSubTypeCompatible,
  loadingTypeOptions,
  loadingTypeLabels,
} from '@/types/freightRequest'
import { SelectField } from '@/components/ui/select-field'

interface Props {
  vehicle: VehicleRequirementsForm
  errors: Record<string, string | null>
}

interface Emits {
  (e: 'update:vehicle', value: VehicleRequirementsForm): void
  (e: 'validateField', field: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function handleVehicleTypeChange(value: string | null | undefined) {
  if (!value) {
    emit('update:vehicle', {
      ...props.vehicle,
      vehicle_type: undefined,
      vehicle_subtype: undefined,
    })
  } else {
    selectVehicleType(value as VehicleType)
  }
}

// Типы кузова с температурным режимом
const temperatureSubTypes: VehicleSubType[] = ['insulated', 'refrigerator']

// Показывать температурные поля только для изотермического/рефрижератора
const isTemperatureSubType = computed(() => {
  return props.vehicle.vehicle_subtype && temperatureSubTypes.includes(props.vehicle.vehicle_subtype)
})

// Галочка для температурного режима
const showTemperature = ref(!!props.vehicle.temperature)

// Получаем доступные подтипы на основе выбранного типа
const availableSubTypes = computed(() => {
  // Если тип не выбран - показываем все подтипы
  if (!props.vehicle.vehicle_type) {
    return allVehicleSubTypeOptions
  }
  // Иначе - только подтипы выбранного типа
  return getVehicleSubTypeOptions(props.vehicle.vehicle_type)
})

// Следим за галочкой: инициализируем температуру при включении, очищаем при выключении
watch(showTemperature, (show) => {
  if (show && !props.vehicle.temperature) {
    // Инициализируем пустой объект температуры для валидации
    updateField('temperature', { min: undefined as unknown as number, max: undefined as unknown as number })
  } else if (!show && props.vehicle.temperature) {
    updateField('temperature', undefined)
  }
})

// Очищаем температурные данные при смене на не-температурный тип кузова
watch(isTemperatureSubType, (isTemp) => {
  if (!isTemp) {
    showTemperature.value = false
    if (props.vehicle.temperature) {
      updateField('temperature', undefined)
    }
    if (props.vehicle.thermograph) {
      updateField('thermograph', false)
    }
  }
})

function updateField<K extends keyof VehicleRequirementsForm>(
  field: K,
  value: VehicleRequirementsForm[K]
) {
  emit('update:vehicle', { ...props.vehicle, [field]: value })
}

function selectVehicleType(type: VehicleType) {
  const currentSubType = props.vehicle.vehicle_subtype
  // Сбрасываем подтип только если он несовместим с новым типом
  const newSubType = currentSubType && isSubTypeCompatible(type, currentSubType)
    ? currentSubType
    : undefined

  emit('update:vehicle', {
    ...props.vehicle,
    vehicle_type: type,
    vehicle_subtype: newSubType as VehicleSubType,
  })
}

function selectVehicleSubType(subtype: VehicleSubType) {
  // Если тип транспорта не выбран - проставляем автоматически
  const vehicleType = props.vehicle.vehicle_type || getVehicleTypeForSubType(subtype)

  emit('update:vehicle', {
    ...props.vehicle,
    vehicle_type: vehicleType,
    vehicle_subtype: subtype,
  })
}

function toggleLoadingType(type: LoadingType) {
  const current = props.vehicle.loading_types || []
  const updated = current.includes(type)
    ? current.filter((t) => t !== type)
    : [...current, type]
  updateField('loading_types', updated)
}

function handleCapacityInput(event: Event) {
  updateField('capacity', parseInputFloatOrUndefined(event))
}

function handleVolumeInput(event: Event) {
  updateField('volume', parseInputFloatOrUndefined(event))
}

function handleDimensionInput(dimension: 'length' | 'width' | 'height', event: Event) {
  updateField(dimension, parseInputFloatOrUndefined(event))
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
    <!-- Vehicle type -->
    <FormField
      label="Тип транспорта"
      required
      :error="errors.vehicle_type"
      data-tutorial="vehicle-type"
    >
      <SelectField
        :model-value="vehicle.vehicle_type"
        :options="vehicleTypeOptions"
        :has-error="!!errors.vehicle_type"
        placeholder="Выберите тип транспорта"
        sheet-label="Тип транспорта"
        clearable
        clear-label="Не выбран"
        @update:model-value="handleVehicleTypeChange"
      />
    </FormField>

    <!-- Vehicle subtype (тип кузова) -->
    <FormField
      label="Тип кузова"
      required
      :error="errors.vehicle_subtype"
      data-tutorial="vehicle-subtype"
    >
      <SelectField
        :model-value="vehicle.vehicle_subtype"
        :options="availableSubTypes"
        :has-error="!!errors.vehicle_subtype"
        placeholder="Выберите тип кузова"
        sheet-label="Тип кузова"
        @update:model-value="selectVehicleSubType($event as VehicleSubType)"
      />
    </FormField>

    <!-- Loading types -->
    <div data-tutorial="vehicle-loading">
      <Label>
        Тип погрузки
      </Label>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="option in loadingTypeOptions"
          :key="option.value"
          type="button"
          :class="[
            'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
            vehicle.loading_types?.includes(option.value)
              ? 'bg-accent border-primary text-accent-foreground'
              : 'bg-white border-input text-foreground hover:bg-muted',
          ]"
          @click="toggleLoadingType(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
      <p v-if="vehicle.loading_types?.length" class="mt-2 text-sm text-muted-foreground">
        Выбрано: {{ vehicle.loading_types.map(t => loadingTypeLabels[t]).join(', ') }}
      </p>
    </div>

    <!-- Capacity -->
    <div data-tutorial="vehicle-capacity">
      <Label>
        Грузоподъёмность, кг
      </Label>
      <Input
        type="number"
        :value="vehicle.capacity || ''"
        placeholder="20000"
        min="0"
        step="100"
        @input="handleCapacityInput"
      />
    </div>

    <!-- Volume -->
    <div data-tutorial="vehicle-volume">
      <Label>
        Объём кузова, м³
      </Label>
      <Input
        type="number"
        :value="vehicle.volume || ''"
        placeholder="82"
        min="0"
        step="1"
        @input="handleVolumeInput"
      />
    </div>

    <!-- Dimensions -->
    <div data-tutorial="vehicle-dimensions">
      <Label>
        Размеры кузова (Д × Ш × В), м
      </Label>
      <div class="grid grid-cols-3 gap-3">
        <Input
          type="number"
          :value="vehicle.length || ''"
          placeholder="13.6"
          min="0"
          step="0.1"
          @input="handleDimensionInput('length', $event)"
        />
        <Input
          type="number"
          :value="vehicle.width || ''"
          placeholder="2.45"
          min="0"
          step="0.1"
          @input="handleDimensionInput('width', $event)"
        />
        <Input
          type="number"
          :value="vehicle.height || ''"
          placeholder="2.7"
          min="0"
          step="0.1"
          @input="handleDimensionInput('height', $event)"
        />
      </div>
    </div>

    <!-- Temperature checkbox (только для изотермического/рефрижератора) -->
    <div v-if="isTemperatureSubType" class="space-y-3" data-tutorial="vehicle-temperature">
      <div class="flex items-center gap-3">
        <input
          id="show_temperature"
          v-model="showTemperature"
          type="checkbox"
          class="h-4 w-4 accent-primary rounded"
        />
        <Label for="show_temperature" class="mb-0 inline cursor-pointer">
          Температурный режим
        </Label>
      </div>

      <!-- Temperature fields (показываются по галочке) -->
      <div v-if="showTemperature" class="pl-7" data-tutorial="vehicle-temperature-range">
        <Label>
          Диапазон температуры, °C <span class="text-destructive">*</span>
        </Label>
        <div class="flex items-center gap-3">
          <div class="flex-1">
            <Input
              type="number"
              :value="vehicle.temperature?.min ?? ''"
              placeholder="от"
              step="1"
              :has-error="!!errors.temperature_min"
              @input="handleTemperatureInput('min', $event)"
            />
            <p v-if="errors.temperature_min" class="mt-1 text-sm text-destructive">
              {{ errors.temperature_min }}
            </p>
          </div>
          <span class="text-muted-foreground">—</span>
          <div class="flex-1">
            <Input
              type="number"
              :value="vehicle.temperature?.max ?? ''"
              placeholder="до"
              step="1"
              :has-error="!!errors.temperature_max"
              @input="handleTemperatureInput('max', $event)"
            />
            <p v-if="errors.temperature_max" class="mt-1 text-sm text-destructive">
              {{ errors.temperature_max }}
            </p>
          </div>
        </div>
        <p v-if="errors.temperature" class="mt-1 text-sm text-destructive">
          {{ errors.temperature }}
        </p>
      </div>
    </div>

    <!-- Thermograph checkbox (только для изотермического/рефрижератора) -->
    <div v-if="isTemperatureSubType" class="flex items-center gap-3" data-tutorial="vehicle-thermograph">
      <input
        id="thermograph"
        :checked="vehicle.thermograph"
        type="checkbox"
        class="h-4 w-4 accent-primary rounded"
        @change="updateField('thermograph', ($event.target as HTMLInputElement).checked)"
      />
      <Label for="thermograph" class="mb-0 inline cursor-pointer">
        Термописец
      </Label>
    </div>
  </div>
</template>
