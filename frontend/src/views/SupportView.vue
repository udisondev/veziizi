<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getFAQ, createTicket, getMyTickets, type FAQItem, type TicketListItem } from '@/api/support'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'

// Shared Components
import { PageHeader } from '@/components/shared'

// Icons
import { HelpCircle, MessageSquare, Send, ChevronRight, Ticket, Clock } from 'lucide-vue-next'

const router = useRouter()

// FAQ
const faq = ref<FAQItem[]>([])
const faqLoading = ref(false)

// Recent tickets
const recentTickets = ref<TicketListItem[]>([])
const ticketsLoading = ref(false)

// Form
const subject = ref('')
const message = ref('')
const submitting = ref(false)
const error = ref('')
const success = ref(false)

// Group FAQ by category
const faqByCategory = computed(() => {
  const grouped: Record<string, FAQItem[]> = {}
  for (const item of faq.value) {
    if (!grouped[item.category]) {
      grouped[item.category] = []
    }
    grouped[item.category].push(item)
  }
  return grouped
})

async function loadFAQ() {
  faqLoading.value = true
  try {
    faq.value = await getFAQ()
  } catch (e) {
    console.error('Failed to load FAQ:', e)
  } finally {
    faqLoading.value = false
  }
}

async function loadRecentTickets() {
  ticketsLoading.value = true
  try {
    recentTickets.value = await getMyTickets({ limit: 3 })
  } catch (e) {
    console.error('Failed to load recent tickets:', e)
  } finally {
    ticketsLoading.value = false
  }
}

async function submitTicket() {
  if (!subject.value.trim() || !message.value.trim()) {
    error.value = 'Заполните все поля'
    return
  }

  error.value = ''
  submitting.value = true
  try {
    const result = await createTicket({
      subject: subject.value.trim(),
      message: message.value.trim(),
    })
    success.value = true
    subject.value = ''
    message.value = ''
    // Navigate to the new ticket
    setTimeout(() => {
      router.push(`/support/tickets/${result.id}`)
    }, 1500)
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Не удалось создать обращение'
  } finally {
    submitting.value = false
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
  })
}

onMounted(() => {
  loadFAQ()
  loadRecentTickets()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Поддержка" class="mb-6">
      <template #actions>
        <Button variant="outline" @click="router.push('/support/my-tickets')">
          <Ticket class="mr-2 h-4 w-4" />
          Мои обращения
        </Button>
      </template>
    </PageHeader>

    <div class="grid gap-6 lg:grid-cols-3">
      <!-- Left column: FAQ -->
      <div class="lg:col-span-2 space-y-6">
        <Card>
          <CardHeader>
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <HelpCircle class="h-5 w-5 text-primary" />
              </div>
              <div>
                <CardTitle>Часто задаваемые вопросы</CardTitle>
                <CardDescription>Ответы на популярные вопросы</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="faqLoading" class="text-center py-8 text-muted-foreground">
              Загрузка...
            </div>

            <div v-else-if="Object.keys(faqByCategory).length === 0" class="text-center py-8 text-muted-foreground">
              FAQ пока пуст
            </div>

            <div v-else class="space-y-6">
              <div v-for="(items, category) in faqByCategory" :key="category">
                <h3 class="text-sm font-medium text-muted-foreground mb-3">{{ category }}</h3>
                <Accordion type="single" collapsible class="w-full">
                  <AccordionItem v-for="(item, idx) in items" :key="idx" :value="`${category}-${idx}`">
                    <AccordionTrigger class="text-left">
                      {{ item.question }}
                    </AccordionTrigger>
                    <AccordionContent>
                      <p class="text-muted-foreground">{{ item.answer }}</p>
                    </AccordionContent>
                  </AccordionItem>
                </Accordion>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Right column: Form + Recent tickets -->
      <div class="space-y-6">
        <!-- Create ticket form -->
        <Card>
          <CardHeader>
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <MessageSquare class="h-5 w-5 text-primary" />
              </div>
              <div>
                <CardTitle>Написать нам</CardTitle>
                <CardDescription>Создать новое обращение</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="success" class="text-center py-4">
              <div class="text-green-600 font-medium mb-2">Обращение создано!</div>
              <p class="text-sm text-muted-foreground">Перенаправляем...</p>
            </div>

            <form v-else @submit.prevent="submitTicket" class="space-y-4">
              <div class="space-y-2">
                <Label for="subject">Тема</Label>
                <Input
                  id="subject"
                  v-model="subject"
                  placeholder="Краткое описание вопроса"
                  :disabled="submitting"
                />
              </div>

              <div class="space-y-2">
                <Label for="message">Сообщение</Label>
                <Textarea
                  id="message"
                  v-model="message"
                  placeholder="Опишите ваш вопрос подробнее..."
                  :disabled="submitting"
                  rows="4"
                />
              </div>

              <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

              <Button type="submit" class="w-full" :disabled="submitting">
                <Send class="mr-2 h-4 w-4" />
                {{ submitting ? 'Отправка...' : 'Отправить' }}
              </Button>
            </form>
          </CardContent>
        </Card>

        <!-- Recent tickets -->
        <Card v-if="recentTickets.length > 0">
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="text-base">Последние обращения</CardTitle>
              <Button variant="ghost" size="sm" @click="router.push('/support/my-tickets')">
                Все
                <ChevronRight class="ml-1 h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div class="space-y-3">
              <div
                v-for="ticket in recentTickets"
                :key="ticket.id"
                class="p-3 rounded-lg border hover:bg-muted/50 cursor-pointer transition-colors"
                @click="router.push(`/support/tickets/${ticket.id}`)"
              >
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0 flex-1">
                    <p class="font-medium text-sm truncate">#{{ ticket.ticket_number }}</p>
                    <p class="text-sm text-muted-foreground truncate">{{ ticket.subject }}</p>
                  </div>
                  <Badge :variant="getStatusVariant(ticket.status)" class="shrink-0">
                    {{ getStatusLabel(ticket.status) }}
                  </Badge>
                </div>
                <div class="flex items-center gap-1 mt-2 text-xs text-muted-foreground">
                  <Clock class="h-3 w-3" />
                  {{ formatDate(ticket.updated_at) }}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>
