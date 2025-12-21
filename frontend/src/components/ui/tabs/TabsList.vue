<script setup lang="ts">
import { computed } from 'vue'
import { TabsList, type TabsListProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<TabsListProps & { class?: string }>()

const classes = computed(() =>
  cn(
    'inline-flex h-10 items-center justify-center rounded-md bg-muted p-1 text-muted-foreground',
    props.class
  )
)
</script>

<template>
  <TabsList :class="classes">
    <slot />
  </TabsList>
</template>
