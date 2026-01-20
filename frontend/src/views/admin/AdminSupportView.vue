<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi, type AdminSupportTicket } from '@/api/admin'

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
import { ErrorBanner } from '@/components/shared'

// Icons
import {
  Building2,
  RefreshCcw,
  LogOut,
  Star,
  AlertTriangle,
  Headphones,
  Clock,
  ChevronRight,
  Mail,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const admin = useAdminStore()

const tickets = ref<AdminSupportTicket[]>([])
const total = ref(0)
const isLoading = ref(true)
const error = ref('')
const statusFilter = ref('all')

const navItems = [
  { to: '/admin/organizations', label: 'Организации', icon: Building2 },
  { to: '/admin/reviews', label: 'Отзывы', icon: Star },
  { to: '/admin/fraudsters', label: 'Накрутчики', icon: AlertTriangle },
  { to: '/admin/support', label: 'Поддержка', icon: Headphones },
  { to: '/admin/email-templates', label: 'Email шаблоны', icon: Mail },
]

const filteredTickets = computed(() => {
  if (statusFilter.value === 'all') return tickets.value
  if (statusFilter.value === 'open') {
    return tickets.value.filter(t => t.status !== 'closed')
  }
  return tickets.value.filter(t => t.status === statusFilter.value)
})

onMounted(async () => {
  await loadTickets()
})

async function loadTickets() {
  isLoading.value = true
  error.value = ''
  try {
    const response = await adminApi.getSupportTickets({ limit: 100 })
    tickets.value = response.tickets || []
    total.value = response.total
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
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
  if (status === 'awaiting_reply') return 'destructive'
  if (status === 'open') return 'secondary'
  return 'outline'
}

async function handleLogout() {
  await admin.logout()
  router.push('/admin/login')
}

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}
</script>

<template>
  <div class="min-h-screen bg-slate-900">
    <!-- Header -->
    <header class="bg-slate-800 border-b border-slate-700 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-14">
          <div class="flex items-center gap-6">
            <h1 class="text-lg font-semibold text-white">Admin Panel</h1>
            <nav class="hidden md:flex items-center gap-1">
              <router-link
                v-for="item in navItems"
                :key="item.to"
                :to="item.to"
                :class="[
                  'px-3 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition-colors',
                  isActive(item.to)
                    ? 'bg-indigo-500/20 text-indigo-400'
                    : 'text-slate-400 hover:text-white hover:bg-slate-700'
                ]"
              >
                <component :is="item.icon" class="h-4 w-4" />
                {{ item.label }}
              </router-link>
            </nav>
          </div>
          <div class="flex items-center gap-4">
            <span class="text-sm text-slate-400 hidden sm:block">{{ admin.email }}</span>
            <Button
              variant="ghost"
              size="sm"
              class="text-slate-400 hover:text-white hover:bg-slate-700"
              @click="handleLogout"
            >
              <LogOut class="h-4 w-4 mr-2" />
              Выйти
            </Button>
          </div>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h2 class="text-2xl font-bold text-white">Обращения в поддержку</h2>
          <p class="text-sm text-slate-400 mt-1">Всего: {{ total }}</p>
        </div>
        <div class="flex items-center gap-3">
          <Select v-model="statusFilter">
            <SelectTrigger class="w-44 bg-slate-800 border-slate-600 text-slate-300">
              <SelectValue placeholder="Статус" />
            </SelectTrigger>
            <SelectContent class="bg-slate-800 border-slate-600">
              <SelectItem value="all" class="text-slate-300">Все</SelectItem>
              <SelectItem value="open" class="text-slate-300">Открытые</SelectItem>
              <SelectItem value="awaiting_reply" class="text-slate-300">Ожидают ответа</SelectItem>
              <SelectItem value="answered" class="text-slate-300">Отвеченные</SelectItem>
              <SelectItem value="closed" class="text-slate-300">Закрытые</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            class="border-slate-600 text-slate-300 hover:bg-slate-700 hover:text-white"
            :disabled="isLoading"
            @click="loadTickets"
          >
            <RefreshCcw class="h-4 w-4 mr-2" :class="{ 'animate-spin': isLoading }" />
            Обновить
          </Button>
        </div>
      </div>

      <!-- Error -->
      <ErrorBanner
        v-if="error"
        :message="error"
        @retry="loadTickets"
        class="mb-6"
      />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Empty -->
      <Card v-else-if="filteredTickets.length === 0" class="bg-slate-800 border-slate-700">
        <CardContent class="py-12 text-center">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-700 mb-4">
            <Headphones class="h-8 w-8 text-slate-400" />
          </div>
          <h3 class="text-lg font-medium text-white mb-2">Нет обращений</h3>
          <p class="text-slate-400">
            {{ statusFilter === 'all' ? 'Пользователи ещё не обращались в поддержку' : 'Нет обращений с выбранным статусом' }}
          </p>
        </CardContent>
      </Card>

      <!-- List -->
      <div v-else class="space-y-3">
        <Card
          v-for="ticket in filteredTickets"
          :key="ticket.id"
          class="bg-slate-800 border-slate-700 hover:bg-slate-750 transition-colors cursor-pointer"
          @click="router.push(`/admin/support/${ticket.id}`)"
        >
          <CardContent class="py-4">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-3 mb-1">
                  <span class="font-mono text-sm text-slate-400">#{{ ticket.ticket_number }}</span>
                  <Badge :variant="getStatusVariant(ticket.status)">
                    {{ getStatusLabel(ticket.status) }}
                  </Badge>
                </div>
                <h3 class="font-medium text-white truncate">{{ ticket.subject }}</h3>
                <div class="flex items-center gap-4 mt-2 text-sm text-slate-400">
                  <span class="flex items-center gap-1">
                    <Clock class="h-3.5 w-3.5" />
                    {{ formatDate(ticket.created_at) }}
                  </span>
                  <span v-if="ticket.updated_at !== ticket.created_at" class="flex items-center gap-1">
                    Обновлён: {{ formatDate(ticket.updated_at) }}
                  </span>
                </div>
              </div>
              <ChevronRight class="h-5 w-5 text-slate-500 shrink-0" />
            </div>
          </CardContent>
        </Card>
      </div>
    </main>
  </div>
</template>
