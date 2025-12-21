<script setup lang="ts">
import { computed } from 'vue'
import { DialogTitle, type DialogTitleProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<DialogTitleProps & { class?: string }>()

const classes = computed(() =>
  cn('text-lg font-semibold leading-none tracking-tight', props.class)
)
</script>

<template>
  <DialogTitle :class="classes">
    <slot />
  </DialogTitle>
</template>
