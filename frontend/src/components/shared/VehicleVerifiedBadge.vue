<script setup lang="ts">
import { BadgeCheck } from 'lucide-vue-next'

// Публичный бейдж подтверждённости техники: публичный DTO отдаёт только флаг
// verified (статус модерации наружу не раскрывается), поэтому StatusBadge с
// vehicleStatusMap здесь неприменим.
defineProps<{
  verified: boolean
}>()
</script>

<template>
  <span
    v-if="verified"
    class="inline-flex items-center gap-1 text-xs font-medium shrink-0 text-emerald-700 dark:text-emerald-400"
  >
    <BadgeCheck class="h-3.5 w-3.5 shrink-0" />
    Подтвержден
  </span>
  <span
    v-else
    class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full shrink-0 bg-muted text-muted-foreground"
  >Не подтвержден</span>
</template>
