<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { PendingReview } from '@/types/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
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
  RefreshCcw,
  LogOut,
  Star,
  AlertTriangle,
  Check,
  X,
  Headphones,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
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

const navItems = [
  { to: '/admin/organizations', label: 'Организации', icon: Building2 },
  { to: '/admin/reviews', label: 'Отзывы', icon: Star },
  { to: '/admin/fraudsters', label: 'Накрутчики', icon: AlertTriangle },
  { to: '/admin/support', label: 'Поддержка', icon: Headphones },
]

const severityVariants: Record<string, 'default' | 'warning' | 'destructive'> = {
  low: 'warning',
  medium: 'warning',
  high: 'destructive',
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
          <h2 class="text-2xl font-bold text-white">Отзывы на модерации</h2>
          <p class="text-sm text-slate-400 mt-1">Всего: {{ total }}</p>
        </div>
        <Button
          variant="outline"
          class="border-slate-600 text-slate-300 hover:bg-slate-700 hover:text-white"
          :disabled="isLoading"
          @click="loadReviews"
        >
          <RefreshCcw class="h-4 w-4 mr-2" :class="{ 'animate-spin': isLoading }" />
          Обновить
        </Button>
      </div>

      <!-- Error -->
      <ErrorBanner
        v-if="error"
        :message="error"
        @retry="loadReviews"
        class="mb-6"
      />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Empty -->
      <Card v-else-if="reviews.length === 0" class="bg-slate-800 border-slate-700">
        <CardContent class="py-12 text-center">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-700 mb-4">
            <Star class="h-8 w-8 text-slate-400" />
          </div>
          <h3 class="text-lg font-medium text-white mb-2">Нет отзывов на модерации</h3>
          <p class="text-slate-400">Все отзывы обработаны</p>
        </CardContent>
      </Card>

      <!-- List -->
      <div v-else class="space-y-4">
        <Card
          v-for="review in reviews"
          :key="review.id"
          class="bg-slate-800 border-slate-700"
        >
          <CardContent class="p-6">
            <div class="flex flex-col lg:flex-row lg:justify-between lg:items-start gap-4 mb-4">
              <div>
                <!-- Rating -->
                <div class="flex items-center gap-3 mb-2">
                  <div class="flex text-yellow-400">
                    <Star
                      v-for="i in 5"
                      :key="i"
                      class="h-5 w-5"
                      :class="i <= review.rating ? 'fill-current' : 'fill-none'"
                    />
                  </div>
                  <span class="text-white font-medium">{{ review.rating }}/5</span>
                  <Badge variant="secondary" class="bg-slate-700 text-slate-300">
                    Fraud: {{ (review.fraud_score * 100).toFixed(0) }}%
                  </Badge>
                </div>

                <!-- Comment -->
                <p v-if="review.comment" class="text-slate-300 mb-3 break-words">
                  {{ review.comment }}
                </p>

                <!-- Meta -->
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-slate-400">
                  <span>Сумма: {{ formatCurrency(review.order_amount, review.order_currency) }}</span>
                  <span>Вес: {{ review.raw_weight.toFixed(2) }}</span>
                  <span>{{ formatDate(review.created_at) }}</span>
                </div>
              </div>

              <!-- Actions -->
              <div class="flex gap-2 shrink-0">
                <Button
                  size="sm"
                  class="bg-green-600 hover:bg-green-500 text-white"
                  @click="openApproveModal(review)"
                >
                  <Check class="h-4 w-4 mr-1" />
                  Одобрить
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  @click="openRejectModal(review)"
                >
                  <X class="h-4 w-4 mr-1" />
                  Отклонить
                </Button>
              </div>
            </div>

            <!-- Fraud Signals -->
            <div v-if="review.fraud_signals.length > 0" class="mt-4 pt-4 border-t border-slate-700">
              <p class="text-sm text-slate-400 mb-2">Обнаруженные сигналы:</p>
              <div class="flex flex-wrap gap-2">
                <Badge
                  v-for="(signal, idx) in review.fraud_signals"
                  :key="idx"
                  :variant="severityVariants[signal.severity]"
                >
                  {{ signalTypeLabels[signal.type] || signal.type }}
                </Badge>
              </div>
            </div>

            <!-- IDs -->
            <div class="mt-4 pt-4 border-t border-slate-700 text-xs text-slate-600 font-mono">
              Review: {{ review.id.slice(0, 8) }}...
            </div>
          </CardContent>
        </Card>
      </div>
    </main>

    <!-- Approve Modal -->
    <Dialog v-model:open="showApproveModal">
      <DialogContent class="bg-slate-800 border-slate-700 text-white sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-white">Одобрить отзыв</DialogTitle>
          <DialogDescription class="text-slate-400">
            Установите итоговый вес отзыва
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-4">
          <div class="space-y-2">
            <Label class="text-slate-200">Итоговый вес</Label>
            <Input
              v-model.number="approveWeight"
              type="number"
              min="0"
              max="1"
              step="0.1"
              class="bg-slate-700 border-slate-600 text-white"
            />
            <p class="text-xs text-slate-500">
              От 0 до 1. Исходный вес: {{ selectedReview?.raw_weight.toFixed(2) }}
            </p>
          </div>

          <div class="space-y-2">
            <Label class="text-slate-200">Примечание (необязательно)</Label>
            <Textarea
              v-model="approveNote"
              rows="2"
              class="bg-slate-700 border-slate-600 text-white resize-none"
              placeholder="Причина изменения веса..."
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="ghost"
            class="text-slate-400 hover:text-white"
            :disabled="isSubmitting"
            @click="closeModals"
          >
            Отмена
          </Button>
          <Button
            class="bg-green-600 hover:bg-green-500 text-white"
            :disabled="isSubmitting"
            @click="submitApprove"
          >
            {{ isSubmitting ? 'Сохранение...' : 'Одобрить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Reject Modal -->
    <Dialog v-model:open="showRejectModal">
      <DialogContent class="bg-slate-800 border-slate-700 text-white sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-white">Отклонить отзыв</DialogTitle>
          <DialogDescription class="text-slate-400">
            Укажите причину отклонения
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label class="text-slate-200">Причина отклонения</Label>
          <Textarea
            v-model="rejectReason"
            rows="3"
            class="bg-slate-700 border-slate-600 text-white resize-none"
            placeholder="Укажите причину..."
          />
        </div>

        <DialogFooter>
          <Button
            variant="ghost"
            class="text-slate-400 hover:text-white"
            :disabled="isSubmitting"
            @click="closeModals"
          >
            Отмена
          </Button>
          <Button
            variant="destructive"
            :disabled="isSubmitting || !rejectReason.trim()"
            @click="submitReject"
          >
            {{ isSubmitting ? 'Сохранение...' : 'Отклонить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
