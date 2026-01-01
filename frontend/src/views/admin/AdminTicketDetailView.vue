<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi, type AdminSupportTicketDetail } from '@/api/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Separator } from '@/components/ui/separator'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// Shared Components
import { ErrorBanner } from '@/components/shared'

// Icons
import {
  Building2,
  LogOut,
  Star,
  AlertTriangle,
  Headphones,
  ArrowLeft,
  Send,
  XCircle,
  Clock,
  User,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const admin = useAdminStore()

const ticketId = computed(() => route.params.id as string)

const ticket = ref<AdminSupportTicketDetail | null>(null)
const isLoading = ref(true)
const error = ref('')

// Message form
const newMessage = ref('')
const sending = ref(false)
const sendError = ref('')

// Close dialog
const showCloseDialog = ref(false)
const closeResolution = ref('')
const closing = ref(false)

const messagesContainer = ref<HTMLElement | null>(null)

const navItems = [
  { to: '/admin/organizations', label: 'Организации', icon: Building2 },
  { to: '/admin/reviews', label: 'Отзывы', icon: Star },
  { to: '/admin/fraudsters', label: 'Накрутчики', icon: AlertTriangle },
  { to: '/admin/support', label: 'Поддержка', icon: Headphones },
]

onMounted(async () => {
  await loadTicket()
})

async function loadTicket() {
  isLoading.value = true
  error.value = ''
  try {
    ticket.value = await adminApi.getSupportTicket(ticketId.value)
    await nextTick()
    scrollToBottom()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

async function sendMessage() {
  if (!newMessage.value.trim()) return

  sendError.value = ''
  sending.value = true
  try {
    await adminApi.addSupportMessage(ticketId.value, newMessage.value.trim())
    newMessage.value = ''
    await loadTicket()
  } catch (e: any) {
    sendError.value = e.response?.data?.error || 'Не удалось отправить сообщение'
  } finally {
    sending.value = false
  }
}

async function closeTicket() {
  closing.value = true
  try {
    await adminApi.closeSupportTicket(ticketId.value, closeResolution.value.trim() || undefined)
    showCloseDialog.value = false
    closeResolution.value = ''
    await loadTicket()
  } catch (e) {
    console.error('Failed to close ticket:', e)
  } finally {
    closing.value = false
  }
}

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
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
    <main class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Back button + Title -->
      <div class="flex items-center justify-between gap-4 mb-6">
        <div class="flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            class="text-slate-400 hover:text-white hover:bg-slate-700"
            @click="router.push('/admin/support')"
          >
            <ArrowLeft class="h-5 w-5" />
          </Button>
          <div v-if="ticket">
            <h2 class="text-xl font-bold text-white">Тикет #{{ ticket.ticket_number }}</h2>
            <p class="text-sm text-slate-400">{{ ticket.subject }}</p>
          </div>
        </div>
        <Badge v-if="ticket" :variant="getStatusVariant(ticket.status)">
          {{ getStatusLabel(ticket.status) }}
        </Badge>
      </div>

      <!-- Error -->
      <ErrorBanner
        v-if="error"
        :message="error"
        @retry="loadTicket"
        class="mb-6"
      />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Ticket content -->
      <div v-else-if="ticket" class="space-y-4">
        <!-- Ticket info -->
        <Card class="bg-slate-800 border-slate-700">
          <CardContent class="py-4">
            <div class="flex flex-wrap items-center gap-4 text-sm text-slate-400">
              <span class="flex items-center gap-1">
                <Clock class="h-4 w-4" />
                Создан: {{ formatDate(ticket.created_at) }}
              </span>
              <span v-if="ticket.closed_at">
                Закрыт: {{ formatDate(ticket.closed_at) }}
              </span>
              <span class="font-mono text-xs text-slate-500">
                Member: {{ ticket.member_id.slice(0, 8) }}...
              </span>
              <span class="font-mono text-xs text-slate-500">
                Org: {{ ticket.org_id.slice(0, 8) }}...
              </span>
            </div>
          </CardContent>
        </Card>

        <!-- Messages -->
        <Card class="bg-slate-800 border-slate-700">
          <CardHeader>
            <CardTitle class="text-base text-white">Переписка</CardTitle>
          </CardHeader>
          <CardContent>
            <div
              ref="messagesContainer"
              class="space-y-4 max-h-96 overflow-y-auto pr-2"
            >
              <div
                v-for="msg in ticket.messages"
                :key="msg.id"
                :class="[
                  'flex gap-3',
                  msg.sender_type === 'admin' ? 'flex-row-reverse' : '',
                ]"
              >
                <!-- Avatar -->
                <div
                  :class="[
                    'flex h-8 w-8 shrink-0 items-center justify-center rounded-full',
                    msg.sender_type === 'admin' ? 'bg-indigo-500/20' : 'bg-slate-700',
                  ]"
                >
                  <component
                    :is="msg.sender_type === 'admin' ? Headphones : User"
                    :class="[
                      'h-4 w-4',
                      msg.sender_type === 'admin' ? 'text-indigo-400' : 'text-slate-400',
                    ]"
                  />
                </div>

                <!-- Message bubble -->
                <div
                  :class="[
                    'max-w-[80%] rounded-lg px-4 py-2',
                    msg.sender_type === 'admin'
                      ? 'bg-indigo-500/20'
                      : 'bg-slate-700',
                  ]"
                >
                  <div class="flex items-center gap-2 mb-1">
                    <span class="text-xs font-medium text-slate-300">
                      {{ msg.sender_type === 'admin' ? 'Поддержка' : 'Пользователь' }}
                    </span>
                    <span class="text-xs text-slate-500">
                      {{ formatDate(msg.created_at) }}
                    </span>
                  </div>
                  <p class="text-sm text-slate-200 whitespace-pre-wrap">{{ msg.content }}</p>
                </div>
              </div>
            </div>

            <Separator class="my-4 bg-slate-700" />

            <!-- Reply form or closed notice -->
            <div v-if="ticket.status === 'closed'" class="text-center py-4">
              <p class="text-slate-400">Тикет закрыт</p>
            </div>

            <div v-else class="space-y-3">
              <Textarea
                v-model="newMessage"
                placeholder="Введите ответ..."
                :disabled="sending"
                rows="3"
                class="bg-slate-700 border-slate-600 text-white resize-none"
              />
              <p v-if="sendError" class="text-sm text-red-400">{{ sendError }}</p>
              <div class="flex justify-between">
                <Button
                  variant="outline"
                  class="border-slate-600 text-slate-300 hover:bg-slate-700 hover:text-white"
                  @click="showCloseDialog = true"
                >
                  <XCircle class="mr-2 h-4 w-4" />
                  Закрыть тикет
                </Button>
                <Button
                  class="bg-indigo-600 hover:bg-indigo-500 text-white"
                  :disabled="sending || !newMessage.trim()"
                  @click="sendMessage"
                >
                  <Send class="mr-2 h-4 w-4" />
                  {{ sending ? 'Отправка...' : 'Отправить' }}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </main>

    <!-- Close Dialog -->
    <Dialog v-model:open="showCloseDialog">
      <DialogContent class="bg-slate-800 border-slate-700 text-white sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-white">Закрыть тикет</DialogTitle>
          <DialogDescription class="text-slate-400">
            Вы можете добавить комментарий к закрытию (необязательно)
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Textarea
            v-model="closeResolution"
            rows="3"
            class="bg-slate-700 border-slate-600 text-white resize-none"
            placeholder="Причина закрытия или итог..."
          />
        </div>

        <DialogFooter>
          <Button
            variant="ghost"
            class="text-slate-400 hover:text-white"
            :disabled="closing"
            @click="showCloseDialog = false"
          >
            Отмена
          </Button>
          <Button
            variant="destructive"
            :disabled="closing"
            @click="closeTicket"
          >
            {{ closing ? 'Закрытие...' : 'Закрыть тикет' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
