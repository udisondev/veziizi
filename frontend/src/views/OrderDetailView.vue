<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { ordersApi } from '@/api/orders'
import { historyApi } from '@/api/history'
import { useAuthStore } from '@/stores/auth'
import type { Order, OrderMessage, OrderDocument, LeaveReviewRequest } from '@/types/order'
import { orderStatusLabels, orderStatusColors, isOrderFinished, isOrderCancelled } from '@/types/order'
import EventHistory from '@/components/EventHistory.vue'

const route = useRoute()
const auth = useAuthStore()

// State
const order = ref<Order | null>(null)
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)

// Menu
const isMenuOpen = ref(false)

// Tabs
type TabType = 'info' | 'messages' | 'documents' | 'reviews' | 'history'
const activeTab = ref<TabType>('info')

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
  // Customer can complete if active or carrier_completed
  if (isCustomer.value && ['active', 'carrier_completed'].includes(order.value.status)) {
    return true
  }
  // Carrier can complete if active or customer_completed
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

  // Allow review after own side completed
  const hasCompletedOwnSide =
    (isCustomer.value && ['customer_completed', 'completed'].includes(order.value.status)) ||
    (isCarrier.value && ['carrier_completed', 'completed'].includes(order.value.status))

  if (!hasCompletedOwnSide) return false

  // Check if already left review
  const myReview = order.value.reviews.find(r => r.reviewer_org_id === auth.organizationId)
  return !myReview
})

const hasAnyAction = computed(() => {
  return canComplete.value || canCancel.value || canLeaveReview.value
})

const canViewHistory = computed(() => {
  return auth.role === 'owner' || auth.role === 'administrator'
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

// Menu handlers
function toggleMenu() {
  isMenuOpen.value = !isMenuOpen.value
}

function closeMenu() {
  isMenuOpen.value = false
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Element
  if (!target.closest('.menu-container')) {
    closeMenu()
  }
}

onMounted(() => {
  loadData()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
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
  return historyApi.getOrderHistory(id, limit, offset)
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

function openCancelModal() {
  closeMenu()
  showCancelModal.value = true
}

function openReviewModal() {
  closeMenu()
  showReviewModal.value = true
}

function handleCompleteClick() {
  closeMenu()
  handleComplete()
}
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-white shadow">
      <div class="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between">
        <router-link to="/orders" class="text-blue-600 hover:text-blue-800 text-sm">
          &larr; К списку заказов
        </router-link>

        <!-- Three-dot menu -->
        <div v-if="hasAnyAction && order" class="relative menu-container">
          <button
            @click.stop="toggleMenu"
            class="p-2 rounded-full hover:bg-gray-100 text-gray-500 hover:text-gray-700"
          >
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
            </svg>
          </button>

          <!-- Dropdown menu -->
          <div
            v-if="isMenuOpen"
            class="absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 z-50"
          >
            <div class="py-1">
              <button
                v-if="canComplete"
                @click="handleCompleteClick"
                :disabled="actionLoading"
                class="w-full text-left px-4 py-2 text-sm text-green-700 hover:bg-green-50 disabled:opacity-50"
              >
                Завершить
              </button>
              <button
                v-if="canCancel"
                @click="openCancelModal"
                class="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50"
              >
                Отменить
              </button>
              <button
                v-if="canLeaveReview"
                @click="openReviewModal"
                class="w-full text-left px-4 py-2 text-sm text-blue-600 hover:bg-blue-50"
              >
                Оставить отзыв
              </button>
            </div>
          </div>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-5xl mx-auto px-4 py-6">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
        <div class="text-gray-500 mt-2">Загрузка...</div>
      </div>

      <!-- Error -->
      <div v-else-if="error && !order" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
        {{ error }}
        <button @click="loadData" class="ml-4 text-red-600 underline">Повторить</button>
      </div>

      <!-- Content -->
      <div v-else-if="order" class="space-y-6">
        <!-- Error banner -->
        <div v-if="error" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg flex justify-between items-center">
          <span>{{ error }}</span>
          <button @click="error = ''" class="text-red-600 text-xl leading-none">&times;</button>
        </div>

        <!-- Order Header Card -->
        <div class="bg-white rounded-lg shadow p-4 sm:p-6">
          <div class="flex flex-col gap-3 sm:gap-4">
            <div>
              <h1 class="text-xl sm:text-2xl font-bold text-gray-900">Заказ #{{ orderNumber }}</h1>
              <p class="text-gray-600 text-sm mt-1">
                {{ counterpartyRole }}:
                <span class="font-medium">{{ counterpartyName }}</span>
              </p>
              <p class="text-gray-500 text-sm mt-1">
                Создан {{ formatDateTime(order.created_at) }}
              </p>
            </div>
            <div class="flex flex-wrap items-center gap-2 sm:gap-3">
              <span :class="[orderStatusColors[order.status], 'px-3 py-1 rounded-full text-sm font-medium']">
                {{ orderStatusLabels[order.status] }}
              </span>
              <router-link
                :to="`/freight-requests/${order.freight_request_id}`"
                class="text-blue-600 hover:text-blue-800 text-sm"
              >
                Перейти к заявке
              </router-link>
            </div>
          </div>
        </div>

        <!-- Tabs -->
        <div class="bg-white rounded-lg shadow">
          <div class="border-b border-gray-200">
            <nav class="flex -mb-px">
              <button
                v-for="tab in [
                  { key: 'info', label: 'Информация' },
                  { key: 'messages', label: 'Сообщения', count: order.messages.length },
                  { key: 'documents', label: 'Документы', count: order.documents.length },
                  { key: 'reviews', label: 'Отзывы', count: order.reviews.length },
                  ...(canViewHistory ? [{ key: 'history', label: 'История' }] : []),
                ]"
                :key="tab.key"
                @click="activeTab = tab.key as TabType"
                :class="[
                  'flex-1 sm:flex-none flex items-center justify-center gap-1.5 px-2 sm:px-4 py-3 text-sm font-medium border-b-2 -mb-px transition-colors',
                  activeTab === tab.key
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                ]"
              >
                <!-- Info icon -->
                <svg v-if="tab.key === 'info'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <!-- Messages icon -->
                <svg v-else-if="tab.key === 'messages'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
                <!-- Documents icon -->
                <svg v-else-if="tab.key === 'documents'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <!-- Reviews icon -->
                <svg v-else-if="tab.key === 'reviews'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
                <!-- History icon -->
                <svg v-else-if="tab.key === 'history'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span class="hidden sm:inline">{{ tab.label }}</span>
                <span v-if="tab.count !== undefined && tab.count > 0" class="text-xs bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded-full">
                  {{ tab.count }}
                </span>
              </button>
            </nav>
          </div>

          <div class="p-4 sm:p-6">
            <!-- Info Tab -->
            <div v-if="activeTab === 'info'" class="space-y-6">
              <!-- Participants -->
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div class="border border-gray-200 rounded-lg p-4">
                  <h3 class="text-sm font-medium text-gray-500 mb-2">Заказчик</h3>
                  <router-link
                    :to="`/members/${order.customer_member_id}`"
                    class="text-blue-600 hover:text-blue-800 font-medium"
                  >
                    {{ order.customer_member_name }}
                  </router-link>
                </div>
                <div class="border border-gray-200 rounded-lg p-4">
                  <h3 class="text-sm font-medium text-gray-500 mb-2">Перевозчик</h3>
                  <router-link
                    :to="`/members/${order.carrier_member_id}`"
                    class="text-blue-600 hover:text-blue-800 font-medium"
                  >
                    {{ order.carrier_member_name }}
                  </router-link>
                </div>
              </div>

              <!-- Order details -->
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div v-if="order.completed_at">
                  <dt class="text-sm text-gray-500">Завершён</dt>
                  <dd class="text-gray-900">{{ formatDateTime(order.completed_at) }}</dd>
                </div>
                <div v-if="order.cancelled_at">
                  <dt class="text-sm text-gray-500">Отменён</dt>
                  <dd class="text-gray-900">{{ formatDateTime(order.cancelled_at) }}</dd>
                </div>
              </div>

              <p class="text-gray-500 text-sm">
                Полная информация о маршруте и грузе доступна в
                <router-link
                  :to="`/freight-requests/${order.freight_request_id}`"
                  class="text-blue-600 hover:text-blue-800"
                >
                  заявке
                </router-link>.
              </p>
            </div>

            <!-- Messages Tab -->
            <div v-else-if="activeTab === 'messages'">
              <!-- Messages List -->
              <div
                ref="messagesContainer"
                class="h-64 sm:h-80 overflow-y-auto border border-gray-200 rounded-lg p-3 sm:p-4 mb-4 space-y-3"
              >
                <div v-if="sortedMessages.length === 0" class="text-center text-gray-500 py-8">
                  Сообщений пока нет
                </div>

                <div
                  v-for="msg in sortedMessages"
                  :key="msg.id"
                  :class="[
                    'max-w-[80%] p-3 rounded-lg',
                    isMyMessage(msg)
                      ? 'ml-auto bg-blue-100 text-blue-900'
                      : 'bg-gray-100 text-gray-900'
                  ]"
                >
                  <div class="text-xs text-gray-500 mb-1">
                    {{ getMessageSenderLabel(msg) }} &middot; {{ formatDateTime(msg.created_at) }}
                  </div>
                  <div class="whitespace-pre-wrap">{{ msg.content }}</div>
                </div>
              </div>

              <!-- Message Input -->
              <div v-if="canSendMessage" class="flex flex-col gap-2 sm:flex-row">
                <input
                  v-model="messageInput"
                  @keyup.enter="handleSendMessage"
                  type="text"
                  placeholder="Введите сообщение..."
                  class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  :disabled="actionLoading"
                />
                <button
                  @click="handleSendMessage"
                  :disabled="!messageInput.trim() || actionLoading"
                  class="w-full sm:w-auto px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
                >
                  Отправить
                </button>
              </div>
              <div v-else-if="isOrderCancelled(order.status)" class="text-sm text-gray-500">
                Отправка сообщений недоступна для отменённого заказа
              </div>
            </div>

            <!-- Documents Tab -->
            <div v-else-if="activeTab === 'documents'">
              <!-- Upload button -->
              <div v-if="canUploadDocument" class="mb-4">
                <input
                  ref="fileInput"
                  type="file"
                  @change="handleFileUpload"
                  class="hidden"
                />
                <button
                  @click="triggerFileUpload"
                  :disabled="uploadingFile"
                  class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
                >
                  {{ uploadingFile ? 'Загрузка...' : 'Загрузить документ' }}
                </button>
              </div>

              <!-- Documents List -->
              <div v-if="order.documents.length === 0" class="text-center text-gray-500 py-8">
                Документов пока нет
              </div>

              <div v-else class="space-y-3">
                <div
                  v-for="doc in order.documents"
                  :key="doc.id"
                  class="flex items-center justify-between p-4 border border-gray-200 rounded-lg"
                >
                  <div class="flex-1 min-w-0">
                    <p class="font-medium text-gray-900 truncate">{{ doc.name }}</p>
                    <p class="text-sm text-gray-500">
                      {{ formatFileSize(doc.size) }} &middot; {{ formatDateTime(doc.created_at) }}
                    </p>
                  </div>
                  <div class="flex gap-2 ml-4">
                    <button
                      @click="handleDownloadDocument(doc)"
                      class="text-sm text-blue-600 hover:text-blue-800"
                    >
                      Скачать
                    </button>
                    <button
                      v-if="isParticipant && !isOrderFinished(order.status)"
                      @click="handleRemoveDocument(doc)"
                      :disabled="actionLoading"
                      class="text-sm text-red-600 hover:text-red-800 disabled:opacity-50"
                    >
                      Удалить
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Reviews Tab -->
            <div v-else-if="activeTab === 'reviews'">
              <div v-if="order.reviews.length === 0" class="text-center text-gray-500 py-8">
                Отзывов пока нет
              </div>

              <div v-else class="space-y-4">
                <div
                  v-for="review in order.reviews"
                  :key="review.id"
                  class="border border-gray-200 rounded-lg p-4"
                >
                  <div class="flex items-center gap-2 mb-2">
                    <div class="flex">
                      <span
                        v-for="star in 5"
                        :key="star"
                        :class="star <= review.rating ? 'text-yellow-400' : 'text-gray-300'"
                      >
                        ★
                      </span>
                    </div>
                    <span class="text-sm text-gray-500">
                      {{ review.reviewer_org_id === order.customer_org_id ? 'Заказчик' : 'Перевозчик' }}
                    </span>
                  </div>
                  <p v-if="review.comment" class="text-gray-700">{{ review.comment }}</p>
                  <p v-else class="text-gray-400 italic">Без комментария</p>
                  <p class="text-xs text-gray-500 mt-2">{{ formatDateTime(review.created_at) }}</p>
                </div>
              </div>
            </div>

            <!-- History Tab -->
            <div v-else-if="activeTab === 'history' && canViewHistory">
              <EventHistory :load-fn="loadOrderHistory" />
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Cancel Modal -->
    <div v-if="showCancelModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-[1000]">
      <div class="bg-white rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Отменить заказ</h3>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-1">Причина отмены</label>
          <textarea
            v-model="cancelReason"
            rows="3"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Укажите причину (опционально)..."
          ></textarea>
        </div>

        <div class="flex gap-3">
          <button
            @click="showCancelModal = false"
            class="flex-1 py-2 px-4 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg"
          >
            Назад
          </button>
          <button
            @click="handleCancel"
            :disabled="actionLoading"
            class="flex-1 py-2 px-4 bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отмена...' : 'Отменить заказ' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Review Modal -->
    <div v-if="showReviewModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-[1000]">
      <div class="bg-white rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Оставить отзыв</h3>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">Оценка *</label>
            <div class="flex gap-2">
              <button
                v-for="star in 5"
                :key="star"
                @click="reviewForm.rating = star"
                :class="[
                  'text-3xl transition-colors',
                  star <= reviewForm.rating ? 'text-yellow-400' : 'text-gray-300 hover:text-yellow-200'
                ]"
              >
                ★
              </button>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Комментарий</label>
            <textarea
              v-model="reviewForm.comment"
              rows="3"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Ваш отзыв (опционально)..."
            ></textarea>
          </div>
        </div>

        <div class="flex gap-3 mt-6">
          <button
            @click="showReviewModal = false"
            class="flex-1 py-2 px-4 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg"
          >
            Отмена
          </button>
          <button
            @click="handleLeaveReview"
            :disabled="!reviewForm.rating || actionLoading"
            class="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отправка...' : 'Отправить' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
