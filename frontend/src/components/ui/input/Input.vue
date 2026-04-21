<script setup lang="ts">
import { ref, computed } from 'vue'
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

const inputEl = ref<HTMLInputElement | null>(null)

defineExpose({ inputEl })

const classes = computed(() =>
  cn(
    'flex h-11 w-full rounded-lg border border-input bg-white px-3 py-2 text-base shadow-sm transition-colors file:border-0 file:bg-transparent file:text-base file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary disabled:cursor-not-allowed disabled:opacity-50',
    props.hasError && 'border-destructive focus-visible:border-destructive',
    props.class
  )
)

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <input
    ref="inputEl"
    :class="classes"
    :value="modelValue ?? defaultValue"
    @input="handleInput"
  />
</template>
