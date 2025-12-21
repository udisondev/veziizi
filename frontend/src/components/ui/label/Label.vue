<script setup lang="ts">
import { computed } from 'vue'
import { Label as LabelPrimitive } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: string
  for?: string
}>()

const classes = computed(() =>
  cn(
    'text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
    props.class
  )
)
</script>

<template>
  <LabelPrimitive :class="classes" :for="for">
    <slot />
  </LabelPrimitive>
</template>
