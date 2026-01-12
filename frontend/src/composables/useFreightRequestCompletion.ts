/**
 * Composable для завершения заявок и управления отзывами
 * Извлекает общую логику из FreightRequestDetailView и FreightRequestCompletionSection
 */

import { ref, computed, type Ref, type ComputedRef } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { freightRequestsApi } from '@/api/freightRequests'
import { getErrorMessage } from '@/api/errors'
import type { FreightRequest, FreightReview } from '@/types/freightRequest'

export interface UseFreightRequestCompletionOptions {
  freightRequest: Ref<FreightRequest | null> | ComputedRef<FreightRequest | null>
  onCompleted?: () => void | Promise<void>
  onReviewLeft?: () => void | Promise<void>
  onReviewEdited?: () => void | Promise<void>
}

export interface UseFreightRequestCompletionReturn {
  // State
  isLoading: Ref<boolean>
  error: Ref<string>

  // Modal state
  showCompleteConfirm: Ref<boolean>
  showReviewModal: Ref<boolean>
  isEditingReview: Ref<boolean>

  // Review form
  reviewRating: Ref<number>
  reviewComment: Ref<string>

  // Role checks
  isCustomer: ComputedRef<boolean>
  isCarrier: ComputedRef<boolean>

  // Completion status
  hasCompleted: ComputedRef<boolean>
  canComplete: ComputedRef<boolean>

  // Review status
  myReview: ComputedRef<FreightReview | undefined>
  counterpartyReview: ComputedRef<FreightReview | undefined>
  canLeaveReview: ComputedRef<boolean>
  canEditReview: ComputedRef<boolean>
  canInteractWithReview: ComputedRef<boolean>
  editTimeRemaining: ComputedRef<string | null>

  // Labels
  counterpartyName: ComputedRef<string>

  // Actions
  handleComplete: () => Promise<void>
  openReviewModal: (edit?: boolean) => void
  submitReview: () => Promise<void>
  skipReview: () => void
  resetError: () => void
}

export function useFreightRequestCompletion(
  options: UseFreightRequestCompletionOptions
): UseFreightRequestCompletionReturn {
  const { freightRequest, onCompleted, onReviewLeft, onReviewEdited } = options
  const auth = useAuthStore()

  // State
  const isLoading = ref(false)
  const error = ref('')

  // Modal state
  const showCompleteConfirm = ref(false)
  const showReviewModal = ref(false)
  const isEditingReview = ref(false)

  // Review form
  const reviewRating = ref(5)
  const reviewComment = ref('')

  // Role checks
  const isCustomer = computed(() => {
    return freightRequest.value?.customer_org_id === auth.organizationId
  })

  const isCarrier = computed(() => {
    return freightRequest.value?.carrier_org_id === auth.organizationId
  })

  // Completion status
  const hasCompleted = computed(() => {
    if (!freightRequest.value) return false
    if (isCustomer.value) return freightRequest.value.customer_completed
    if (isCarrier.value) return freightRequest.value.carrier_completed
    return false
  })

  const canComplete = computed(() => {
    if (!freightRequest.value) return false
    const status = freightRequest.value.status
    const canCompleteStatuses = ['confirmed', 'partially_completed']
    return canCompleteStatuses.includes(status) && !hasCompleted.value
  })

  // Review status
  const myReview = computed((): FreightReview | undefined => {
    if (!freightRequest.value) return undefined
    if (isCustomer.value) return freightRequest.value.customer_review
    if (isCarrier.value) return freightRequest.value.carrier_review
    return undefined
  })

  const counterpartyReview = computed((): FreightReview | undefined => {
    if (!freightRequest.value) return undefined
    if (isCustomer.value) return freightRequest.value.carrier_review
    if (isCarrier.value) return freightRequest.value.customer_review
    return undefined
  })

  const canLeaveReview = computed(() => {
    return hasCompleted.value && !myReview.value
  })

  const canEditReview = computed(() => {
    return myReview.value?.can_edit ?? false
  })

  const canInteractWithReview = computed(() => {
    if (!hasCompleted.value) return false
    if (!myReview.value) return true // Можно оставить
    return myReview.value.can_edit // Можно редактировать (24ч)
  })

  const editTimeRemaining = computed((): string | null => {
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

  // Labels
  const counterpartyName = computed(() => {
    if (!freightRequest.value) return 'Контрагент'
    if (isCustomer.value) {
      return freightRequest.value.carrier_org_name || 'Перевозчик'
    }
    return freightRequest.value.customer_org_name || 'Заказчик'
  })

  // Actions
  async function handleComplete() {
    if (!freightRequest.value) return
    isLoading.value = true
    error.value = ''
    try {
      await freightRequestsApi.complete(freightRequest.value.id)
      showCompleteConfirm.value = false
      // Показываем модал для отзыва
      reviewRating.value = 5
      reviewComment.value = ''
      isEditingReview.value = false
      showReviewModal.value = true
      await onCompleted?.()
    } catch (e) {
      error.value = getErrorMessage(e)
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
      reviewRating.value = myReview.value?.rating ?? 5
      reviewComment.value = myReview.value?.comment ?? ''
    }
    showReviewModal.value = true
  }

  async function submitReview() {
    if (!freightRequest.value) return
    isLoading.value = true
    error.value = ''
    try {
      const data = {
        rating: reviewRating.value,
        comment: reviewComment.value || undefined,
      }

      if (isEditingReview.value || myReview.value) {
        await freightRequestsApi.editReview(freightRequest.value.id, data)
        await onReviewEdited?.()
      } else {
        await freightRequestsApi.leaveReview(freightRequest.value.id, data)
        await onReviewLeft?.()
      }
      showReviewModal.value = false
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      isLoading.value = false
    }
  }

  function skipReview() {
    showReviewModal.value = false
  }

  function resetError() {
    error.value = ''
  }

  return {
    // State
    isLoading,
    error,

    // Modal state
    showCompleteConfirm,
    showReviewModal,
    isEditingReview,

    // Review form
    reviewRating,
    reviewComment,

    // Role checks
    isCustomer,
    isCarrier,

    // Completion status
    hasCompleted,
    canComplete,

    // Review status
    myReview,
    counterpartyReview,
    canLeaveReview,
    canEditReview,
    canInteractWithReview,
    editTimeRemaining,

    // Labels
    counterpartyName,

    // Actions
    handleComplete,
    openReviewModal,
    submitReview,
    skipReview,
    resetError,
  }
}
