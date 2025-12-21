<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  title: string
  description?: string
  class?: string
}>()

const classes = computed(() =>
  cn('flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between', props.class)
)
</script>

<template>
  <div :class="classes">
    <div>
      <h1 class="text-2xl font-bold tracking-tight text-foreground">{{ title }}</h1>
      <p v-if="description" class="text-muted-foreground">{{ description }}</p>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
