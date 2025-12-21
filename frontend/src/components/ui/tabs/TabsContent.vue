<script setup lang="ts">
import { computed } from 'vue'
import { TabsContent, type TabsContentProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<TabsContentProps & { class?: string }>()

const classes = computed(() =>
  cn(
    'mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
    props.class
  )
)
</script>

<template>
  <TabsContent :class="classes" v-bind="props">
    <slot />
  </TabsContent>
</template>
