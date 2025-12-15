<script setup lang="ts">
import { computed, useSlots } from 'vue'

const props = defineProps<{
  condition: boolean
}>()

const slots = useSlots()
const hasFallback = computed(() => !!slots.fallback)
</script>

<template>
  <template v-if="props.condition">
    <slot />
  </template>
  <template v-else-if="hasFallback">
    <slot name="fallback" />
  </template>
</template>
