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
  cn('flex flex-row items-center justify-between gap-2', props.class)
)

// Если есть default слот, используем его вместо title prop
const hasDefaultSlot = computed(() => !!slots.default)
</script>

<template>
  <div :class="classes">
    <div v-if="hasDefaultSlot" class="min-w-0">
      <slot />
    </div>
    <div v-else class="min-w-0">
      <h1 class="text-2xl font-bold tracking-tight text-foreground truncate">{{ title }}</h1>
      <p v-if="description" class="text-muted-foreground">{{ description }}</p>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <slot name="actions" />
    </div>
  </div>
</template>
