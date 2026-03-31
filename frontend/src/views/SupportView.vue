<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useOnboardingStore } from '@/stores/onboarding'
import { usePermissions } from '@/composables/usePermissions'
import { storeToRefs } from 'pinia'
import { createTicket, getMyTickets, type TicketListItem } from '@/api/support'
import { getErrorMessage } from '@/api/errors'
import type { ScenarioType } from '@/types/tutorial'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'

// Shared Components
import { PageHeader } from '@/components/shared'

// Icons
import {
  GraduationCap,
  MessageSquare,
  Send,
  ChevronRight,
  Ticket,
  Clock,
  Package,
  Truck,
  Users,
  Play,
  CheckCircle,
  HandCoins,
} from 'lucide-vue-next'

const router = useRouter()
const onboarding = useOnboardingStore()
const { progress } = storeToRefs(onboarding)
const { canManageMembers } = usePermissions()

// Recent tickets
const recentTickets = ref<TicketListItem[]>([])
const ticketsLoading = ref(false)

// Form
const subject = ref('')
const message = ref('')
const submitting = ref(false)
const error = ref('')
const success = ref(false)

// Курсы обучения
interface CourseInfo {
  id: ScenarioType
  title: string
  description: string
  icon: typeof Package
  color: string
  duration: string
  requiresRole?: ('owner' | 'administrator')[]
}

const allCourses: CourseInfo[] = [
  {
    id: 'customer_flow',
    title: 'Создание заявки',
    description: 'Научитесь создавать заявки на перевозку',
    icon: Package,
    color: 'bg-blue-100 text-blue-600',
    duration: '~5 мин',
  },
  {
    id: 'offers_receive_flow',
    title: 'Выбор предложения',
    description: 'Как выбирать предложения перевозчиков',
    icon: Truck,
    color: 'bg-green-100 text-green-600',
    duration: '~2 мин',
  },
  {
    id: 'carrier_flow',
    title: 'Создание предложения',
    description: 'Как делать предложения на заявки',
    icon: HandCoins,
    color: 'bg-amber-100 text-amber-600',
    duration: '~3 мин',
  },
  {
    id: 'completion_flow',
    title: 'Завершение заявки',
    description: 'Как завершить перевозку и оставить отзыв',
    icon: CheckCircle,
    color: 'bg-emerald-100 text-emerald-600',
    duration: '~3 мин',
  },
  {
    id: 'admin_flow',
    title: 'Управление командой',
    description: 'Приглашение сотрудников и управление ролями',
    icon: Users,
    color: 'bg-purple-100 text-purple-600',
    duration: '~2 мин',
    requiresRole: ['owner', 'administrator'],
  },
]

const courses = computed(() =>
  allCourses.filter(course => {
    if (!course.requiresRole) return true
    return canManageMembers.value
  })
)

function isCompleted(courseId: ScenarioType): boolean {
  return progress.value.completedScenarios.includes(courseId)
}

async function startCourse(courseId: ScenarioType) {
  await onboarding.enterSandbox(courseId)
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
  } catch (e) {
    error.value = getErrorMessage(e)
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
  loadRecentTickets()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Поддержка" class="mb-6 !flex-row !items-center !justify-between !gap-4">
      <template #actions>
        <Button variant="outline" @click="router.push('/support/my-tickets')">
          <Ticket class="mr-2 h-4 w-4" />
          Мои обращения
        </Button>
      </template>
    </PageHeader>

    <!-- Заголовок Мини-курсы -->
    <div class="flex items-center gap-3 mb-6">
      <div class="hidden lg:flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
        <GraduationCap class="h-5 w-5 text-primary" />
      </div>
      <div>
        <CardTitle>Мини-курсы</CardTitle>
        <CardDescription class="mt-1">Интерактивное обучение работе с платформой</CardDescription>
      </div>
    </div>

    <div class="grid gap-10 lg:gap-5 lg:grid-cols-3">
      <!-- Карточки курсов -->
      <div class="lg:col-span-2 flex flex-col gap-3 content-start">
        <Card
          v-for="course in courses"
          :key="course.id"
          class="cursor-pointer transition-all hover:border-primary hover:shadow-md"
          @click="startCourse(course.id)"
        >
          <CardHeader class="flex-row items-start gap-4 p-4">
            <div :class="['flex h-12 w-12 shrink-0 items-center justify-center rounded-lg mb-0', course.color]">
              <component :is="course.icon" class="h-6 w-6" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex flex-wrap items-start gap-x-2 gap-y-2">
                <div class="flex-1 min-w-0">
                  <CardTitle class="text-base">{{ course.title }}</CardTitle>
                  <CardDescription class="mt-1 line-clamp-2">{{ course.description }}</CardDescription>
                </div>
                <div class="flex items-center gap-1.5 shrink-0 w-full lg:w-auto">
                  <Badge variant="outline" class="text-xs">{{ course.duration }}</Badge>
                  <Play v-if="!isCompleted(course.id)" class="h-3 w-3 text-muted-foreground" />
                  <CheckCircle v-else class="h-4 w-4 text-green-500" />
                </div>
              </div>
            </div>
          </CardHeader>
        </Card>
      </div>

      <!-- Форма + последние обращения -->
      <div class="space-y-6 lg:sticky lg:top-4 lg:self-start">
        <!-- Create ticket form -->
        <Card>
          <CardHeader>
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <MessageSquare class="h-5 w-5 text-primary" />
              </div>
              <div>
                <CardTitle>Написать нам</CardTitle>
                <CardDescription class="mt-1">Создать новое обращение</CardDescription>
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
