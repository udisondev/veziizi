<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { ordersApi } from '@/api/orders'
import { historyApi } from '@/api/history'
import { membersApi } from '@/api/members'
import type { MemberListItem } from '@/types/member'
import { useAuthStore } from '@/stores/auth'
import type { Order, OrderMessage, OrderDocument, LeaveReviewRequest } from '@/types/order'
import { isOrderFinished, isOrderCancelled, isOrderActive } from '@/types/order'
import EventHistory from '@/components/EventHistory.vue'

// UI Components
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent } from '@/components/ui/tabs'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

// Shared Components
import { BackLink, StatusBadge, LoadingSpinner, ErrorBanner, TabsDropdown, type TabItem } from '@/components/shared'

// Icons
import {
  MoreVertical,
  Info,
  MessageSquare,
  FileText,
  Star,
  Clock,
  Send,
  Upload,
  Download,
  Trash2,
  Check,
  XCircle,
  UserCog,
  Building2,
  X,
} from 'lucide-vue-next'

const route = useRoute()
const auth = useAuthStore()

// State
const order = ref<Order | null>(null)
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)

// Tabs
const activeTab = ref('info')

// Messages
const messageInput = ref('')
const messagesContainer = ref<HTMLDivElement | null>(null)

// Documents
const fileInput = ref<HTMLInputElement | null>(null)
const uploadingFile = ref(false)

// Modals
const showCancelModal = ref(false)
const cancelReason = ref('')

const showReviewModal = ref(false)
const reviewForm = ref<LeaveReviewRequest>({
  rating: 5,
  comment: '',
})

// Reassign modal
const showReassignModal = ref(false)
const selectedNewMember = ref('')
const availableMembers = ref<{ id: string; name: string }[]>([])

// Status map for StatusBadge
const orderStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  active: { label: 'Активен', variant: 'info' },
  customer_completed: { label: 'Завершён заказчиком', variant: 'warning' },
  carrier_completed: { label: 'Завершён перевозчиком', variant: 'warning' },
  completed: { label: 'Завершён', variant: 'success' },
  cancelled: { label: 'Отменён', variant: 'destructive' },
}

// Computed
const isCustomer = computed(() => {
  if (!order.value) return false
  return order.value.customer_org_id === auth.organizationId
})

const isCarrier = computed(() => {
  if (!order.value) return false
  return order.value.carrier_org_id === auth.organizationId
})

const isParticipant = computed(() => isCustomer.value || isCarrier.value)

const canComplete = computed(() => {
  if (!order.value || !isParticipant.value) return false
  if (isCustomer.value && ['active', 'carrier_completed'].includes(order.value.status)) {
    return true
  }
  if (isCarrier.value && ['active', 'customer_completed'].includes(order.value.status)) {
    return true
  }
  return false
})

const canCancel = computed(() => {
  if (!order.value || !isParticipant.value) return false
  return order.value.status === 'active'
})

const canSendMessage = computed(() => {
  if (!order.value || !isParticipant.value) return false
  return !isOrderCancelled(order.value.status)
})

const canUploadDocument = computed(() => {
  if (!order.value || !isParticipant.value) return false
  return !isOrderFinished(order.value.status)
})

const canLeaveReview = computed(() => {
  if (!order.value || !isParticipant.value) return false
  if (isOrderCancelled(order.value.status)) return false

  const hasCompletedOwnSide =
    (isCustomer.value && ['customer_completed', 'completed'].includes(order.value.status)) ||
    (isCarrier.value && ['carrier_completed', 'completed'].includes(order.value.status))

  if (!hasCompletedOwnSide) return false

  const myReview = order.value.reviews.find(r => r.reviewer_org_id === auth.organizationId)
  return !myReview
})

const canReassign = computed(() => {
  if (!order.value || !isParticipant.value) return false
  if (auth.role !== 'owner' && auth.role !== 'administrator') return false
  return isOrderActive(order.value.status)
})

const hasAnyAction = computed(() => {
  return canComplete.value || canCancel.value || canLeaveReview.value || canReassign.value
})

const canViewHistory = computed(() => {
  return auth.role === 'owner' || auth.role === 'administrator'
})

const tabItems = computed((): TabItem[] => {
  const items: TabItem[] = [
    { value: 'info', label: 'Информация', icon: Info },
    { value: 'messages', label: 'Сообщения', icon: MessageSquare, badge: order.value?.messages.length || undefined },
    { value: 'documents', label: 'Документы', icon: FileText, badge: order.value?.documents.length || undefined },
    { value: 'reviews', label: 'Отзывы', icon: Star, badge: order.value?.reviews.length || undefined },
  ]
  if (canViewHistory.value) {
    items.push({ value: 'history', label: 'История', icon: Clock, separator: true })
  }
  return items
})

const orderNumber = computed(() => {
  if (!order.value) return 0
  return order.value.order_number
})

const counterpartyName = computed(() => {
  if (!order.value) return ''
  return isCustomer.value ? order.value.carrier_org_name : order.value.customer_org_name
})

const counterpartyRole = computed(() => {
  return isCustomer.value ? 'Перевозчик' : 'Заказчик'
})

const sortedMessages = computed(() => {
  if (!order.value) return []
  return [...order.value.messages].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  )
})

// Methods
async function loadData() {
  isLoading.value = true
  error.value = ''
  try {
    const id = route.params.id as string
    order.value = await ordersApi.get(id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function loadOrderHistory(limit: number, offset: number) {
  const id = route.params.id as string
  return historyApi.getOrderHistory(id, { limit, offset })
}

function isMyMessage(msg: OrderMessage): boolean {
  return msg.sender_org_id === auth.organizationId
}

function getMessageSenderLabel(msg: OrderMessage): string {
  if (!order.value) return ''
  if (msg.sender_org_id === order.value.customer_org_id) return 'Заказчик'
  if (msg.sender_org_id === order.value.carrier_org_id) return 'Перевозчик'
  return ''
}

async function scrollToBottom() {
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

// Actions
async function handleSendMessage() {
  if (!order.value || !messageInput.value.trim()) return

  actionLoading.value = true
  try {
    await ordersApi.sendMessage(order.value.id, { content: messageInput.value.trim() })
    messageInput.value = ''
    await loadData()
    scrollToBottom()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка отправки'
  } finally {
    actionLoading.value = false
  }
}

function triggerFileUpload() {
  fileInput.value?.click()
}

async function handleFileUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file || !order.value) return

  uploadingFile.value = true
  try {
    await ordersApi.uploadDocument(order.value.id, file)
    await loadData()
    target.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    uploadingFile.value = false
  }
}

async function handleDownloadDocument(doc: OrderDocument) {
  if (!order.value) return

  try {
    const { url } = await ordersApi.downloadDocument(order.value.id, doc.id)
    const a = document.createElement('a')
    a.href = url
    a.download = doc.name
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка скачивания'
  }
}

async function handleRemoveDocument(doc: OrderDocument) {
  if (!order.value || !confirm(`Удалить документ "${doc.name}"?`)) return

  actionLoading.value = true
  try {
    await ordersApi.removeDocument(order.value.id, doc.id)
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка удаления'
  } finally {
    actionLoading.value = false
  }
}

async function handleComplete() {
  if (!order.value) return
  actionLoading.value = true
  try {
    await ordersApi.complete(order.value.id)
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

async function handleCancel() {
  if (!order.value) return
  actionLoading.value = true
  try {
    await ordersApi.cancel(order.value.id, cancelReason.value || undefined)
    showCancelModal.value = false
    cancelReason.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

async function handleLeaveReview() {
  if (!order.value) return
  if (!reviewForm.value.rating) {
    error.value = 'Укажите оценку'
    return
  }
  actionLoading.value = true
  try {
    await ordersApi.leaveReview(order.value.id, {
      rating: reviewForm.value.rating,
      comment: reviewForm.value.comment || undefined,
    })
    showReviewModal.value = false
    reviewForm.value = { rating: 5, comment: '' }
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

async function openReassignModal() {
  if (!auth.organizationId) return

  try {
    const members = await membersApi.listByOrganization(auth.organizationId)
    availableMembers.value = members
      .filter((m: MemberListItem) => m.status === 'active')
      .map((m: MemberListItem) => ({ id: m.id, name: m.name }))
    showReassignModal.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки сотрудников'
  }
}

async function handleReassign() {
  if (!order.value || !selectedNewMember.value) return
  actionLoading.value = true
  try {
    await ordersApi.reassign(order.value.id, selectedNewMember.value)
    showReassignModal.value = false
    selectedNewMember.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка переназначения'
  } finally {
    actionLoading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <header class="bg-card border-b">
      <div class="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between">
        <BackLink to="/orders" label="К списку заказов" />

        <!-- Actions dropdown -->
        <DropdownMenu v-if="hasAnyAction && order">
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="icon">
              <MoreVertical class="h-5 w-5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              v-if="canComplete"
              :disabled="actionLoading"
              class="text-success focus:text-success"
              @click="handleComplete"
            >
              <Check class="mr-2 h-4 w-4" />
              Завершить
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="canCancel"
              class="text-destructive focus:text-destructive"
              @click="showCancelModal = true"
            >
              <XCircle class="mr-2 h-4 w-4" />
              Отменить
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="canLeaveReview"
              @click="showReviewModal = true"
            >
              <Star class="mr-2 h-4 w-4" />
              Оставить отзыв
            </DropdownMenuItem>
            <DropdownMenuSeparator v-if="canReassign && (canComplete || canCancel || canLeaveReview)" />
            <DropdownMenuItem
              v-if="canReassign"
              @click="openReassignModal"
            >
              <UserCog class="mr-2 h-4 w-4" />
              Переназначить
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-5xl mx-auto px-4 py-6">
      <!-- Loading -->
      <LoadingSpinner v-if="isLoading" text="Загрузка заказа..." />

      <!-- Error -->
      <ErrorBanner
        v-else-if="error && !order"
        :message="error"
        @retry="loadData"
      />

      <!-- Content -->
      <div v-else-if="order" class="space-y-6">
        <!-- Error banner -->
        <Card v-if="error" class="border-destructive/50 bg-destructive/5">
          <CardContent class="flex items-center justify-between py-3">
            <span class="text-sm text-destructive">{{ error }}</span>
            <Button variant="ghost" size="sm" @click="error = ''">
              <X class="h-4 w-4" />
            </Button>
          </CardContent>
        </Card>

        <!-- Order Header Card -->
        <Card>
          <CardContent class="p-4 sm:p-6">
            <div class="flex flex-col gap-4">
              <div>
                <h1 class="text-xl sm:text-2xl font-bold text-foreground">
                  Заказ #{{ orderNumber }}
                </h1>
                <p class="text-muted-foreground text-sm mt-1 break-words">
                  {{ counterpartyRole }}:
                  <span class="font-medium text-foreground">{{ counterpartyName }}</span>
                </p>
                <p class="text-muted-foreground text-sm mt-1">
                  Создан {{ formatDateTime(order.created_at) }}
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <StatusBadge :status="order.status" :status-map="orderStatusMap" />
                <router-link
                  :to="`/freight-requests/${order.freight_request_id}`"
                >
                  <Badge variant="outline" class="cursor-pointer">
                    <FileText class="mr-1 h-3 w-3" />
                    Перейти к заявке
                  </Badge>
                </router-link>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Tabs -->
        <Card>
          <Tabs v-model="activeTab" class="w-full">
            <!-- Tab selector dropdown -->
            <div class="border-b p-3">
              <TabsDropdown v-model="activeTab" :items="tabItems" />
            </div>

            <div class="p-4 sm:p-6">
              <!-- Info Tab -->
              <TabsContent value="info" class="mt-0 space-y-6">
                <!-- Participants -->
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
                  <Card>
                    <CardHeader class="pb-2">
                      <CardTitle class="text-sm text-muted-foreground">Заказчик</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <router-link
                        :to="{ name: 'organization-profile', params: { id: order.customer_org_id } }"
                        class="text-primary hover:underline font-medium flex items-center gap-1"
                      >
                        <Building2 class="h-4 w-4" />
                        {{ order.customer_org_name }}
                      </router-link>
                      <div class="text-xs text-muted-foreground mt-2">Контакт:</div>
                      <router-link
                        :to="`/members/${order.customer_member_id}`"
                        class="text-primary hover:underline text-sm"
                      >
                        {{ order.customer_member_name }}
                      </router-link>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader class="pb-2">
                      <CardTitle class="text-sm text-muted-foreground">Перевозчик</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <router-link
                        :to="{ name: 'organization-profile', params: { id: order.carrier_org_id } }"
                        class="text-primary hover:underline font-medium flex items-center gap-1"
                      >
                        <Building2 class="h-4 w-4" />
                        {{ order.carrier_org_name }}
                      </router-link>
                      <div class="text-xs text-muted-foreground mt-2">Контакт:</div>
                      <router-link
                        :to="`/members/${order.carrier_member_id}`"
                        class="text-primary hover:underline text-sm"
                      >
                        {{ order.carrier_member_name }}
                      </router-link>
                    </CardContent>
                  </Card>
                </div>

                <!-- Order details -->
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div v-if="order.completed_at">
                    <dt class="text-sm text-muted-foreground">Завершён</dt>
                    <dd class="text-foreground">{{ formatDateTime(order.completed_at) }}</dd>
                  </div>
                  <div v-if="order.cancelled_at">
                    <dt class="text-sm text-muted-foreground">Отменён</dt>
                    <dd class="text-foreground">{{ formatDateTime(order.cancelled_at) }}</dd>
                  </div>
                </div>

                <p class="text-muted-foreground text-sm">
                  Полная информация о маршруте и грузе доступна в
                  <router-link
                    :to="`/freight-requests/${order.freight_request_id}`"
                    class="text-primary hover:underline"
                  >
                    заявке
                  </router-link>.
                </p>
              </TabsContent>

              <!-- Messages Tab -->
              <TabsContent value="messages" class="mt-0">
                <!-- Messages List -->
                <div
                  ref="messagesContainer"
                  class="h-64 sm:h-80 overflow-y-auto border rounded-lg p-3 sm:p-4 mb-4 space-y-3"
                >
                  <div v-if="sortedMessages.length === 0" class="text-center text-muted-foreground py-8">
                    Сообщений пока нет
                  </div>

                  <div
                    v-for="msg in sortedMessages"
                    :key="msg.id"
                    :class="[
                      'max-w-[80%] p-3 rounded-lg',
                      isMyMessage(msg)
                        ? 'ml-auto bg-primary/10 text-foreground'
                        : 'bg-muted text-foreground'
                    ]"
                  >
                    <div class="text-xs text-muted-foreground mb-1">
                      {{ getMessageSenderLabel(msg) }} &middot; {{ formatDateTime(msg.created_at) }}
                    </div>
                    <div class="whitespace-pre-wrap break-words">{{ msg.content }}</div>
                  </div>
                </div>

                <!-- Message Input -->
                <div v-if="canSendMessage" class="flex flex-col gap-2 sm:flex-row">
                  <Input
                    v-model="messageInput"
                    @keyup.enter="handleSendMessage"
                    placeholder="Введите сообщение..."
                    :disabled="actionLoading"
                    class="flex-1"
                  />
                  <Button
                    :disabled="!messageInput.trim() || actionLoading"
                    @click="handleSendMessage"
                  >
                    <Send class="mr-2 h-4 w-4" />
                    Отправить
                  </Button>
                </div>
                <div v-else-if="isOrderCancelled(order.status)" class="text-sm text-muted-foreground">
                  Отправка сообщений недоступна для отменённого заказа
                </div>
              </TabsContent>

              <!-- Documents Tab -->
              <TabsContent value="documents" class="mt-0">
                <!-- Upload button -->
                <div v-if="canUploadDocument" class="mb-4">
                  <input
                    ref="fileInput"
                    type="file"
                    @change="handleFileUpload"
                    class="hidden"
                  />
                  <Button
                    :disabled="uploadingFile"
                    @click="triggerFileUpload"
                  >
                    <Upload class="mr-2 h-4 w-4" />
                    {{ uploadingFile ? 'Загрузка...' : 'Загрузить документ' }}
                  </Button>
                </div>

                <!-- Documents List -->
                <div v-if="order.documents.length === 0" class="text-center text-muted-foreground py-8">
                  Документов пока нет
                </div>

                <div v-else class="space-y-3">
                  <Card
                    v-for="doc in order.documents"
                    :key="doc.id"
                  >
                    <CardContent class="flex items-center justify-between p-4">
                      <div class="flex-1 min-w-0">
                        <p class="font-medium text-foreground truncate">{{ doc.name }}</p>
                        <p class="text-sm text-muted-foreground">
                          {{ formatFileSize(doc.size) }} &middot; {{ formatDateTime(doc.created_at) }}
                        </p>
                      </div>
                      <div class="flex gap-2 ml-4">
                        <Button
                          variant="ghost"
                          size="sm"
                          @click="handleDownloadDocument(doc)"
                        >
                          <Download class="h-4 w-4" />
                        </Button>
                        <Button
                          v-if="isParticipant && !isOrderFinished(order.status)"
                          variant="ghost"
                          size="sm"
                          class="text-destructive hover:text-destructive"
                          :disabled="actionLoading"
                          @click="handleRemoveDocument(doc)"
                        >
                          <Trash2 class="h-4 w-4" />
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <!-- Reviews Tab -->
              <TabsContent value="reviews" class="mt-0">
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
              </TabsContent>

              <!-- History Tab -->
              <TabsContent v-if="canViewHistory" value="history" class="mt-0">
                <EventHistory :load-fn="loadOrderHistory" />
              </TabsContent>
            </div>
          </Tabs>
        </Card>
      </div>
    </main>

    <!-- Cancel Dialog -->
    <Dialog v-model:open="showCancelModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отменить заказ</DialogTitle>
          <DialogDescription>
            Это действие нельзя отменить
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отмены</Label>
          <Textarea
            v-model="cancelReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showCancelModal = false">
            Назад
          </Button>
          <Button
            variant="destructive"
            :disabled="actionLoading"
            @click="handleCancel"
          >
            {{ actionLoading ? 'Отмена...' : 'Отменить заказ' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Review Dialog -->
    <Dialog v-model:open="showReviewModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Оставить отзыв</DialogTitle>
          <DialogDescription>
            Оцените работу контрагента
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-4">
          <div class="space-y-2">
            <Label>Оценка *</Label>
            <div class="flex gap-2">
              <button
                v-for="star in 5"
                :key="star"
                type="button"
                @click="reviewForm.rating = star"
                :class="[
                  'text-3xl transition-colors',
                  star <= reviewForm.rating ? 'text-warning' : 'text-muted-foreground hover:text-warning/50'
                ]"
              >
                ★
              </button>
            </div>
          </div>

          <div class="space-y-2">
            <Label>Комментарий</Label>
            <Textarea
              v-model="reviewForm.comment"
              rows="3"
              placeholder="Ваш отзыв (опционально)..."
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showReviewModal = false">
            Отмена
          </Button>
          <Button
            :disabled="!reviewForm.rating || actionLoading"
            @click="handleLeaveReview"
          >
            {{ actionLoading ? 'Отправка...' : 'Отправить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Reassign Dialog -->
    <Dialog v-model:open="showReassignModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Переназначить ответственного</DialogTitle>
          <DialogDescription>
            Выберите нового ответственного сотрудника
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Новый ответственный</Label>
          <Select v-model="selectedNewMember">
            <SelectTrigger>
              <SelectValue placeholder="Выберите сотрудника" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="member in availableMembers"
                :key="member.id"
                :value="member.id"
              >
                {{ member.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showReassignModal = false">
            Отмена
          </Button>
          <Button
            :disabled="!selectedNewMember || actionLoading"
            @click="handleReassign"
          >
            {{ actionLoading ? 'Сохранение...' : 'Сохранить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
