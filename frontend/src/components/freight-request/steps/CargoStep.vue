<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { FormField } from '@/components/ui/form-field'
import type { CargoInfo, ADRClass } from '@/types/freightRequest'
import { adrClassOptions } from '@/types/freightRequest'
import { SelectField } from '@/components/ui/select-field'
import { parseInputFloat, parseInputFloatOrUndefined, parseInputInt } from '@/utils/inputParsers'

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
  updateField('weight', parseInputFloat(event))
}

function handleVolumeInput(event: Event) {
  updateField('volume', parseInputFloatOrUndefined(event))
}

function handleQuantityInput(event: Event) {
  updateField('quantity', parseInputInt(event))
}

function handleAdrClassChange(value: ADRClass) {
  updateField('adr_class', value === 'none' ? undefined : value)
}

function handleDimensionInput(dimension: 'length' | 'width' | 'height', event: Event) {
  const value = parseInputFloat(event)
  const current = props.cargo.dimensions || { length: 0, width: 0, height: 0 }
  const updated = { ...current, [dimension]: value }

  if (updated.length === 0 && updated.width === 0 && updated.height === 0) {
    updateField('dimensions', undefined)
  } else {
    updateField('dimensions', updated)
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Description -->
    <FormField
      data-tutorial="cargo-description"
      label="Описание груза"
      required
      :error="errors.description"
    >
      <Textarea
        :value="cargo.description"
        placeholder="Что перевозим? Например: Мебель, ДСП, упакованная"
        rows="3"
        :class="errors.description ? 'border-destructive' : ''"
        @input="handleDescriptionInput"
        @blur="emit('validateField', 'description')"
      />
    </FormField>

    <!-- Weight and Volume -->
    <div class="grid grid-cols-2 gap-4">
      <FormField
        data-tutorial="cargo-weight"
        label="Вес, кг"
        required
        :error="errors.weight"
      >
        <Input
          type="number"
          :value="cargo.weight || ''"
          placeholder="0"
          min="0"
          step="0.1"
          :has-error="!!errors.weight"
          @input="handleWeightInput"
          @blur="emit('validateField', 'weight')"
        />
      </FormField>

      <FormField data-tutorial="cargo-volume" label="Объём, м³">
        <Input
          type="number"
          :value="cargo.volume || ''"
          placeholder="0"
          min="0"
          step="0.1"
          @input="handleVolumeInput"
        />
      </FormField>
    </div>

    <!-- Dimensions -->
    <div data-tutorial="cargo-dimensions">
      <Label>Габариты (Д × Ш × В), м</Label>
      <div class="grid grid-cols-3 gap-3">
        <Input
          type="number"
          :value="cargo.dimensions?.length || ''"
          placeholder="Длина"
          min="0"
          step="0.01"
          @input="handleDimensionInput('length', $event)"
        />
        <Input
          type="number"
          :value="cargo.dimensions?.width || ''"
          placeholder="Ширина"
          min="0"
          step="0.01"
          @input="handleDimensionInput('width', $event)"
        />
        <Input
          type="number"
          :value="cargo.dimensions?.height || ''"
          placeholder="Высота"
          min="0"
          step="0.01"
          @input="handleDimensionInput('height', $event)"
        />
      </div>
    </div>

    <!-- Quantity -->
    <FormField
      data-tutorial="cargo-quantity"
      label="Количество мест"
      required
      :error="errors.quantity"
    >
      <Input
        type="number"
        :value="cargo.quantity || ''"
        placeholder="1"
        min="1"
        step="1"
        :has-error="!!errors.quantity"
        @input="handleQuantityInput"
        @blur="emit('validateField', 'quantity')"
      />
    </FormField>

    <!-- ADR Class -->
    <FormField
      data-tutorial="cargo-adr"
      label="Класс опасности груза"
      hint="Выберите класс, если груз относится к опасным"
    >
      <SelectField
        :model-value="cargo.adr_class || 'none'"
        :options="adrClassOptions"
        placeholder="Не требуется"
        sheet-label="Класс опасности груза"
        @update:model-value="handleAdrClassChange($event as ADRClass)"
      />
    </FormField>
  </div>
</template>
