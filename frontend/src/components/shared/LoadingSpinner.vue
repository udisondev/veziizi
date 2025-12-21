<script setup lang="ts">
import { computed } from 'vue'
import { Spinner } from '@/components/ui/spinner'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    text?: string
    size?: 'sm' | 'default' | 'lg'
    class?: string
  }>(),
  {
    text: 'Загрузка...',
    size: 'default',
  }
)

const classes = computed(() =>
  cn('flex flex-col items-center justify-center gap-3 py-12', props.class)
)
</script>

<template>
  <div :class="classes">
    <Spinner :size="size" />
    <p v-if="text" class="text-sm text-muted-foreground">{{ text }}</p>
  </div>
</template>
