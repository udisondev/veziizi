<script setup lang="ts">
import { computed } from 'vue'
import { Separator as SeparatorPrimitive } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    class?: string
    orientation?: 'horizontal' | 'vertical'
    decorative?: boolean
  }>(),
  {
    orientation: 'horizontal',
    decorative: true,
  }
)

const classes = computed(() =>
  cn(
    'shrink-0 bg-border',
    props.orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
    props.class
  )
)
</script>

<template>
  <SeparatorPrimitive
    :class="classes"
    :orientation="orientation"
    :decorative="decorative"
  />
</template>
