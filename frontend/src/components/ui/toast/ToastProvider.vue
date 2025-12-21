<script setup lang="ts">
import { ToastProvider, type ToastProviderProps } from 'reka-ui'

const props = withDefaults(defineProps<ToastProviderProps>(), {
  swipeDirection: 'right',
})
</script>

<template>
  <ToastProvider v-bind="props">
    <slot />
  </ToastProvider>
</template>
