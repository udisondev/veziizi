<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { freightRequestsApi } from '@/api/freightRequests'
import StarRating from './StarRating.vue'
import type { FreightRequest, FreightReview } from '@/types/freightRequest'
import { formatDateTime } from '@/utils/formatters'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// Icons
import { Check, CheckCircle2, Clock, Pencil, Star } from 'lucide-vue-next'

const props = defineProps<{
  freightRequest: FreightRequest
}>()

const emit = defineEmits<{
  completed: []
  reviewLeft: []
  reviewEdited: []
}>()

const auth = useAuthStore()

// State
const isLoading = ref(false)
const error = ref('')

// Modals
const showCompleteConfirm = ref(false)
const showReviewModal = ref(false)
const isEditingReview = ref(false)

// Review form
const reviewRating = ref(5)
const reviewComment = ref('')

// Computed properties
const isCustomer = computed(() => props.freightRequest.customer_org_id === auth.organizationId)
const isCarrier = computed(() => props.freightRequest.carrier_org_id === auth.organizationId)

const hasCompleted = computed(() => {
  if (isCustomer.value) return props.freightRequest.customer_completed
  if (isCarrier.value) return props.freightRequest.carrier_completed
  return false
})

const canComplete = computed(() => {
  const status = props.freightRequest.status
  const canCompleteStatuses = ['confirmed', 'partially_completed']
  return canCompleteStatuses.includes(status) && !hasCompleted.value
})

const myReview = computed((): FreightReview | undefined => {
  if (isCustomer.value) return props.freightRequest.customer_review
  if (isCarrier.value) return props.freightRequest.carrier_review
  return undefined
})

const counterpartyReview = computed((): FreightReview | undefined => {
  if (isCustomer.value) return props.freightRequest.carrier_review
  if (isCarrier.value) return props.freightRequest.customer_review
  return undefined
})

const canLeaveReview = computed(() => {
  return hasCompleted.value && !myReview.value
})

const canEditReview = computed(() => {
  return myReview.value?.can_edit ?? false
})

const editTimeRemaining = computed(() => {
  if (!myReview.value?.edit_expires_at) return null
  const expiresAt = new Date(myReview.value.edit_expires_at)
  const now = new Date()
  const diffMs = expiresAt.getTime() - now.getTime()
  if (diffMs <= 0) return null

  const hours = Math.floor(diffMs / (1000 * 60 * 60))
  const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))

  if (hours > 0) {
    return `${hours} ч ${minutes} мин`
  }
  return `${minutes} мин`
})

const counterpartyName = computed(() => {
  if (isCustomer.value) {
    return props.freightRequest.carrier_org_name || 'Перевозчик'
  }
  return props.freightRequest.customer_org_name || 'Заказчик'
})

// Actions
async function handleComplete() {
  isLoading.value = true
  error.value = ''
  try {
    await freightRequestsApi.complete(props.freightRequest.id)
    showCompleteConfirm.value = false
    // Показываем модал для отзыва
    showReviewModal.value = true
    emit('completed')
  } catch (e: any) {
    error.value = e.message || 'Ошибка при завершении'
  } finally {
    isLoading.value = false
  }
}

function openReviewModal(edit = false) {
  isEditingReview.value = edit
  if (edit && myReview.value) {
    reviewRating.value = myReview.value.rating
    reviewComment.value = myReview.value.comment || ''
  } else {
    reviewRating.value = 5
    reviewComment.value = ''
  }
  showReviewModal.value = true
}

async function submitReview() {
  isLoading.value = true
  error.value = ''
  try {
    const data = {
      rating: reviewRating.value,
      comment: reviewComment.value || undefined,
    }

    if (isEditingReview.value) {
      await freightRequestsApi.editReview(props.freightRequest.id, data)
      emit('reviewEdited')
    } else {
      await freightRequestsApi.leaveReview(props.freightRequest.id, data)
      emit('reviewLeft')
    }
    showReviewModal.value = false
  } catch (e: any) {
    error.value = e.message || 'Ошибка при сохранении отзыва'
  } finally {
    isLoading.value = false
  }
}

function skipReview() {
  showReviewModal.value = false
}
</script>

<template>
  <Card class="mt-4">
    <CardHeader>
      <CardTitle class="flex items-center gap-2">
        <CheckCircle2 class="h-5 w-5" />
        Завершение и отзывы
      </CardTitle>
    </CardHeader>
    <CardContent class="space-y-4">
      <!-- Completion Status -->
      <div class="space-y-2">
        <h4 class="font-medium">Статус завершения</h4>
        <div class="flex flex-col gap-2 text-sm">
          <div class="flex items-center gap-2">
            <Check
              v-if="freightRequest.customer_completed"
              class="h-4 w-4 text-green-600"
            />
            <Clock v-else class="h-4 w-4 text-muted-foreground" />
            <span>
              Заказчик:
              <span v-if="freightRequest.customer_completed" class="text-green-600">
                завершил {{ freightRequest.customer_completed_at ? formatDateTime(freightRequest.customer_completed_at) : '' }}
              </span>
              <span v-else class="text-muted-foreground">ожидает</span>
            </span>
          </div>
          <div class="flex items-center gap-2">
            <Check
              v-if="freightRequest.carrier_completed"
              class="h-4 w-4 text-green-600"
            />
            <Clock v-else class="h-4 w-4 text-muted-foreground" />
            <span>
              Перевозчик:
              <span v-if="freightRequest.carrier_completed" class="text-green-600">
                завершил {{ freightRequest.carrier_completed_at ? formatDateTime(freightRequest.carrier_completed_at) : '' }}
              </span>
              <span v-else class="text-muted-foreground">ожидает</span>
            </span>
          </div>
        </div>
      </div>

      <!-- Complete Button -->
      <div v-if="canComplete">
        <Button @click="showCompleteConfirm = true" :disabled="isLoading">
          <Check class="mr-2 h-4 w-4" />
          Завершить перевозку
        </Button>
      </div>

      <!-- My Review Section -->
      <div v-if="hasCompleted" class="space-y-2 border-t pt-4">
        <h4 class="font-medium">Ваш отзыв о {{ counterpartyName }}</h4>

        <div v-if="myReview" class="space-y-2">
          <div class="flex items-center gap-2">
            <StarRating :model-value="myReview.rating" readonly size="sm" />
            <span class="text-sm text-muted-foreground">
              {{ formatDateTime(myReview.created_at) }}
            </span>
          </div>
          <p v-if="myReview.comment" class="text-sm">{{ myReview.comment }}</p>

          <div v-if="canEditReview" class="flex items-center gap-2">
            <Button variant="outline" size="sm" @click="openReviewModal(true)">
              <Pencil class="mr-2 h-3 w-3" />
              Редактировать
            </Button>
            <span v-if="editTimeRemaining" class="text-xs text-muted-foreground">
              Осталось {{ editTimeRemaining }}
            </span>
          </div>
        </div>

        <div v-else>
          <Button variant="outline" @click="openReviewModal(false)">
            <Star class="mr-2 h-4 w-4" />
            Оставить отзыв
          </Button>
        </div>
      </div>

      <!-- Counterparty Review Section -->
      <div v-if="counterpartyReview" class="space-y-2 border-t pt-4">
        <h4 class="font-medium">
          Отзыв от {{ isCustomer ? 'перевозчика' : 'заказчика' }}
        </h4>
        <div class="flex items-center gap-2">
          <StarRating :model-value="counterpartyReview.rating" readonly size="sm" />
          <span class="text-sm text-muted-foreground">
            {{ formatDateTime(counterpartyReview.created_at) }}
          </span>
        </div>
        <p v-if="counterpartyReview.comment" class="text-sm">
          {{ counterpartyReview.comment }}
        </p>
      </div>

      <!-- Error -->
      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
    </CardContent>
  </Card>

  <!-- Complete Confirmation -->
  <Dialog v-model:open="showCompleteConfirm">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Завершить перевозку?</DialogTitle>
        <DialogDescription>
          Вы подтверждаете, что перевозка выполнена. После подтверждения вы сможете оставить отзыв о контрагенте.
        </DialogDescription>
      </DialogHeader>
      <DialogFooter>
        <Button variant="outline" @click="showCompleteConfirm = false">
          Отмена
        </Button>
        <Button @click="handleComplete" :disabled="isLoading">
          Завершить
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Review Modal -->
  <Dialog v-model:open="showReviewModal">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>
          {{ isEditingReview ? 'Редактировать отзыв' : 'Оставить отзыв' }}
        </DialogTitle>
        <DialogDescription>
          {{ isEditingReview
            ? 'Вы можете изменить отзыв в течение 24 часов после создания'
            : `Оцените работу с ${counterpartyName}`
          }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-4">
        <div class="space-y-2">
          <Label>Оценка</Label>
          <StarRating v-model="reviewRating" size="lg" />
        </div>

        <div class="space-y-2">
          <Label for="comment">Комментарий (необязательно)</Label>
          <Textarea
            id="comment"
            v-model="reviewComment"
            placeholder="Опишите ваш опыт работы..."
            rows="3"
          />
        </div>

        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
      </div>

      <DialogFooter class="gap-2 sm:gap-0">
        <Button v-if="!isEditingReview" variant="outline" @click="skipReview">
          Пропустить
        </Button>
        <Button @click="submitReview" :disabled="isLoading || reviewRating < 1">
          {{ isEditingReview ? 'Сохранить' : 'Отправить отзыв' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
