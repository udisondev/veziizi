<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getMyTickets, type TicketListItem } from '@/api/support'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// Shared Components
import { PageHeader } from '@/components/shared'

// Icons
import { Plus, Clock, MessageSquare, ChevronRight } from 'lucide-vue-next'

const router = useRouter()

const tickets = ref<TicketListItem[]>([])
const loading = ref(false)
const statusFilter = ref<string>('all')

const filteredTickets = computed(() => {
  if (statusFilter.value === 'all') return tickets.value
  if (statusFilter.value === 'open') {
    return tickets.value.filter(t => t.status !== 'closed')
  }
  return tickets.value.filter(t => t.status === statusFilter.value)
})

async function loadTickets() {
  loading.value = true
  try {
    tickets.value = await getMyTickets({ limit: 50 })
  } catch (e) {
    console.error('Failed to load tickets:', e)
  } finally {
    loading.value = false
  }
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    open: 'Открыт',
    answered: 'Отвечен',
    awaiting_reply: 'Ожидает ответа',
    closed: 'Закрыт',
  }
  return labels[status] || status
}

function getStatusVariant(status: string): 'default' | 'secondary' | 'outline' | 'destructive' {
  if (status === 'answered') return 'default'
  if (status === 'open' || status === 'awaiting_reply') return 'secondary'
  return 'outline'
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

onMounted(() => {
  loadTickets()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Мои обращения" class="mb-6">
      <template #actions>
        <Button @click="router.push('/support')">
          <Plus class="mr-2 h-4 w-4" />
          Новое обращение
        </Button>
      </template>
    </PageHeader>

    <!-- Filters -->
    <div class="flex items-center gap-4 mb-6">
      <Select v-model="statusFilter">
        <SelectTrigger class="w-48">
          <SelectValue placeholder="Статус" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Все обращения</SelectItem>
          <SelectItem value="open">Открытые</SelectItem>
          <SelectItem value="answered">Отвеченные</SelectItem>
          <SelectItem value="awaiting_reply">Ожидают ответа</SelectItem>
          <SelectItem value="closed">Закрытые</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-muted-foreground">
      Загрузка...
    </div>

    <!-- Empty state -->
    <Card v-else-if="filteredTickets.length === 0">
      <CardContent class="py-12 text-center">
        <MessageSquare class="mx-auto h-12 w-12 text-muted-foreground/50 mb-4" />
        <h3 class="text-lg font-medium mb-2">Нет обращений</h3>
        <p class="text-muted-foreground mb-4">
          {{ statusFilter === 'all' ? 'У вас пока нет обращений в поддержку' : 'Нет обращений с выбранным статусом' }}
        </p>
        <Button @click="router.push('/support')">
          <Plus class="mr-2 h-4 w-4" />
          Создать обращение
        </Button>
      </CardContent>
    </Card>

    <!-- Tickets list -->
    <div v-else class="space-y-3">
      <Card
        v-for="ticket in filteredTickets"
        :key="ticket.id"
        class="hover:bg-muted/30 transition-colors cursor-pointer"
        @click="router.push(`/support/tickets/${ticket.id}`)"
      >
        <CardContent class="py-4">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-3 mb-1">
                <span class="font-mono text-sm text-muted-foreground">#{{ ticket.ticket_number }}</span>
                <Badge :variant="getStatusVariant(ticket.status)">
                  {{ getStatusLabel(ticket.status) }}
                </Badge>
              </div>
              <h3 class="font-medium truncate">{{ ticket.subject }}</h3>
              <div class="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
                <span class="flex items-center gap-1">
                  <Clock class="h-3.5 w-3.5" />
                  Создан: {{ formatDate(ticket.created_at) }}
                </span>
                <span v-if="ticket.updated_at !== ticket.created_at" class="flex items-center gap-1">
                  Обновлён: {{ formatDate(ticket.updated_at) }}
                </span>
              </div>
            </div>
            <ChevronRight class="h-5 w-5 text-muted-foreground shrink-0" />
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
