<script setup lang="ts">
import { computed, useSlots } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  title?: string
  description?: string
  class?: string
}>()

const slots = useSlots()

const classes = computed(() =>
  cn('flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between', props.class)
)

// Если есть default слот, используем его вместо title prop
const hasDefaultSlot = computed(() => !!slots.default)
</script>

<template>
  <div :class="classes">
    <div v-if="hasDefaultSlot">
      <slot />
    </div>
    <div v-else>
      <h1 class="text-2xl font-bold tracking-tight text-foreground">{{ title }}</h1>
      <p v-if="description" class="text-muted-foreground">{{ description }}</p>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
