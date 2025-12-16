<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import { ordersApi } from '@/api/orders'
import type { Order, OrderMessage, OrderDocument } from '@/types/order'
import {
  orderStatusLabels,
  orderStatusColors,
  isOrderFinished,
  isOrderCancelled,
  isOrderActive,
} from '@/types/order'

const route = useRoute()
const auth = useAuthStore()
const permissions = usePermissions()

// State
const order = ref<Order | null>(null)
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)

// Tabs
type TabType = 'info' | 'messages' | 'documents' | 'reviews'
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
const reviewRating = ref(5)
const reviewComment = ref('')

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

const myRole = computed(() => {
  if (isCustomer.value) return 'Заказчик'
  if (isCarrier.value) return 'Перевозчик'
  return ''
})

const canComplete = computed(() => {
  if (!order.value || !isParticipant.value) return false
  if (!isOrderActive(order.value.status)) return false

  // Check if current participant already completed
  if (isCustomer.value && ['customer_completed', 'completed'].includes(order.value.status)) {
    return false
  }
  if (isCarrier.value && ['carrier_completed', 'completed'].includes(order.value.status)) {
    return false
  }

  return permissions.canCompleteOrder(order.value.customer_org_id, order.value.carrier_org_id)
})

const canCancel = computed(() => {
  if (!order.value || !isParticipant.value) return false
  // Can only cancel active order (not partially completed)
  return order.value.status === 'active' &&
    permissions.canCancelOrder(order.value.customer_org_id, order.value.carrier_org_id)
})

const canSendMessage = computed(() => {
  if (!order.value || !isParticipant.value) return false
  return !isOrderCancelled(order.value.status) &&
    permissions.canAddOrderMessage(order.value.customer_org_id, order.value.carrier_org_id)
})

const canUploadDocument = computed(() => {
  if (!order.value || !isParticipant.value) return false
  return !isOrderFinished(order.value.status) &&
    permissions.canUploadOrderDocument(order.value.customer_org_id, order.value.carrier_org_id)
})

const canLeaveReview = computed(() => {
  if (!order.value || !isParticipant.value) return false
  if (isOrderCancelled(order.value.status)) return false

  // Allow review after own side completed (not waiting for both sides)
  const hasCompletedOwnSide =
    (isCustomer.value && ['customer_completed', 'completed'].includes(order.value.status)) ||
    (isCarrier.value && ['carrier_completed', 'completed'].includes(order.value.status))

  if (!hasCompletedOwnSide) return false

  // Check if already left review
  const myReview = order.value.reviews.find(r => r.reviewer_org_id === auth.organizationId)
  return !myReview && permissions.canLeaveOrderReview(order.value.customer_org_id, order.value.carrier_org_id)
})

const sortedMessages = computed(() => {
  if (!order.value) return []
  return [...order.value.messages].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  )
})

const shortId = computed(() => {
  if (!order.value) return ''
  return order.value.id.slice(0, 8)
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

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString('ru-RU', {
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
    // Reset input
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

    // Create download link
    const a = document.createElement('a')
    a.href = url
    a.download = doc.name
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)

    // Cleanup blob URL
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

  actionLoading.value = true
  try {
    await ordersApi.leaveReview(order.value.id, {
      rating: reviewRating.value,
      comment: reviewComment.value || undefined,
    })
    showReviewModal.value = false
    reviewRating.value = 5
    reviewComment.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-white shadow">
      <div class="max-w-5xl mx-auto px-4 py-4">
        <router-link to="/orders" class="text-blue-600 hover:text-blue-800 text-sm">
          &larr; К списку заказов
        </router-link>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-5xl mx-auto px-4 py-6">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
        <div class="text-gray-500 mt-2">Загрузка...</div>
      </div>

      <!-- Error (full page) -->
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

        <!-- Header Card -->
        <div class="bg-white rounded-lg shadow p-6">
          <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <h1 class="text-2xl font-bold text-gray-900">Заказ #{{ shortId }}</h1>
              <p class="text-gray-500 text-sm mt-1">
                Создан {{ formatDateTime(order.created_at) }}
                <span v-if="myRole" class="ml-2 text-blue-600">({{ myRole }})</span>
              </p>
            </div>
            <div class="flex items-center gap-3 flex-wrap">
              <span :class="[orderStatusColors[order.status], 'px-3 py-1 rounded-full text-sm font-medium']">
                {{ orderStatusLabels[order.status] }}
              </span>

              <button
                v-if="canComplete"
                @click="handleComplete"
                :disabled="actionLoading"
                class="px-4 py-2 bg-green-600 hover:bg-green-700 text-white text-sm font-medium rounded-lg disabled:opacity-50"
              >
                Завершить
              </button>

              <button
                v-if="canCancel"
                @click="showCancelModal = true"
                class="px-4 py-2 text-red-600 hover:bg-red-50 text-sm font-medium rounded-lg border border-red-200"
              >
                Отменить
              </button>

              <button
                v-if="canLeaveReview"
                @click="showReviewModal = true"
                class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg"
              >
                Оставить отзыв
              </button>
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
                ]"
                :key="tab.key"
                @click="activeTab = tab.key as TabType"
                :class="[
                  'px-4 py-3 text-sm font-medium border-b-2 -mb-px transition-colors',
                  activeTab === tab.key
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                ]"
              >
                {{ tab.label }}
                <span v-if="tab.count !== undefined" class="ml-1 text-xs bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded-full">
                  {{ tab.count }}
                </span>
              </button>
            </nav>
          </div>

          <div class="p-6">
            <!-- Info Tab -->
            <div v-if="activeTab === 'info'" class="space-y-6">
              <!-- Counterparty info -->
              <div class="border border-gray-200 rounded-lg p-4">
                <h3 class="text-sm font-medium text-gray-900 mb-3">
                  {{ isCarrier ? 'Заказчик' : 'Перевозчик' }}
                </h3>
                <dl class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <dt class="text-xs text-gray-500">Организация</dt>
                    <dd>
                      <router-link
                        :to="{ name: 'organization-profile', params: { id: isCarrier ? order.customer_org_id : order.carrier_org_id } }"
                        class="text-blue-600 hover:text-blue-800 hover:underline"
                      >
                        {{ isCarrier ? order.customer_org_name : order.carrier_org_name }}
                      </router-link>
                    </dd>
                  </div>
                  <div>
                    <dt class="text-xs text-gray-500">Сотрудник</dt>
                    <dd>
                      <router-link
                        :to="{ name: 'member-profile', params: { id: isCarrier ? order.customer_member_id : order.carrier_member_id } }"
                        class="text-blue-600 hover:text-blue-800 hover:underline"
                      >
                        {{ isCarrier ? order.customer_member_name : order.carrier_member_name }}
                      </router-link>
                    </dd>
                  </div>
                </dl>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <dt class="text-sm text-gray-500">ID заявки</dt>
                  <dd>
                    <router-link
                      :to="`/freight-requests/${order.freight_request_id}`"
                      class="text-blue-600 hover:text-blue-800"
                    >
                      {{ order.freight_request_id.slice(0, 8) }}...
                    </router-link>
                  </dd>
                </div>
                <div v-if="order.completed_at">
                  <dt class="text-sm text-gray-500">Завершён</dt>
                  <dd>{{ formatDateTime(order.completed_at) }}</dd>
                </div>
                <div v-if="order.cancelled_at">
                  <dt class="text-sm text-gray-500">Отменён</dt>
                  <dd>{{ formatDateTime(order.cancelled_at) }}</dd>
                </div>
              </div>

              <p class="text-gray-500 text-sm">
                Полная информация о маршруте и грузе доступна в заявке.
              </p>
            </div>

            <!-- Messages Tab -->
            <div v-else-if="activeTab === 'messages'">
              <!-- Messages List -->
              <div
                ref="messagesContainer"
                class="h-80 overflow-y-auto border border-gray-200 rounded-lg p-4 mb-4 space-y-3"
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
              <div v-if="canSendMessage" class="flex gap-2">
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
                  class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
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

              <!-- Documents Grid -->
              <div v-if="order.documents.length === 0" class="text-center text-gray-500 py-8">
                Документов пока нет
              </div>

              <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                <div
                  v-for="doc in order.documents"
                  :key="doc.id"
                  class="border border-gray-200 rounded-lg p-4"
                >
                  <div class="flex items-start justify-between">
                    <div class="flex-1 min-w-0">
                      <div class="font-medium text-gray-900 truncate" :title="doc.name">
                        {{ doc.name }}
                      </div>
                      <div class="text-sm text-gray-500">
                        {{ formatFileSize(doc.size) }}
                      </div>
                      <div class="text-xs text-gray-400 mt-1">
                        {{ formatDateTime(doc.created_at) }}
                      </div>
                    </div>
                  </div>

                  <div class="flex gap-2 mt-3">
                    <button
                      @click="handleDownloadDocument(doc)"
                      class="text-sm text-blue-600 hover:text-blue-800"
                    >
                      Скачать
                    </button>
                    <button
                      v-if="permissions.canRemoveOrderDocument(order.customer_org_id, order.carrier_org_id)"
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
                    <!-- Stars -->
                    <div class="flex">
                      <span
                        v-for="star in 5"
                        :key="star"
                        :class="star <= review.rating ? 'text-yellow-400' : 'text-gray-300'"
                      >
                        &#9733;
                      </span>
                    </div>
                    <span class="text-sm text-gray-500">
                      {{ review.reviewer_org_id === order.customer_org_id ? 'Заказчик' : 'Перевозчик' }}
                    </span>
                  </div>
                  <p v-if="review.comment" class="text-gray-700">{{ review.comment }}</p>
                  <p v-else class="text-gray-400 italic">Без комментария</p>
                  <div class="text-xs text-gray-400 mt-2">
                    {{ formatDateTime(review.created_at) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Cancel Modal -->
    <div v-if="showCancelModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-[1000]">
      <div class="bg-white rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Отменить заказ</h3>

        <p class="text-gray-600 mb-4">
          Вы уверены, что хотите отменить заказ? Это действие нельзя отменить.
        </p>

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
            <label class="block text-sm font-medium text-gray-700 mb-2">Оценка</label>
            <div class="flex gap-2">
              <button
                v-for="star in 5"
                :key="star"
                @click="reviewRating = star"
                :class="[
                  'text-3xl transition-colors',
                  star <= reviewRating ? 'text-yellow-400' : 'text-gray-300 hover:text-yellow-200'
                ]"
              >
                &#9733;
              </button>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Комментарий</label>
            <textarea
              v-model="reviewComment"
              rows="3"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Опишите ваш опыт работы..."
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
            :disabled="actionLoading"
            class="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отправка...' : 'Отправить' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
