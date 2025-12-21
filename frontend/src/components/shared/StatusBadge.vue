<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'

type StatusVariant = 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' | 'outline'

const props = defineProps<{
  status: string
  statusMap?: Record<string, { label: string; variant: StatusVariant }>
}>()

// Стандартные маппинги для типичных статусов
const defaultStatusMap: Record<string, { label: string; variant: StatusVariant }> = {
  // FreightRequest статусы
  draft: { label: 'Черновик', variant: 'secondary' },
  published: { label: 'Опубликована', variant: 'info' },
  in_progress: { label: 'В работе', variant: 'warning' },
  completed: { label: 'Завершена', variant: 'success' },
  cancelled: { label: 'Отменена', variant: 'destructive' },

  // Order статусы
  created: { label: 'Создан', variant: 'info' },
  confirmed: { label: 'Подтверждён', variant: 'success' },
  loading: { label: 'Погрузка', variant: 'warning' },
  in_transit: { label: 'В пути', variant: 'warning' },
  unloading: { label: 'Разгрузка', variant: 'warning' },
  delivered: { label: 'Доставлен', variant: 'success' },

  // Offer статусы
  pending: { label: 'Ожидает', variant: 'secondary' },
  accepted: { label: 'Принят', variant: 'success' },
  rejected: { label: 'Отклонён', variant: 'destructive' },

  // Organization статусы
  active: { label: 'Активна', variant: 'success' },
  pending_review: { label: 'На проверке', variant: 'warning' },
  blocked: { label: 'Заблокирована', variant: 'destructive' },

  // Общие
  new: { label: 'Новый', variant: 'info' },
  closed: { label: 'Закрыт', variant: 'secondary' },
}

const statusInfo = computed(() => {
  const map = props.statusMap || defaultStatusMap
  return map[props.status] || { label: props.status, variant: 'default' as StatusVariant }
})
</script>

<template>
  <Badge :variant="statusInfo.variant">
    {{ statusInfo.label }}
  </Badge>
</template>
