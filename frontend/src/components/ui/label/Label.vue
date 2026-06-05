<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: string
  for?: string
}>()

const classes = computed(() =>
  cn(
    'text-base font-medium leading-none mb-1.5 block peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
    props.class
  )
)
</script>

<template>
  <label :class="classes" :for="for">
    <slot />
  </label>
</template>
