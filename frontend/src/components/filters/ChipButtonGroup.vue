<script setup lang="ts" generic="T extends string">
import { Label } from '@/components/ui/label'

interface Option {
  value: T
  label: string
}

interface Props {
  modelValue: T[]
  options: Option[]
  label?: string
  emptyText?: string
}

interface Emits {
  (e: 'update:modelValue', value: T[]): void
}

const props = withDefaults(defineProps<Props>(), {
  emptyText: 'Не выбрано — все варианты',
})
const emit = defineEmits<Emits>()

function toggle(value: T) {
  const current = [...props.modelValue]
  const index = current.indexOf(value)
  if (index === -1) {
    current.push(value)
  } else {
    current.splice(index, 1)
  }
  emit('update:modelValue', current)
}

function isSelected(value: T): boolean {
  return props.modelValue.includes(value)
}
</script>

<template>
  <div>
    <Label v-if="label">{{ label }}</Label>
    <div class="flex flex-wrap gap-2">
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        :class="[
          'px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
          isSelected(option.value)
            ? 'bg-accent border-primary text-accent-foreground'
            : 'bg-white border-input text-foreground hover:bg-muted',
        ]"
        @click="toggle(option.value)"
      >
        {{ option.label }}
      </button>
    </div>
    <p v-if="!modelValue.length && emptyText" class="text-xs text-muted-foreground mt-1">
      {{ emptyText }}
    </p>
  </div>
</template>
