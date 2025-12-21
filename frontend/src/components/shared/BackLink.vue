<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { cn } from '@/lib/utils'
import { ArrowLeft } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    to: RouteLocationRaw
    label?: string
    class?: string
  }>(),
  {
    label: 'Назад',
  }
)

const classes = computed(() =>
  cn(
    'inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors',
    props.class
  )
)
</script>

<template>
  <RouterLink :to="to" :class="classes">
    <ArrowLeft class="h-4 w-4" />
    {{ label }}
  </RouterLink>
</template>
