<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getTicket, addMessage, reopenTicket, type TicketDetail } from '@/api/support'
import { useAuthStore } from '@/stores/auth'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'

// Shared Components
import { PageHeader } from '@/components/shared'

// Icons
import { Send, ArrowLeft, RefreshCw, Clock, User, Headphones } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const ticketId = computed(() => route.params.id as string)

const ticket = ref<TicketDetail | null>(null)
const loading = ref(false)
const error = ref('')

// Message form
const newMessage = ref('')
const sending = ref(false)
const sendError = ref('')

// Reopen
const reopening = ref(false)

const messagesContainer = ref<HTMLElement | null>(null)

async function loadTicket() {
  loading.value = true
  error.value = ''
  try {
    ticket.value = await getTicket(ticketId.value)
    await nextTick()
    scrollToBottom()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Не удалось загрузить обращение'
  } finally {
    loading.value = false
  }
}

async function sendMessage() {
  if (!newMessage.value.trim()) return

  sendError.value = ''
  sending.value = true
  try {
    await addMessage(ticketId.value, { content: newMessage.value.trim() })
    newMessage.value = ''
    await loadTicket()
  } catch (e: any) {
    sendError.value = e.response?.data?.error || 'Не удалось отправить сообщение'
  } finally {
    sending.value = false
  }
}

async function handleReopen() {
  reopening.value = true
  try {
    await reopenTicket(ticketId.value)
    await loadTicket()
  } catch (e: any) {
    console.error('Failed to reopen ticket:', e)
  } finally {
    reopening.value = false
  }
}

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
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

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString('ru-RU', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function isMyMessage(message: { sender_type: string; sender_id: string }): boolean {
  return message.sender_type === 'user' && message.sender_id === auth.memberId
}

onMounted(() => {
  loadTicket()
})
</script>

<template>
  <div class="max-w-4xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader class="mb-6">
      <template #default>
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" @click="router.push('/support/my-tickets')">
            <ArrowLeft class="h-5 w-5" />
          </Button>
          <div v-if="ticket">
            <h1 class="text-xl font-semibold">Обращение #{{ ticket.ticket_number }}</h1>
            <p class="text-sm text-muted-foreground">{{ ticket.subject }}</p>
          </div>
          <div v-else>
            <h1 class="text-xl font-semibold">Обращение</h1>
          </div>
        </div>
      </template>
      <template #actions>
        <Badge v-if="ticket" :variant="getStatusVariant(ticket.status)">
          {{ getStatusLabel(ticket.status) }}
        </Badge>
      </template>
    </PageHeader>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-muted-foreground">
      Загрузка...
    </div>

    <!-- Error -->
    <Card v-else-if="error">
      <CardContent class="py-12 text-center">
        <p class="text-destructive mb-4">{{ error }}</p>
        <Button variant="outline" @click="router.push('/support/my-tickets')">
          <ArrowLeft class="mr-2 h-4 w-4" />
          К списку обращений
        </Button>
      </CardContent>
    </Card>

    <!-- Ticket content -->
    <div v-else-if="ticket" class="space-y-4">
      <!-- Ticket info -->
      <Card>
        <CardContent class="py-4">
          <div class="flex items-center gap-4 text-sm text-muted-foreground">
            <span class="flex items-center gap-1">
              <Clock class="h-4 w-4" />
              Создан: {{ formatDateTime(ticket.created_at) }}
            </span>
            <span v-if="ticket.closed_at">
              Закрыт: {{ formatDateTime(ticket.closed_at) }}
            </span>
          </div>
        </CardContent>
      </Card>

      <!-- Messages -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Переписка</CardTitle>
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
                isMyMessage(msg) ? 'flex-row-reverse' : '',
              ]"
            >
              <!-- Avatar -->
              <div
                :class="[
                  'flex h-8 w-8 shrink-0 items-center justify-center rounded-full',
                  msg.sender_type === 'admin' ? 'bg-primary/10' : 'bg-muted',
                ]"
              >
                <component
                  :is="msg.sender_type === 'admin' ? Headphones : User"
                  :class="[
                    'h-4 w-4',
                    msg.sender_type === 'admin' ? 'text-primary' : 'text-muted-foreground',
                  ]"
                />
              </div>

              <!-- Message bubble -->
              <div
                :class="[
                  'max-w-[80%] rounded-lg px-4 py-2',
                  msg.sender_type === 'admin'
                    ? 'bg-primary/10'
                    : 'bg-muted',
                ]"
              >
                <div class="flex items-center gap-2 mb-1">
                  <span class="text-xs font-medium">
                    {{ msg.sender_type === 'admin' ? 'Поддержка' : 'Вы' }}
                  </span>
                  <span class="text-xs text-muted-foreground">
                    {{ formatDateTime(msg.created_at) }}
                  </span>
                </div>
                <p class="text-sm whitespace-pre-wrap">{{ msg.content }}</p>
              </div>
            </div>
          </div>

          <Separator class="my-4" />

          <!-- Reply form or closed notice -->
          <div v-if="ticket.status === 'closed'" class="text-center py-4">
            <p class="text-muted-foreground mb-3">Обращение закрыто</p>
            <Button variant="outline" @click="handleReopen" :disabled="reopening">
              <RefreshCw :class="['mr-2 h-4 w-4', reopening && 'animate-spin']" />
              {{ reopening ? 'Открытие...' : 'Открыть заново' }}
            </Button>
          </div>

          <form v-else @submit.prevent="sendMessage" class="space-y-3">
            <Textarea
              v-model="newMessage"
              placeholder="Введите сообщение..."
              :disabled="sending"
              rows="3"
            />
            <p v-if="sendError" class="text-sm text-destructive">{{ sendError }}</p>
            <div class="flex justify-end">
              <Button type="submit" :disabled="sending || !newMessage.trim()">
                <Send class="mr-2 h-4 w-4" />
                {{ sending ? 'Отправка...' : 'Отправить' }}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
