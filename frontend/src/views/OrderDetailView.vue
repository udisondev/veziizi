<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ordersApi } from '@/api/orders'
import { historyApi } from '@/api/history'
import { membersApi } from '@/api/members'
import type { MemberListItem } from '@/types/member'
import { useAuthStore } from '@/stores/auth'
import { useTutorialEvent } from '@/composables/useTutorialEvent'
import type { Order, OrderDocument, LeaveReviewRequest } from '@/types/order'
import { isOrderFinished, isOrderCancelled, isOrderActive } from '@/types/order'
import { orderStatusMap } from '@/constants/statusMaps'
import { formatDateTime } from '@/utils/formatters'
import EventHistory from '@/components/EventHistory.vue'
import OrderMessagesTab from '@/components/order/OrderMessagesTab.vue'
import OrderDocumentsTab from '@/components/order/OrderDocumentsTab.vue'
import OrderReviewsTab from '@/components/order/OrderReviewsTab.vue'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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
import { DetailPageHeader, StatusBadge, LoadingSpinner, ErrorBanner, TabsDropdown, type TabItem } from '@/components/shared'

// Icons
import {
  MoreVertical,
  Info,
  MessageSquare,
  FileText,
  Star,
  Clock,
  Check,
  XCircle,
  UserCog,
  Building2,
  X,
} from 'lucide-vue-next'

const route = useRoute()
const auth = useAuthStore()
const { emit: emitTutorial } = useTutorialEvent()

// State
const order = ref<Order | null>(null)
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)

// Tabs
const activeTab = ref('info')

// Отправляем событие для туториала при смене таба
watch(activeTab, (newTab) => {
  if (newTab === 'messages') emitTutorial('tab:messages')
  if (newTab === 'documents') emitTutorial('tab:documents')
  if (newTab === 'reviews') emitTutorial('tab:reviews')
})

// Component refs
const messagesTabRef = ref<InstanceType<typeof OrderMessagesTab> | null>(null)
const documentsTabRef = ref<InstanceType<typeof OrderDocumentsTab> | null>(null)

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

// Проверяет, является ли текущий пользователь ответственным за заказ со своей стороны
const isResponsible = computed(() => {
  if (!order.value || !auth.memberId) return false
  return order.value.customer_member_id === auth.memberId || order.value.carrier_member_id === auth.memberId
})

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

const canLeaveReview = computed(() => {
  if (!order.value || !isResponsible.value) return false
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

function loadOrderHistory(limit: number, offset: number) {
  const id = route.params.id as string
  return historyApi.getOrderHistory(id, { limit, offset })
}

// Actions
async function handleSendMessage(content: string) {
  if (!order.value) return

  actionLoading.value = true
  try {
    await ordersApi.sendMessage(order.value.id, { content })
    await loadData()
    messagesTabRef.value?.scrollToBottom()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка отправки'
  } finally {
    actionLoading.value = false
  }
}

async function handleFileUpload(file: File) {
  if (!order.value) return

  try {
    await ordersApi.uploadDocument(order.value.id, file)
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    documentsTabRef.value?.onUploadComplete()
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
    <DetailPageHeader back-to="/orders" back-label="К списку заказов">
      <template #actions>
        <DropdownMenu v-if="hasAnyAction && order">
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="icon">
              <MoreVertical class="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              v-if="canComplete"
              data-tutorial="complete-order-btn"
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
              data-tutorial="leave-review-btn"
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
      </template>
    </DetailPageHeader>

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
            <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
              <div class="min-w-0">
                <div class="flex items-center gap-3">
                  <h1 class="text-xl sm:text-2xl font-bold text-foreground">
                    Заказ #{{ orderNumber }}
                  </h1>
                  <StatusBadge :status="order.status" :status-map="orderStatusMap" />
                </div>
                <p class="text-muted-foreground text-sm mt-1">
                  Создан {{ formatDateTime(order.created_at) }}
                </p>
              </div>
              <router-link
                :to="`/freight-requests/${order.freight_request_id}`"
                class="text-primary hover:underline text-sm flex items-center gap-1 shrink-0"
              >
                <FileText class="h-3 w-3" />
                Перейти к заявке
              </router-link>
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
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-6 sm:divide-x">
                  <div>
                    <div class="text-sm text-muted-foreground mb-1">Заказчик</div>
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
                  </div>
                  <div class="sm:pl-6 border-t sm:border-t-0 pt-4 sm:pt-0">
                    <div class="text-sm text-muted-foreground mb-1">Перевозчик</div>
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
                  </div>
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

              </TabsContent>

              <!-- Messages Tab -->
              <TabsContent value="messages" data-tutorial="messages-tab" class="mt-0">
                <OrderMessagesTab
                  ref="messagesTabRef"
                  :order="order"
                  :action-loading="actionLoading"
                  @send="handleSendMessage"
                />
              </TabsContent>

              <!-- Documents Tab -->
              <TabsContent value="documents" data-tutorial="documents-tab" class="mt-0">
                <OrderDocumentsTab
                  ref="documentsTabRef"
                  :order="order"
                  :action-loading="actionLoading"
                  @upload="handleFileUpload"
                  @download="handleDownloadDocument"
                  @remove="handleRemoveDocument"
                />
              </TabsContent>

              <!-- Reviews Tab -->
              <TabsContent value="reviews" class="mt-0">
                <OrderReviewsTab :order="order" />
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

        <div class="space-y-3">
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
