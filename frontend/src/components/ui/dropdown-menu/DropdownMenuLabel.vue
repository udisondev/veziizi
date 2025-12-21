<script setup lang="ts">
import { computed } from 'vue'
import { DropdownMenuLabel, type DropdownMenuLabelProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<DropdownMenuLabelProps & { class?: string; inset?: boolean }>()

const classes = computed(() =>
  cn('px-2 py-1.5 text-sm font-semibold', props.inset && 'pl-8', props.class)
)
</script>

<template>
  <DropdownMenuLabel :class="classes">
    <slot />
  </DropdownMenuLabel>
</template>
