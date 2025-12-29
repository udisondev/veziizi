<script setup lang="ts">
import type { Order } from '@/types/order'
import { formatDateTime } from '@/utils/formatters'

// UI Components
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'

// Icons
import { Star, Clock } from 'lucide-vue-next'

interface Props {
  order: Order
}

defineProps<Props>()
</script>

<template>
  <div>
    <div v-if="order.reviews.length === 0" class="text-center text-muted-foreground py-8">
      Отзывов пока нет
    </div>

    <div v-else class="space-y-4">
      <Card
        v-for="review in order.reviews"
        :key="review.id"
      >
        <CardContent class="p-4">
          <div class="flex items-center gap-2 mb-2">
            <div class="flex">
              <Star
                v-for="star in 5"
                :key="star"
                :class="[
                  'h-5 w-5',
                  star <= review.rating ? 'text-warning fill-warning' : 'text-muted-foreground'
                ]"
              />
            </div>
            <Badge variant="secondary">
              {{ review.reviewer_org_id === order.customer_org_id ? 'Заказчик' : 'Перевозчик' }}
            </Badge>
          </div>
          <p v-if="review.comment" class="text-foreground break-words">{{ review.comment }}</p>
          <p v-else class="text-muted-foreground italic">Без комментария</p>
          <p class="text-xs text-muted-foreground mt-2 flex items-center gap-1">
            <Clock class="h-3 w-3" />
            {{ formatDateTime(review.created_at) }}
          </p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
