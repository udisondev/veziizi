<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: string
  interactive?: boolean
}>()

const classes = computed(() =>
  cn(
    'rounded-xl border bg-card text-card-foreground shadow-md',
    props.interactive && 'cursor-pointer transition-all duration-200 hover:scale-[1.02] hover:shadow-xl',
    props.class
  )
)
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>
