<script setup lang="ts">
import { computed, watch } from 'vue'
import { useFreightRequestCompletion } from '@/composables/useFreightRequestCompletion'
import { useTutorialEvent } from '@/composables/useTutorialEvent'
import StarRating from './StarRating.vue'
import type { FreightRequest } from '@/types/freightRequest'
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

// Tutorial events
const { emit: emitTutorial } = useTutorialEvent()

// Use composable for completion/review logic
const freightRequestRef = computed(() => props.freightRequest)

const {
  isLoading,
  error,
  showCompleteConfirm,
  showReviewModal,
  isEditingReview,
  reviewRating,
  reviewComment,
  isCustomer,
  hasCompleted,
  canComplete,
  myReview,
  counterpartyReview,
  canEditReview,
  editTimeRemaining,
  counterpartyName,
  handleComplete: doComplete,
  openReviewModal: doOpenReviewModal,
  submitReview: doSubmitReview,
  skipReview,
} = useFreightRequestCompletion({
  freightRequest: freightRequestRef,
  onCompleted: () => emit('completed'),
  onReviewLeft: () => emit('reviewLeft'),
  onReviewEdited: () => emit('reviewEdited'),
})

// Watch for confirm modal opening
watch(showCompleteConfirm, (opened) => {
  if (opened) {
    emitTutorial('completion:confirmOpened', undefined)
  }
})

// Watch for rating selection
watch(reviewRating, (rating) => {
  emitTutorial('review:ratingSelected', { rating })
})

// Wrap actions to emit tutorial events
async function handleComplete() {
  await doComplete()
  emitTutorial('completion:completed', { frId: props.freightRequest.id })
}

function openReviewModal(edit = false) {
  if (edit) {
    emitTutorial('review:editOpened', undefined)
  }
  doOpenReviewModal(edit)
}

async function submitReview() {
  await doSubmitReview()
  if (isEditingReview.value) {
    emitTutorial('review:edited', { frId: props.freightRequest.id })
  } else {
    emitTutorial('review:submitted', { frId: props.freightRequest.id, reviewId: myReview.value?.id || '' })
  }
}
</script>

<template>
  <Card class="mt-4" data-tutorial="completion-section">
    <CardHeader>
      <CardTitle class="flex items-center gap-2">
        <CheckCircle2 class="h-5 w-5" />
        Завершение и отзывы
      </CardTitle>
    </CardHeader>
    <CardContent class="space-y-4">
      <!-- Completion Status -->
      <div class="space-y-2" data-tutorial="completion-status">
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
        <Button @click="showCompleteConfirm = true" :disabled="isLoading" data-tutorial="complete-btn">
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
            <Button variant="outline" size="sm" @click="openReviewModal(true)" data-tutorial="edit-review-btn">
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
    <DialogContent class="sm:max-w-md" data-tutorial="complete-confirm-modal">
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
        <Button @click="handleComplete" :disabled="isLoading" data-tutorial="complete-confirm-btn">
          Завершить
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Review Modal -->
  <Dialog v-model:open="showReviewModal">
    <DialogContent class="sm:max-w-md" data-tutorial="review-modal">
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
          <StarRating v-model="reviewRating" size="lg" data-tutorial="review-rating" />
        </div>

        <div class="space-y-2">
          <Label for="comment">Комментарий (необязательно)</Label>
          <Textarea
            id="comment"
            v-model="reviewComment"
            placeholder="Опишите ваш опыт работы..."
            rows="3"
            data-tutorial="review-comment"
          />
        </div>

        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
      </div>

      <DialogFooter class="gap-2">
        <Button variant="ghost" @click="skipReview" data-tutorial="review-skip-btn">
          Пропустить
        </Button>
        <Button @click="submitReview" :disabled="isLoading || reviewRating < 1" data-tutorial="review-submit-btn">
          {{ isEditingReview ? 'Сохранить' : 'Отправить' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
