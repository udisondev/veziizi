<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { freightRequestsApi } from '@/api/freightRequests'
import { historyApi } from '@/api/history'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import { useTutorialEvent } from '@/composables/useTutorialEvent'
import { useSandboxReady } from '@/composables/useSandboxReady'
import { tutorialBus } from '@/sandbox/events'
import LeafletMap from '@/components/freight-request/shared/LeafletMap.vue'
import EventHistory from '@/components/EventHistory.vue'
import FreightRequestOffersTab from '@/components/freight-request/FreightRequestOffersTab.vue'
import StarRating from '@/components/freight-request/StarRating.vue'
import type { FreightRequest, Offer, Currency } from '@/types/freightRequest'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  loadingTypeLabels,
  vatTypeLabels,
  paymentMethodLabels,
  paymentTermsLabels,
  adrClassLabels,
} from '@/types/freightRequest'
import { freightRequestStatusMap } from '@/constants/statusMaps'
import { formatDate, formatDateTime, formatMoney } from '@/utils/formatters'

// UI Components
import { Button } from '@/components/ui/button'
import { Tooltip } from '@/components/ui/tooltip'
import { AppLink } from '@/components/ui/app-link'
import RoutePoint from '@/components/freight-request/shared/RoutePoint.vue'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent } from '@/components/ui/tabs'

// Shared Components
import { DetailPageHeader, StatusBadge, LoadingSpinner, ErrorBanner, type TabItem } from '@/components/shared'
import BottomSheet from '@/components/shared/BottomSheet.vue'
import { TabsSlider } from '@/components/ui/tabs'

// Icons
import {
  FileText,
  Pencil,
  XCircle,
  MoreVertical,
  Users,
  MapPin,
  Package,
  Truck,
  CreditCard,
  MessageSquare,
  Check,
  X,
  Clock,
  Building2,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const permissions = usePermissions()
const { emit: emitTutorial } = useTutorialEvent()
const { waitForReady } = useSandboxReady()

// State
const freightRequest = ref<FreightRequest | null>(null)
const offers = ref<Offer[]>([])
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)

// Tabs
const currentTab = ref('details')

// Отправляем событие для туториала при смене таба
watch(currentTab, (newTab) => {
  if (newTab === 'offers') {
    emitTutorial('tab:offers')
    markOffersAsSeen()
  }
  if (newTab === 'details') emitTutorial('tab:details')
})

// Подсветка таба "Предложения" при наличии новых офферов
function offersSeenKey() {
  return `offers_seen_${route.params.id}`
}

const offerSeenAt = ref<string | null>(localStorage.getItem(offersSeenKey()))

function markOffersAsSeen() {
  const ts = new Date().toISOString()
  localStorage.setItem(offersSeenKey(), ts)
  offerSeenAt.value = ts
}

const hasNewOffers = computed(() => {
  if (!isOwner.value || offers.value.length === 0) return false
  if (!offerSeenAt.value) return true
  return offers.value.some(o => new Date(o.created_at) > new Date(offerSeenAt.value!))
})

// History loader
function loadFreightRequestHistory(limit: number, offset: number) {
  const id = route.params.id as string
  return historyApi.getFreightRequestHistory(id, { limit, offset })
}

// Check if user can view history
const canViewHistory = computed(() => {
  if (!freightRequest.value) return false
  if (freightRequest.value.customer_org_id !== auth.organizationId) return false
  return auth.role === 'owner' || auth.role === 'administrator'
})

const tabItems = computed((): TabItem[] => {
  const items: TabItem[] = [
    { value: 'details', label: 'Детали заявки', icon: FileText },
  ]
  if (visibleOffers.value.length > 0 || isOwner.value) {
    items.push({ value: 'offers', label: 'Предложения', icon: Users, badge: offers.value.length || undefined, highlight: hasNewOffers.value })
  }
  if (canViewHistory.value) {
    items.push({ value: 'history', label: 'История', icon: Clock, separator: true })
  }
  return items
})

// Inner detail tabs
const currentDetailTab = ref('route')
const routeExpanded = ref(false)

const detailTabItems = computed((): TabItem[] => {
  const items: TabItem[] = [
    { value: 'route', label: 'Маршрут', icon: MapPin },
    { value: 'cargo', label: 'Груз', icon: Package },
    { value: 'vehicle', label: 'Транспорт', icon: Truck },
    { value: 'payment', label: 'Оплата', icon: CreditCard },
  ]
  if (freightRequest.value?.comment) {
    items.push({ value: 'comment', label: 'Комментарий', icon: MessageSquare })
  }
  return items
})

// Modals
const showCancelModal = ref(false)
const showActionsSheet = ref(false)
const cancelReason = ref('')

const showRejectModal = ref(false)
const rejectOfferId = ref<string | null>(null)
const rejectReason = ref('')

const showWithdrawModal = ref(false)
const withdrawOfferId = ref<string | null>(null)
const withdrawReason = ref('')

// Completion & Review modals
const showCompleteConfirm = ref(false)
const showReviewModal = ref(false)
const reviewRating = ref(5)
const reviewComment = ref('')

// Offer action confirmation modals
const showSelectModal = ref(false)
const selectOfferId = ref<string | null>(null)

const showConfirmOfferModal = ref(false)
const confirmOfferId = ref<string | null>(null)

const showDeclineModal = ref(false)
const declineOfferId = ref<string | null>(null)
const declineReason = ref('')

const showUnselectModal = ref(false)
const unselectOfferId = ref<string | null>(null)
const unselectReason = ref('')

// Computed
const isOwner = computed(() => {
  if (!freightRequest.value) return false
  return permissions.isFreightRequestOwner(freightRequest.value.customer_org_id)
})

// Является ли текущая организация перевозчиком
const isCarrier = computed(() => {
  if (!freightRequest.value?.carrier_org_id) return false
  return freightRequest.value.carrier_org_id === auth.organizationId
})

// Показывать ли carrier info (перевозчику всегда, заказчику при confirmed+, другим — никогда)
const showCarrierInfo = computed(() => {
  if (!freightRequest.value?.carrier_org_id) return false
  const confirmedStatuses = ['confirmed', 'partially_completed', 'completed', 'cancelled_after_confirmed']
  // Перевозчик видит всегда, заказчик только при confirmed
  return isCarrier.value || (isOwner.value && confirmedStatuses.includes(freightRequest.value.status))
})

// Может ли переназначить ответственного перевозчика
const canReassignCarrier = computed(() => {
  if (!freightRequest.value) return false
  const confirmedStatuses = ['confirmed', 'partially_completed']
  return (
    permissions.canReassignCarrierMember(freightRequest.value.carrier_org_id) &&
    confirmedStatuses.includes(freightRequest.value.status)
  )
})

const canMakeOffer = computed(() => {
  if (!freightRequest.value) return false
  return (
    permissions.canCreateOffer(freightRequest.value.customer_org_id) &&
    ['published', 'selected'].includes(freightRequest.value.status)
  )
})

const canCancel = computed(() => {
  if (!freightRequest.value) return false
  return (
    permissions.canCancelFreightRequest(
      freightRequest.value.customer_org_id,
      freightRequest.value.customer_member_id
    ) &&
    ['published', 'selected'].includes(freightRequest.value.status)
  )
})

const canEdit = computed(() => {
  if (!freightRequest.value) return false
  return (
    permissions.canEditFreightRequest(
      freightRequest.value.customer_org_id,
      freightRequest.value.customer_member_id
    ) &&
    freightRequest.value.status === 'published'
  )
})

const canReassign = computed(() => {
  if (!freightRequest.value) return false
  const allowedStatuses = ['published', 'selected', 'confirmed', 'partially_completed']
  return (
    permissions.canReassignFreightRequest(freightRequest.value.customer_org_id) &&
    allowedStatuses.includes(freightRequest.value.status)
  )
})

const myActiveOffer = computed(() => {
  return offers.value.find(
    (o) =>
      o.carrier_org_id === auth.organizationId &&
      ['pending', 'selected'].includes(o.status)
  )
})

// Может ли текущий пользователь завершить заявку (только ответственный)
const canComplete = computed(() => {
  if (!freightRequest.value) return false
  const status = freightRequest.value.status
  if (!['confirmed', 'partially_completed'].includes(status)) return false

  // Проверка что это ответственный
  if (isOwner.value) {
    return freightRequest.value.customer_member_id === auth.memberId &&
           !freightRequest.value.customer_completed
  }
  if (isCarrier.value) {
    return freightRequest.value.carrier_member_id === auth.memberId &&
           !freightRequest.value.carrier_completed
  }
  return false
})

// Завершил ли текущий пользователь свою часть
const hasCompleted = computed(() => {
  if (!freightRequest.value) return false
  if (isOwner.value) return freightRequest.value.customer_completed
  if (isCarrier.value) return freightRequest.value.carrier_completed
  return false
})

// Мой отзыв
const myReview = computed(() => {
  if (!freightRequest.value) return null
  if (isOwner.value) return freightRequest.value.customer_review
  if (isCarrier.value) return freightRequest.value.carrier_review
  return null
})

// Можно ли оставить/редактировать отзыв
const canInteractWithReview = computed(() => {
  if (!hasCompleted.value) return false
  if (!myReview.value) return true // Можно оставить
  return myReview.value.can_edit // Можно редактировать (24ч)
})

const visibleOffers = computed(() => {
  if (isOwner.value) {
    return offers.value
  }
  return offers.value.filter((o) => o.carrier_org_id === auth.organizationId)
})

const requestNumber = computed(() => {
  if (!freightRequest.value) return 0
  return freightRequest.value.request_number
})

// Tutorial: emit event when completion confirm opens
watch(showCompleteConfirm, (opened) => {
  if (opened) {
    emitTutorial('completion:confirmOpened', undefined)
  }
})

// Tutorial: emit event when review rating changes
watch(reviewRating, (rating) => {
  if (showReviewModal.value) {
    emitTutorial('review:ratingSelected', { rating })
  }
})

// Methods
async function loadData() {
  isLoading.value = true
  error.value = ''

  // Ждём готовности sandbox перед загрузкой данных
  // (решает race condition при восстановлении сессии из localStorage)
  await waitForReady()

  try {
    const id = route.params.id as string
    const [fr, offersList] = await Promise.all([
      freightRequestsApi.get(id),
      freightRequestsApi.listOffers(id),
    ])
    freightRequest.value = fr
    offers.value = offersList
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatPrice(amount: number, currency: Currency): string {
  return formatMoney({ amount, currency })
}

function getPointTypeLabel(point: { is_loading: boolean; is_unloading: boolean }): string {
  if (point.is_loading && point.is_unloading) return 'Погрузка/Разгрузка'
  if (point.is_loading) return 'Погрузка'
  if (point.is_unloading) return 'Разгрузка'
  return 'Точка'
}

function getPointBadgeClass(point: { is_loading: boolean; is_unloading: boolean }): string {
  if (point.is_loading && point.is_unloading) return 'bg-accent border-primary/40 text-accent-foreground'
  if (point.is_loading) return 'bg-accent border-primary/40 text-accent-foreground'
  if (point.is_unloading) return 'bg-success/10 border-success/40 text-success'
  return 'bg-muted border-border text-muted-foreground'
}

// Actions
async function handleCancel() {
  if (!freightRequest.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.cancel(freightRequest.value.id, cancelReason.value || undefined)
    router.push('/')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
    showCancelModal.value = false
  }
}

function openSelectModal(offerId: string) {
  selectOfferId.value = offerId
  showSelectModal.value = true
  emitTutorial('selectModal:opened')
}

async function confirmSelectOffer() {
  if (!freightRequest.value || !selectOfferId.value) return
  actionLoading.value = true
  const offerId = selectOfferId.value
  try {
    await freightRequestsApi.selectOffer(freightRequest.value.id, offerId)
    emitTutorial('offer:selected', { frId: freightRequest.value.id, offerId })
    showSelectModal.value = false
    selectOfferId.value = null
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function openRejectModal(offerId: string) {
  rejectOfferId.value = offerId
  rejectReason.value = ''
  showRejectModal.value = true
  emitTutorial('rejectModal:opened')
}

async function confirmRejectOffer() {
  if (!freightRequest.value || !rejectOfferId.value) return
  actionLoading.value = true
  const offerId = rejectOfferId.value
  try {
    await freightRequestsApi.rejectOffer(freightRequest.value.id, offerId, rejectReason.value || undefined)
    emitTutorial('offer:rejected', { frId: freightRequest.value.id, offerId })
    showRejectModal.value = false
    rejectOfferId.value = null
    rejectReason.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function openWithdrawModal(offerId: string) {
  withdrawOfferId.value = offerId
  withdrawReason.value = ''
  showWithdrawModal.value = true
}

async function confirmWithdrawOffer() {
  if (!freightRequest.value || !withdrawOfferId.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.withdrawOffer(freightRequest.value.id, withdrawOfferId.value, withdrawReason.value || undefined)
    showWithdrawModal.value = false
    withdrawOfferId.value = null
    withdrawReason.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function goToReassign() {
  if (!freightRequest.value) return
  router.push({
    path: '/members',
    query: {
      selectFor: 'freightRequest',
      frId: freightRequest.value.id,
    },
  })
}

function goToReassignCarrier() {
  if (!freightRequest.value) return
  router.push({
    path: '/members',
    query: {
      selectFor: 'carrierMember',
      frId: freightRequest.value.id,
    },
  })
}

function openConfirmOfferModal(offerId: string) {
  confirmOfferId.value = offerId
  showConfirmOfferModal.value = true
}

async function confirmConfirmOffer() {
  if (!freightRequest.value || !confirmOfferId.value) return
  actionLoading.value = true
  const offerId = confirmOfferId.value
  try {
    await freightRequestsApi.confirmOffer(freightRequest.value.id, offerId)
    emitTutorial('offer:confirmed', { frId: freightRequest.value.id, offerId })
    showConfirmOfferModal.value = false
    confirmOfferId.value = null
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function openDeclineModal(offerId: string) {
  declineOfferId.value = offerId
  declineReason.value = ''
  showDeclineModal.value = true
}

async function confirmDeclineOffer() {
  if (!freightRequest.value || !declineOfferId.value) return
  actionLoading.value = true
  const offerId = declineOfferId.value
  try {
    await freightRequestsApi.declineOffer(freightRequest.value.id, offerId, declineReason.value || undefined)
    showDeclineModal.value = false
    declineOfferId.value = null
    declineReason.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function openUnselectModal(offerId: string) {
  unselectOfferId.value = offerId
  unselectReason.value = ''
  showUnselectModal.value = true
  emitTutorial('unselectModal:opened')
}

async function confirmUnselectOffer() {
  if (!freightRequest.value || !unselectOfferId.value) return
  actionLoading.value = true
  const offerId = unselectOfferId.value
  try {
    await freightRequestsApi.unselectOffer(freightRequest.value.id, offerId, unselectReason.value || undefined)
    emitTutorial('offer:unselected', { frId: freightRequest.value.id, offerId })
    showUnselectModal.value = false
    unselectOfferId.value = null
    unselectReason.value = ''
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

// Completion & Reviews
async function handleComplete() {
  if (!freightRequest.value) return
  actionLoading.value = true
  try {
    await freightRequestsApi.complete(freightRequest.value.id)
    showCompleteConfirm.value = false
    emitTutorial('completion:completed', { frId: freightRequest.value.id })
    // Показываем модал для отзыва
    reviewRating.value = 5
    reviewComment.value = ''
    showReviewModal.value = true
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

function openReviewModal(rating: number) {
  reviewRating.value = rating
  reviewComment.value = myReview.value?.comment ?? ''
  showReviewModal.value = true
}

async function submitReview() {
  if (!freightRequest.value) return
  actionLoading.value = true
  try {
    const data = { rating: reviewRating.value, comment: reviewComment.value || undefined }
    if (myReview.value) {
      await freightRequestsApi.editReview(freightRequest.value.id, data)
      emitTutorial('review:edited', { frId: freightRequest.value.id })
    } else {
      await freightRequestsApi.leaveReview(freightRequest.value.id, data)
      emitTutorial('review:submitted', { frId: freightRequest.value.id, reviewId: '' })
    }
    showReviewModal.value = false
    emitTutorial('review:closed', undefined)
    await loadData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

// Слушатель переключения вкладки из туториала
function handleTutorialTabSwitch() {
  currentTab.value = 'offers'
}

// Обработчик autoconfirm из sandbox — перезагружает данные
function handleAutoConfirm() {
  loadData()
}

onMounted(() => {
  loadData()
  // Подписываемся на события из туториала
  tutorialBus.on('tab:offers', handleTutorialTabSwitch)
  tutorialBus.on('offer:confirmed', handleAutoConfirm)
})

onUnmounted(() => {
  tutorialBus.off('tab:offers', handleTutorialTabSwitch)
  tutorialBus.off('offer:confirmed', handleAutoConfirm)
})
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <DetailPageHeader back-to="/" back-label="К списку заявок">
      <template #actions>
        <div class="flex items-center gap-2">
          <!-- Завершение заявки и отзывы -->
          <template v-if="isOwner || isCarrier">
            <!-- Кнопка Завершить (до завершения) -->
            <Button
              v-if="canComplete"
              size="sm"
              data-tutorial="complete-request-btn"
              @click="showCompleteConfirm = true"
            >
              <Check class="mr-2 h-4 w-4" />
              Завершить заявку
            </Button>

            <!-- Звёздочки (после завершения) -->
            <StarRating
              v-else-if="hasCompleted"
              :model-value="myReview?.rating ?? 0"
              :readonly="!canInteractWithReview"
              size="sm"
              @click="openReviewModal"
            />
          </template>

          <!-- Make Offer Button for carriers -->
          <Button
            v-if="canMakeOffer && !myActiveOffer"
            data-tutorial="make-offer-btn"
            size="sm"
            @click="router.push({ name: 'freight-request-make-offer', params: { id: freightRequest!.id } })"
          >
            Сделать предложение
          </Button>

          <!-- Мобиле: кнопка + BottomSheet -->
          <div v-if="freightRequest && (canEdit || canCancel)" class="md:hidden">
            <Button variant="ghost" size="icon" type="button" @click="showActionsSheet = true">
              <MoreVertical class="h-4 w-4" />
            </Button>
            <BottomSheet v-model="showActionsSheet" label="Действия">
              <div class="flex flex-col">
                <button
                  v-if="canEdit"
                  type="button"
                  class="w-full px-4 py-3.5 text-left text-base border-b border-border flex items-center gap-3 active:bg-accent"
                  @click="showActionsSheet = false; router.push(`/freight-requests/${freightRequest.id}/edit`)"
                >
                  <Pencil class="h-5 w-5 text-muted-foreground" />
                  Редактировать
                </button>
                <button
                  v-if="canCancel"
                  type="button"
                  class="w-full px-4 py-3.5 text-left text-base flex items-center gap-3 active:bg-destructive/10 text-destructive"
                  @click="showActionsSheet = false; showCancelModal = true"
                >
                  <XCircle class="h-5 w-5" />
                  Отменить заявку
                </button>
              </div>
            </BottomSheet>
          </div>

          <!-- Планшет и десктоп: отдельные кнопки -->
          <template v-if="freightRequest && (canEdit || canCancel)">
            <Button
              v-if="canEdit"
              variant="outline"
              size="sm"
              class="hidden md:inline-flex"
              @click="router.push(`/freight-requests/${freightRequest.id}/edit`)"
            >
              <Pencil class="mr-2 h-4 w-4" />
              Редактировать
            </Button>
            <Button
              v-if="canCancel"
              variant="outline"
              size="sm"
              class="hidden md:inline-flex text-destructive hover:text-destructive border-destructive/30 hover:border-destructive/60 hover:bg-destructive/5"
              @click="showCancelModal = true"
            >
              <XCircle class="mr-2 h-4 w-4" />
              Отменить заявку
            </Button>
          </template>
        </div>
      </template>
    </DetailPageHeader>

    <!-- Content -->
    <main class="max-w-5xl mx-auto px-4 py-6">
      <!-- Loading -->
      <LoadingSpinner v-if="isLoading" text="Загрузка заявки..." />

      <!-- Error -->
      <ErrorBanner
        v-else-if="error && !freightRequest"
        :message="error"
        @retry="loadData"
      />

      <!-- Content -->
      <div v-else-if="freightRequest" class="space-y-6">
        <!-- Error banner -->
        <Card v-if="error" class="border-destructive/50 bg-destructive/5">
          <CardContent class="flex items-center justify-between py-3">
            <span class="text-sm text-destructive">{{ error }}</span>
            <Button variant="ghost" size="sm" @click="error = ''">
              <X class="h-4 w-4" />
            </Button>
          </CardContent>
        </Card>

        <!-- Tabs -->
        <Tabs v-model="currentTab" class="w-full">
          <!-- Tab selector dropdown -->
          <div v-if="tabItems.length > 1" class="mb-6">
            <TabsSlider v-model="currentTab" :items="tabItems" />
          </div>

          <!-- Details Tab -->
          <TabsContent value="details" class="space-y-6">
            <!-- Header Card -->
            <Card>
              <CardContent class="p-4 sm:p-6">
                <div class="flex flex-col gap-4">
                  <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                    <div class="min-w-0">
                      <div class="flex items-center gap-3">
                        <h1 class="text-xl sm:text-2xl font-bold text-foreground">
                          Заявка #{{ requestNumber }}
                        </h1>
                        <StatusBadge :status="freightRequest.status" :status-map="freightRequestStatusMap" />
                      </div>
                      <div class="mt-3 grid grid-cols-[auto_1fr] items-center gap-x-6 gap-y-2 text-sm">
                        <!-- Заказчик -->
                        <span class="text-muted-foreground whitespace-nowrap">Заказчик</span>
                        <AppLink
                          :to="`/organizations/${freightRequest.customer_org_id}`"
                          class="inline-flex items-center gap-1 min-w-0"
                          :title="freightRequest.customer_org_name || 'Организация'"
                        >
                          <Building2 class="h-4 w-4 shrink-0" />
                          <span class="truncate">{{ freightRequest.customer_org_name || 'Организация' }}</span>
                        </AppLink>

                        <!-- Ответственный заказчика -->
                        <template v-if="freightRequest.customer_member_name">
                          <span class="text-muted-foreground whitespace-nowrap">Ответственный</span>
                          <div class="inline-flex items-center gap-1 min-w-0">
                            <AppLink
                              :to="`/members/${freightRequest.customer_member_id}`"
                              class="inline-flex items-center gap-1 min-w-0"
                              :title="freightRequest.customer_member_name"
                            >
                              <Users class="h-4 w-4 shrink-0" />
                              <span class="truncate">{{ freightRequest.customer_member_name }}</span>
                            </AppLink>
                            <Tooltip v-if="canReassign" text="Переназначить ответственного" side="top">
                              <Button
                                variant="ghost"
                                size="sm"
                                class="h-auto p-0.5 shrink-0"
                                @click="goToReassign"
                              >
                                <Pencil class="h-3 w-3" />
                              </Button>
                            </Tooltip>
                          </div>
                        </template>

                        <!-- Дата создания -->
                        <span class="text-muted-foreground whitespace-nowrap">Создано</span>
                        <span class="inline-flex items-center gap-1 text-muted-foreground">
                          <Clock class="h-4 w-4 shrink-0" />
                          {{ formatDateTime(freightRequest.created_at) }}
                        </span>
                      </div>

                      <!-- Carrier Info (когда confirmed или для перевозчика) -->
                      <div v-if="showCarrierInfo" data-tutorial="carrier-info" class="mt-3 pt-3 border-t border-border grid grid-cols-[auto_1fr] items-center gap-x-6 gap-y-2 text-sm">
                        <!-- Организация перевозчика -->
                        <template v-if="freightRequest.carrier_org_id">
                          <span class="text-muted-foreground whitespace-nowrap">Перевозчик</span>
                          <AppLink
                            :to="`/organizations/${freightRequest.carrier_org_id}`"
                            class="inline-flex items-center gap-1 min-w-0"
                            :title="freightRequest.carrier_org_name || 'Организация'"
                            data-tutorial="carrier-org-link"
                          >
                            <Building2 class="h-4 w-4 shrink-0" />
                            <span class="truncate">{{ freightRequest.carrier_org_name || 'Организация' }}</span>
                          </AppLink>
                        </template>

                        <!-- Ответственный перевозчика -->
                        <template v-if="freightRequest.carrier_member_id && freightRequest.carrier_member_name">
                          <span class="text-muted-foreground whitespace-nowrap">Ответственный</span>
                          <div class="inline-flex items-center gap-1 min-w-0">
                            <AppLink
                              :to="`/members/${freightRequest.carrier_member_id}`"
                              class="inline-flex items-center gap-1 min-w-0"
                              :title="freightRequest.carrier_member_name"
                              data-tutorial="carrier-member-link"
                            >
                              <Users class="h-4 w-4 shrink-0" />
                              <span class="truncate">{{ freightRequest.carrier_member_name }}</span>
                            </AppLink>
                            <Tooltip v-if="canReassignCarrier" text="Переназначить ответственного" side="top">
                              <Button
                                variant="ghost"
                                size="sm"
                                class="h-auto p-0.5 shrink-0"
                                @click="goToReassignCarrier"
                              >
                                <Pencil class="h-3 w-3" />
                              </Button>
                            </Tooltip>
                          </div>
                        </template>
                      </div>
                    </div>

                  </div>
                </div>
              </CardContent>
            </Card>

            <!-- Inner detail tabs -->
            <Card>
              <CardContent class="p-4 sm:p-6">
                <Tabs v-model="currentDetailTab" class="w-full">
                  <div class="mb-6">
                    <TabsSlider v-model="currentDetailTab" :items="detailTabItems" />
                  </div>

                  <!-- Маршрут -->
                  <TabsContent value="route" class="space-y-4">
                    <LeafletMap
                      :points="freightRequest.route.points"
                      height="300px"
                    />

                    <div class="mt-6">
                      <!-- Первая точка -->
                      <RoutePoint
                        :point="freightRequest.route.points[0]!"
                        :index="0"
                        :get-badge-class="getPointBadgeClass"
                        :get-type-label="getPointTypeLabel"
                        :format-date="formatDate"
                      />

                      <!-- Промежуточные точки (если > 2 точек) -->
                      <template v-if="freightRequest.route.points.length > 2">
                        <!-- Кнопка аккордеона — сама является разделителем -->
                        <div class="relative flex items-center">
                          <div class="flex-1 border-t border-border" />
                          <button
                            type="button"
                            class="mx-3 flex items-center gap-2 px-4 py-1.5 rounded-full border border-border bg-background text-sm font-medium text-muted-foreground hover:text-foreground hover:border-foreground/30 transition-colors shrink-0"
                            @click="routeExpanded = !routeExpanded"
                          >
                            {{ routeExpanded ? '↑ Скрыть' : `↓ Ещё ${freightRequest.route.points.length - 2}` }}
                          </button>
                          <div class="flex-1 border-t border-border" />
                        </div>

                        <!-- Анимированный аккордеон через grid -->
                        <div class="accordion-grid" :class="{ 'is-open': routeExpanded }">
                          <div>
                            <div
                              v-for="(point, index) in freightRequest.route.points.slice(1, -1)"
                              :key="index + 1"
                            >
                              <RoutePoint
                                :point="point"
                                :index="index + 1"
                                :get-badge-class="getPointBadgeClass"
                                :get-type-label="getPointTypeLabel"
                                :format-date="formatDate"
                              />
                              <div class="border-t border-border" />
                            </div>
                          </div>
                        </div>
                      </template>

                      <!-- Разделитель перед последней точкой (только если <= 2 точек) -->
                      <div v-if="freightRequest.route.points.length === 2" class="border-t border-border" />

                      <!-- Последняя точка (только если больше одной) -->
                      <RoutePoint
                        v-if="freightRequest.route.points.length > 1"
                        :point="freightRequest.route.points[freightRequest.route.points.length - 1]!"
                        :index="freightRequest.route.points.length - 1"
                        :get-badge-class="getPointBadgeClass"
                        :get-type-label="getPointTypeLabel"
                        :format-date="formatDate"
                      />
                    </div>
                  </TabsContent>

                  <!-- Груз -->
                  <TabsContent value="cargo">
                    <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <dt class="text-sm text-muted-foreground">Описание</dt>
                        <dd class="text-foreground break-words">{{ freightRequest.cargo.description }}</dd>
                      </div>
                      <div>
                        <dt class="text-sm text-muted-foreground">Вес</dt>
                        <dd class="text-foreground">{{ freightRequest.cargo.weight }} кг</dd>
                      </div>
                      <div v-if="freightRequest.cargo.volume">
                        <dt class="text-sm text-muted-foreground">Объём</dt>
                        <dd class="text-foreground">{{ freightRequest.cargo.volume }} м³</dd>
                      </div>
                      <div v-if="freightRequest.cargo.dimensions">
                        <dt class="text-sm text-muted-foreground">Габариты (ДхШхВ)</dt>
                        <dd class="text-foreground">
                          {{ freightRequest.cargo.dimensions.length }} x
                          {{ freightRequest.cargo.dimensions.width }} x
                          {{ freightRequest.cargo.dimensions.height }} м
                        </dd>
                      </div>
                      <div v-if="freightRequest.cargo.quantity">
                        <dt class="text-sm text-muted-foreground">Количество</dt>
                        <dd class="text-foreground">{{ freightRequest.cargo.quantity }} шт</dd>
                      </div>
                      <div v-if="freightRequest.cargo.adr_class && freightRequest.cargo.adr_class !== 'none'">
                        <dt class="text-sm text-muted-foreground">ADR класс</dt>
                        <dd class="text-foreground">{{ adrClassLabels[freightRequest.cargo.adr_class] }}</dd>
                      </div>
                    </dl>
                  </TabsContent>

                  <!-- Требования к транспорту -->
                  <TabsContent value="vehicle">
                    <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <dt class="text-sm text-muted-foreground">Тип транспорта</dt>
                        <dd class="text-foreground">{{ vehicleTypeLabels[freightRequest.vehicle_requirements.vehicle_type] }}</dd>
                      </div>
                      <div>
                        <dt class="text-sm text-muted-foreground">Тип кузова</dt>
                        <dd class="text-foreground">{{ vehicleSubTypeLabels[freightRequest.vehicle_requirements.vehicle_subtype] }}</dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.loading_types?.length">
                        <dt class="text-sm text-muted-foreground">Типы загрузки</dt>
                        <dd class="text-foreground">
                          {{ freightRequest.vehicle_requirements.loading_types.map(t => loadingTypeLabels[t]).join(', ') }}
                        </dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.capacity">
                        <dt class="text-sm text-muted-foreground">Грузоподъёмность</dt>
                        <dd class="text-foreground">{{ freightRequest.vehicle_requirements.capacity }} т</dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.volume">
                        <dt class="text-sm text-muted-foreground">Объём</dt>
                        <dd class="text-foreground">{{ freightRequest.vehicle_requirements.volume }} м³</dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.length">
                        <dt class="text-sm text-muted-foreground">Длина</dt>
                        <dd class="text-foreground">{{ freightRequest.vehicle_requirements.length }} м</dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.width">
                        <dt class="text-sm text-muted-foreground">Ширина</dt>
                        <dd class="text-foreground">{{ freightRequest.vehicle_requirements.width }} м</dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.height">
                        <dt class="text-sm text-muted-foreground">Высота</dt>
                        <dd class="text-foreground">{{ freightRequest.vehicle_requirements.height }} м</dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.temperature">
                        <dt class="text-sm text-muted-foreground">Температурный режим</dt>
                        <dd class="text-foreground">
                          от {{ freightRequest.vehicle_requirements.temperature.min }}°C
                          до {{ freightRequest.vehicle_requirements.temperature.max }}°C
                        </dd>
                      </div>
                      <div v-if="freightRequest.vehicle_requirements.thermograph">
                        <dt class="text-sm text-muted-foreground">Термописец</dt>
                        <dd class="text-foreground">Да</dd>
                      </div>
                    </dl>
                  </TabsContent>

                  <!-- Оплата -->
                  <TabsContent value="payment">
                    <template v-if="freightRequest.payment.price && freightRequest.payment.price.amount > 0">
                      <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        <div>
                          <dt class="text-sm text-muted-foreground">Цена</dt>
                          <dd class="text-xl font-semibold text-success">
                            {{ formatPrice(freightRequest.payment.price.amount, freightRequest.payment.price.currency) }}
                          </dd>
                        </div>
                        <div>
                          <dt class="text-sm text-muted-foreground">НДС</dt>
                          <dd class="text-foreground">{{ vatTypeLabels[freightRequest.payment.vat_type] }}</dd>
                        </div>
                        <div>
                          <dt class="text-sm text-muted-foreground">Способ оплаты</dt>
                          <dd class="text-foreground">{{ paymentMethodLabels[freightRequest.payment.method] }}</dd>
                        </div>
                        <div>
                          <dt class="text-sm text-muted-foreground">Условия оплаты</dt>
                          <dd class="text-foreground">
                            {{ paymentTermsLabels[freightRequest.payment.terms] }}
                            <template v-if="freightRequest.payment.terms === 'deferred' && freightRequest.payment.deferred_days">
                              ({{ freightRequest.payment.deferred_days }} дней)
                            </template>
                          </dd>
                        </div>
                      </dl>
                    </template>
                    <template v-else>
                      <p class="text-muted-foreground">Цена не указана — перевозчики предложат свою</p>
                    </template>
                  </TabsContent>

                  <!-- Комментарий -->
                  <TabsContent v-if="freightRequest.comment" value="comment">
                    <p class="text-foreground break-words">{{ freightRequest.comment }}</p>
                  </TabsContent>
                </Tabs>
              </CardContent>
            </Card>
          </TabsContent>

          <!-- Offers Tab -->
          <TabsContent value="offers" data-tutorial="offers-tab">
            <FreightRequestOffersTab
              :freight-request="freightRequest"
              :offers="offers"
              :action-loading="actionLoading"
              @select="openSelectModal"
              @reject="openRejectModal"
              @withdraw="openWithdrawModal"
              @confirm="openConfirmOfferModal"
              @decline="openDeclineModal"
              @unselect="openUnselectModal"
            />
          </TabsContent>

          <!-- History Tab -->
          <TabsContent value="history">
            <EventHistory :load-fn="loadFreightRequestHistory" />
          </TabsContent>
        </Tabs>
      </div>
    </main>

    <!-- Cancel Dialog -->
    <Dialog v-model:open="showCancelModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отменить заявку</DialogTitle>
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
            {{ actionLoading ? 'Отмена...' : 'Отменить заявку' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Reject Offer Dialog -->
    <Dialog v-model:open="showRejectModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader data-tutorial="reject-offer-modal">
          <DialogTitle>Отклонить предложение</DialogTitle>
          <DialogDescription>
            Укажите причину отклонения (опционально) и подтвердите действие.
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отклонения</Label>
          <Textarea
            v-model="rejectReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showRejectModal = false">
            Отменить
          </Button>
          <Button
            data-tutorial="confirm-reject-btn"
            variant="destructive"
            :disabled="actionLoading"
            @click="confirmRejectOffer"
          >
            {{ actionLoading ? 'Отклонение...' : 'Отклонить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Withdraw Offer Dialog -->
    <Dialog v-model:open="showWithdrawModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отозвать предложение</DialogTitle>
          <DialogDescription>
            Вы уверены, что хотите отозвать своё предложение?
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отзыва</Label>
          <Textarea
            v-model="withdrawReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showWithdrawModal = false">
            Отмена
          </Button>
          <Button
            variant="secondary"
            :disabled="actionLoading"
            @click="confirmWithdrawOffer"
          >
            {{ actionLoading ? 'Отзыв...' : 'Отозвать' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Select Offer Dialog -->
    <Dialog v-model:open="showSelectModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader class="space-y-3">
          <DialogTitle class="text-xl">Выбрать предложение?</DialogTitle>
          <DialogDescription class="text-base">
            После выбора перевозчик получит уведомление и должен будет подтвердить своё участие.
            Если перевозчик подтвердит, все остальные предложения будут автоматически отклонены.
          </DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button variant="outline" @click="showSelectModal = false">
            Отмена
          </Button>
          <Button :disabled="actionLoading" @click="confirmSelectOffer">
            {{ actionLoading ? 'Выбор...' : 'Выбрать' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Confirm Offer Dialog -->
    <Dialog v-model:open="showConfirmOfferModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Подтвердить участие?</DialogTitle>
          <DialogDescription>
            После подтверждения заявка перейдёт в статус "Подтверждена" и вы станете перевозчиком.
            Это действие подразумевает обязательство выполнить перевозку.
          </DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button variant="outline" @click="showConfirmOfferModal = false">
            Отмена
          </Button>
          <Button :disabled="actionLoading" @click="confirmConfirmOffer">
            {{ actionLoading ? 'Подтверждение...' : 'Подтвердить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Decline Offer Dialog -->
    <Dialog v-model:open="showDeclineModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отказаться от предложения?</DialogTitle>
          <DialogDescription>
            Вы отказываетесь от выбранного заказчиком предложения.
            Заказчик сможет выбрать другого перевозчика.
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отказа</Label>
          <Textarea
            v-model="declineReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showDeclineModal = false">
            Отмена
          </Button>
          <Button
            variant="destructive"
            :disabled="actionLoading"
            @click="confirmDeclineOffer"
          >
            {{ actionLoading ? 'Отказ...' : 'Отказаться' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Unselect Offer Dialog -->
    <Dialog v-model:open="showUnselectModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Отменить выбор?</DialogTitle>
          <DialogDescription>
            Предложение вернётся в статус "Ожидает" и перевозчик получит уведомление.
            Вы сможете выбрать это или другое предложение позже.
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label>Причина отмены</Label>
          <Textarea
            v-model="unselectReason"
            rows="3"
            placeholder="Укажите причину (опционально)..."
          />
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showUnselectModal = false">
            Назад
          </Button>
          <Button
            variant="destructive"
            :disabled="actionLoading"
            @click="confirmUnselectOffer"
          >
            {{ actionLoading ? 'Отмена выбора...' : 'Отменить выбор' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Complete Confirmation Dialog -->
    <Dialog v-model:open="showCompleteConfirm">
      <DialogContent class="sm:max-w-md" data-tutorial="complete-confirm-modal">
        <DialogHeader>
          <DialogTitle>Завершить заявку?</DialogTitle>
          <DialogDescription>
            Вы подтверждаете, что заявка выполнена.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showCompleteConfirm = false">
            Отмена
          </Button>
          <Button :disabled="actionLoading" data-tutorial="complete-confirm-btn" @click="handleComplete">
            {{ actionLoading ? 'Завершение...' : 'Завершить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Review Modal -->
    <Dialog v-model:open="showReviewModal">
      <DialogContent class="sm:max-w-md" data-tutorial="review-modal">
        <DialogHeader>
          <DialogTitle>
            {{ myReview ? 'Редактировать отзыв' : 'Оставить отзыв' }}
          </DialogTitle>
          <DialogDescription>
            {{ myReview ? 'Изменить можно в течение 24 часов после создания' : 'Оцените работу с контрагентом' }}
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-4 py-4">
          <div class="space-y-2">
            <Label>Оценка</Label>
            <StarRating v-model="reviewRating" size="lg" data-tutorial="review-rating" />
          </div>

          <div class="space-y-2">
            <Label for="review-comment">Комментарий (необязательно)</Label>
            <Textarea
              id="review-comment"
              v-model="reviewComment"
              placeholder="Опишите ваш опыт работы..."
              rows="3"
              data-tutorial="review-comment"
            />
          </div>

          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
        </div>

        <DialogFooter class="gap-2">
          <Button variant="ghost" data-tutorial="review-skip-btn" @click="showReviewModal = false">
            Пропустить
          </Button>
          <Button :disabled="actionLoading || reviewRating < 1" data-tutorial="review-submit-btn" @click="submitReview">
            {{ actionLoading ? 'Сохранение...' : (myReview ? 'Сохранить' : 'Отправить') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
