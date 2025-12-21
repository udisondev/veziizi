<script setup lang="ts">
import { computed } from 'vue'
import { DialogDescription, type DialogDescriptionProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<DialogDescriptionProps & { class?: string }>()

const classes = computed(() => cn('text-sm text-muted-foreground', props.class))
</script>

<template>
  <DialogDescription :class="classes">
    <slot />
  </DialogDescription>
</template>
