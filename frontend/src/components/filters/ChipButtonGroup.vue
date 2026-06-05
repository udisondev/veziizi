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
    <div class="flex flex-col gap-0.5 mt-2">
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        :class="[
          'px-0 py-1 text-sm font-medium text-left transition-colors w-full',
          isSelected(option.value)
            ? 'text-foreground'
            : 'text-muted-foreground hover:text-foreground',
        ]"
        @click="toggle(option.value)"
      >
        <span class="flex items-center gap-2">
          <span
            class="flex-shrink-0 w-4 h-4 rounded border transition-colors flex items-center justify-center"
            :class="isSelected(option.value) ? 'bg-primary border-primary' : 'border-input bg-background'"
          >
            <svg v-if="isSelected(option.value)" viewBox="0 0 10 8" class="w-2.5 h-2 text-primary-foreground" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="1,4 3.5,6.5 9,1" />
            </svg>
          </span>
          {{ option.label }}
        </span>
      </button>
    </div>
  </div>
</template>
