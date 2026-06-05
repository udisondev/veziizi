<script setup lang="ts">
import type { RoutePoint } from '@/types/freightRequest'
import { Badge } from '@/components/ui/badge'

defineProps<{
  point: RoutePoint
  index: number
  getBadgeClass: (point: RoutePoint) => string
  getTypeLabel: (point: RoutePoint) => string
  formatDate: (date: string) => string
}>()
</script>

<template>
  <div class="py-5">
    <div class="flex items-start gap-3">
      <div
        :class="[
          'w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium shrink-0',
          point.is_loading && point.is_unloading ? 'bg-gradient-to-r from-primary from-50% to-success to-50%' :
          point.is_loading ? 'bg-primary' :
          point.is_unloading ? 'bg-success' : 'bg-muted-foreground'
        ]"
      >
        {{ index + 1 }}
      </div>
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 mb-1">
          <Badge variant="outline" :class="getBadgeClass(point)">
            {{ getTypeLabel(point) }}
          </Badge>
        </div>
        <p class="font-medium text-foreground break-words">{{ point.address }}</p>
        <p class="text-sm text-muted-foreground mt-1">
          {{ formatDate(point.date_from) }}<template v-if="point.date_to"> — {{ formatDate(point.date_to) }}</template>
          <template v-if="point.time_from">
            <span class="mx-1">·</span>
            {{ point.time_from }}<template v-if="point.time_to"> — {{ point.time_to }}</template>
          </template>
        </p>
        <p v-if="point.contact_name" class="text-sm text-muted-foreground">
          {{ point.contact_name }}<template v-if="point.contact_phone">, {{ point.contact_phone }}</template>
        </p>
        <p v-if="point.comment" class="text-sm text-muted-foreground italic mt-1 break-words">{{ point.comment }}</p>
      </div>
    </div>
  </div>
</template>
