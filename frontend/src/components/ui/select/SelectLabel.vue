<script setup lang="ts">
import { computed } from 'vue'
import { SelectLabel, type SelectLabelProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<SelectLabelProps & { class?: string }>()

const classes = computed(() => cn('py-1.5 pl-8 pr-2 text-sm font-semibold', props.class))
</script>

<template>
  <SelectLabel :class="classes">
    <slot />
  </SelectLabel>
</template>
