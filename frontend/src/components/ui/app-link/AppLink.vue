<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { RouteLocationRaw } from 'vue-router'

defineProps<{
  to: RouteLocationRaw
}>()
</script>

<template>
  <RouterLink :to="to" class="app-link" v-bind="$attrs">
    <slot />
  </RouterLink>
</template>
