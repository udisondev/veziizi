<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import BackLink from './BackLink.vue'

withDefaults(
  defineProps<{
    backTo: RouteLocationRaw
    backLabel?: string
    useHistory?: boolean
    backTutorialId?: string
  }>(),
  {
    backLabel: 'Назад',
    useHistory: false,
  }
)
</script>

<template>
  <header class="sticky top-14 z-40 bg-card border-b">
    <div class="max-w-5xl mx-auto px-4 py-3 flex items-center justify-between">
      <BackLink :to="backTo" :label="backLabel" :use-history="useHistory" :tutorial-id="backTutorialId" />
      <slot name="actions" />
    </div>
  </header>
</template>
