<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { PendingReview } from '@/types/admin'

const router = useRouter()
const admin = useAdminStore()

const reviews = ref<PendingReview[]>([])
const total = ref(0)
const isLoading = ref(true)
const error = ref('')

// Modal state
const showApproveModal = ref(false)
const showRejectModal = ref(false)
const selectedReview = ref<PendingReview | null>(null)
const approveWeight = ref(1.0)
const approveNote = ref('')
const rejectReason = ref('')
const isSubmitting = ref(false)

const severityColors: Record<string, string> = {
  low: 'bg-yellow-900/50 text-yellow-200 border-yellow-500',
  medium: 'bg-orange-900/50 text-orange-200 border-orange-500',
  high: 'bg-red-900/50 text-red-200 border-red-500',
}

const signalTypeLabels: Record<string, string> = {
  mutual_reviews: 'Взаимные отзывы',
  fast_completion: 'Быстрое завершение',
  perfect_ratings: 'Только 5 звёзд',
  new_org_burst: 'Бурный рост отзывов',
  same_ip: 'Совпадение IP',
  same_fingerprint: 'Совпадение устройств',
}

onMounted(async () => {
  await loadReviews()
})

async function loadReviews() {
  isLoading.value = true
  error.value = ''
  try {
    const response = await adminApi.getPendingReviews()
    reviews.value = response.reviews
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

function formatCurrency(amount: number, currency: string): string {
  const symbols: Record<string, string> = { RUB: '₽', USD: '$', EUR: '€' }
  return `${(amount / 100).toLocaleString('ru-RU')} ${symbols[currency] || currency}`
}

function openApproveModal(review: PendingReview) {
  selectedReview.value = review
  approveWeight.value = review.raw_weight
  approveNote.value = ''
  showApproveModal.value = true
}

function openRejectModal(review: PendingReview) {
  selectedReview.value = review
  rejectReason.value = ''
  showRejectModal.value = true
}

function closeModals() {
  showApproveModal.value = false
  showRejectModal.value = false
  selectedReview.value = null
}

async function submitApprove() {
  if (!selectedReview.value) return
  isSubmitting.value = true
  try {
    await adminApi.approveReview(selectedReview.value.id, {
      final_weight: approveWeight.value,
      note: approveNote.value || undefined,
    })
    closeModals()
    await loadReviews()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка одобрения'
  } finally {
    isSubmitting.value = false
  }
}

async function submitReject() {
  if (!selectedReview.value || !rejectReason.value.trim()) return
  isSubmitting.value = true
  try {
    await adminApi.rejectReview(selectedReview.value.id, {
      reason: rejectReason.value.trim(),
    })
    closeModals()
    await loadReviews()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка отклонения'
  } finally {
    isSubmitting.value = false
  }
}

async function handleLogout() {
  await admin.logout()
  router.push('/admin/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-900">
    <!-- Header -->
    <header class="bg-gray-800 shadow">
      <div class="max-w-7xl mx-auto px-4 py-4 flex justify-between items-center">
        <div class="flex items-center gap-6">
          <h1 class="text-xl font-bold text-white">Панель администратора</h1>
          <nav class="flex gap-4">
            <router-link to="/admin/organizations" class="text-gray-400 hover:text-white text-sm">
              Организации
            </router-link>
            <router-link to="/admin/reviews" class="text-white text-sm font-medium">
              Отзывы
            </router-link>
            <router-link to="/admin/fraudsters" class="text-gray-400 hover:text-white text-sm">
              Накрутчики
            </router-link>
          </nav>
        </div>
        <div class="flex items-center gap-4">
          <span class="text-gray-400 text-sm">{{ admin.email }}</span>
          <button
            @click="handleLogout"
            class="text-gray-400 hover:text-white text-sm"
          >
            Выйти
          </button>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-7xl mx-auto px-4 py-8">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h2 class="text-2xl font-bold text-white">Отзывы на модерации</h2>
          <p class="text-gray-400 text-sm mt-1">Всего: {{ total }}</p>
        </div>
        <button
          @click="loadReviews"
          :disabled="isLoading"
          class="px-4 py-2 text-sm bg-gray-700 text-white rounded hover:bg-gray-600 disabled:opacity-50"
        >
          Обновить
        </button>
      </div>

      <!-- Error -->
      <div v-if="error" class="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded mb-6">
        {{ error }}
      </div>

      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="text-gray-400">Загрузка...</div>
      </div>

      <!-- Empty -->
      <div v-else-if="reviews.length === 0" class="text-center py-12">
        <div class="text-gray-400 text-lg">Нет отзывов на модерации</div>
        <p class="text-gray-500 mt-2">Все отзывы обработаны</p>
      </div>

      <!-- List -->
      <div v-else class="space-y-4">
        <div
          v-for="review in reviews"
          :key="review.id"
          class="bg-gray-800 rounded-lg p-6"
        >
          <div class="flex justify-between items-start mb-4">
            <div>
              <div class="flex items-center gap-3 mb-2">
                <div class="flex text-yellow-400">
                  <span v-for="i in 5" :key="i" class="text-lg">
                    {{ i <= review.rating ? '★' : '☆' }}
                  </span>
                </div>
                <span class="text-white font-medium">{{ review.rating }}/5</span>
                <span class="text-gray-500">|</span>
                <span class="text-gray-400 text-sm">
                  Fraud Score: {{ (review.fraud_score * 100).toFixed(0) }}%
                </span>
              </div>
              <p v-if="review.comment" class="text-gray-300 mb-2 break-words">{{ review.comment }}</p>
              <div class="text-sm text-gray-500">
                <span>Сумма заказа: {{ formatCurrency(review.order_amount, review.order_currency) }}</span>
                <span class="mx-2">|</span>
                <span>Вес: {{ review.raw_weight.toFixed(2) }}</span>
                <span class="mx-2">|</span>
                <span>{{ formatDate(review.created_at) }}</span>
              </div>
            </div>
            <div class="flex gap-2">
              <button
                @click="openApproveModal(review)"
                class="px-3 py-1.5 text-sm bg-green-600 text-white rounded hover:bg-green-500"
              >
                Одобрить
              </button>
              <button
                @click="openRejectModal(review)"
                class="px-3 py-1.5 text-sm bg-red-600 text-white rounded hover:bg-red-500"
              >
                Отклонить
              </button>
            </div>
          </div>

          <!-- Fraud Signals -->
          <div v-if="review.fraud_signals.length > 0" class="mt-4">
            <div class="text-sm font-medium text-gray-400 mb-2">Обнаруженные сигналы:</div>
            <div class="flex flex-wrap gap-2">
              <div
                v-for="(signal, idx) in review.fraud_signals"
                :key="idx"
                :class="['px-3 py-1.5 rounded border text-sm', severityColors[signal.severity]]"
              >
                <div class="font-medium">
                  {{ signalTypeLabels[signal.type] || signal.type }}
                </div>
                <div class="text-xs opacity-80 break-words">{{ signal.description }}</div>
              </div>
            </div>
          </div>

          <!-- IDs -->
          <div class="mt-4 pt-4 border-t border-gray-700 text-xs text-gray-600">
            <span>Review: {{ review.id.slice(0, 8) }}...</span>
            <span class="mx-2">|</span>
            <span>Order: {{ review.order_id.slice(0, 8) }}...</span>
            <span class="mx-2">|</span>
            <span>Reviewer: {{ review.reviewer_org_id.slice(0, 8) }}...</span>
            <span class="mx-2">|</span>
            <span>Reviewed: {{ review.reviewed_org_id.slice(0, 8) }}...</span>
          </div>
        </div>
      </div>
    </main>

    <!-- Approve Modal -->
    <div v-if="showApproveModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-lg font-bold text-white mb-4">Одобрить отзыв</h3>

        <div class="mb-4">
          <label class="block text-sm text-gray-400 mb-1">Итоговый вес</label>
          <input
            v-model.number="approveWeight"
            type="number"
            min="0"
            max="1"
            step="0.1"
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white"
          />
          <p class="text-xs text-gray-500 mt-1">От 0 до 1. Исходный вес: {{ selectedReview?.raw_weight.toFixed(2) }}</p>
        </div>

        <div class="mb-6">
          <label class="block text-sm text-gray-400 mb-1">Примечание (необязательно)</label>
          <textarea
            v-model="approveNote"
            rows="2"
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white resize-none"
            placeholder="Причина изменения веса..."
          ></textarea>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="closeModals"
            :disabled="isSubmitting"
            class="px-4 py-2 text-gray-400 hover:text-white"
          >
            Отмена
          </button>
          <button
            @click="submitApprove"
            :disabled="isSubmitting"
            class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-500 disabled:opacity-50"
          >
            {{ isSubmitting ? 'Сохранение...' : 'Одобрить' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Reject Modal -->
    <div v-if="showRejectModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-lg font-bold text-white mb-4">Отклонить отзыв</h3>

        <div class="mb-6">
          <label class="block text-sm text-gray-400 mb-1">Причина отклонения</label>
          <textarea
            v-model="rejectReason"
            rows="3"
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white resize-none"
            placeholder="Укажите причину..."
          ></textarea>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="closeModals"
            :disabled="isSubmitting"
            class="px-4 py-2 text-gray-400 hover:text-white"
          >
            Отмена
          </button>
          <button
            @click="submitReject"
            :disabled="isSubmitting || !rejectReason.trim()"
            class="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-500 disabled:opacity-50"
          >
            {{ isSubmitting ? 'Сохранение...' : 'Отклонить' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
