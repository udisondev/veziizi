<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: string
  defaultValue?: string
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const classes = computed(() =>
  cn(
    'flex min-h-[80px] w-full rounded-lg border border-input bg-white px-3 py-2 text-base shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary disabled:cursor-not-allowed disabled:opacity-50',
    props.class
  )
)

function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <textarea
    :class="classes"
    :value="modelValue ?? defaultValue"
    @input="handleInput"
  />
</template>
