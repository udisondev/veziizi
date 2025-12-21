<script setup lang="ts">
import { computed } from 'vue'
import { SelectSeparator, type SelectSeparatorProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<SelectSeparatorProps & { class?: string }>()

const classes = computed(() => cn('-mx-1 my-1 h-px bg-muted', props.class))
</script>

<template>
  <SelectSeparator :class="classes" />
</template>
