<script setup lang="ts">
import { ref, computed, useAttrs } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: string
  defaultValue?: string | number
  modelValue?: string | number
  hasError?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const attrs = useAttrs()
const inputEl = ref<HTMLInputElement | null>(null)

defineExpose({ inputEl })

const classes = computed(() =>
  cn(
    'flex h-11 w-full rounded-lg border border-input bg-white px-3 py-2 text-base shadow-sm transition-colors file:border-0 file:bg-transparent file:text-base file:font-medium file:text-foreground placeholder:text-muted-foreground hover:border-primary/50 focus:outline-none focus:border-primary disabled:cursor-not-allowed disabled:opacity-50',
    props.hasError && 'border-destructive focus:border-destructive',
    props.class
  )
)

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}

function handleKeydown(event: KeyboardEvent) {
  if (attrs.type !== 'number') return
  // Всегда блокируем e/E (научная нотация)
  if (event.key === 'e' || event.key === 'E') {
    event.preventDefault()
    return
  }
  // Блокируем минус если min >= 0
  const min = Number(attrs.min)
  if (event.key === '-' && !isNaN(min) && min >= 0) {
    event.preventDefault()
  }
}
</script>

<template>
  <input
    ref="inputEl"
    :class="classes"
    :value="modelValue ?? defaultValue"
    @input="handleInput"
    @keydown="handleKeydown"
  />
</template>
