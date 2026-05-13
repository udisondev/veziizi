<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { offersApi, type MyOfferListItem } from '@/api/offers'
import type { OfferStatus, OfferStatusFilter } from '@/types/freightRequest'
import {
  offerStatusOptions,
  currencyLabels,
  type Currency,
} from '@/types/freightRequest'
import { logger } from '@/utils/logger'
import { useAsyncList } from '@/composables/useAsyncData'
import { useNotificationsStore } from '@/stores/notifications'
import { getCategoryByType } from '@/types/notification'

// UI Components
import { TabsSlider } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { SelectField } from '@/components/ui/select-field'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// Shared Components
import {
  PageHeader,
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
} from '@/components/shared'

// Icons
import {
  HandCoins,
  ArrowRight,
  Package,
  Calendar,
  AlertCircle,
  Check,
  XCircle,
  Undo2,
} from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()

// Filters
const ownershipFilter = ref<'org' | 'my'>('org')
const statusFilter = ref<OfferStatusFilter>('all')

const ownershipItems = [
  { value: 'org', label: 'Организации' },
  { value: 'my', label: 'Мои' },
]

const {
  data: items,
  isLoading,
  error,
  execute: loadItems,
} = useAsyncList<MyOfferListItem>(
  () => {
    return offersApi.listMy({
      status: statusFilter.value !== 'all' ? statusFilter.value as OfferStatus : undefined,
      member_id: ownershipFilter.value === 'my' ? auth.memberId ?? undefined : undefined,
    })
  },
  { immediate: true }
)

// Status map for StatusBadge
const offerStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  pending: { label: 'Ожидает', variant: 'info' },
  selected: { label: 'Выбран', variant: 'warning' },
  confirmed: { label: 'Подтверждён', variant: 'success' },
  declined: { label: 'Отклонён', variant: 'destructive' },
  withdrawn: { label: 'Отозван', variant: 'secondary' },
  rejected: { label: 'Отклонён', variant: 'destructive' },
}

// Action state
const actionLoading = ref<string | null>(null)
const actionError = ref<string | null>(null)

// Confirm modal
const showConfirmModal = ref(false)
const confirmAction = ref<{ type: 'withdraw' | 'confirm' | 'decline'; item: MyOfferListItem } | null>(null)
const declineReason = ref('')


function goToFreightRequest(frId: string, tab: 'offers' | 'details' = 'offers') {
  router.push(`/freight-requests/${frId}?tab=${tab}`)
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function formatPrice(amount?: number, currency?: string): string {
  if (!amount || !currency) return '—'
  const formatted = (amount / 100).toLocaleString('ru-RU', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  })
  const symbol = currencyLabels[currency as Currency] || currency
  return `${formatted} ${symbol}`
}

function formatWeight(weight?: number): string {
  if (!weight) return '—'
  return `${weight.toLocaleString('ru-RU')} кг`
}

function formatRoute(item: MyOfferListItem): string {
  const origin = item.origin_address || '?'
  const dest = item.destination_address || '?'
  return `${origin} → ${dest}`
}

// Action handlers
function canWithdraw(status: OfferStatus): boolean {
  return status === 'pending'
}

function canConfirm(status: OfferStatus): boolean {
  return status === 'selected'
}

function canDecline(status: OfferStatus): boolean {
  return status === 'selected'
}

function openConfirmModal(type: 'withdraw' | 'confirm' | 'decline', item: MyOfferListItem) {
  confirmAction.value = { type, item }
  declineReason.value = ''
  actionError.value = null
  showConfirmModal.value = true
}

function closeConfirmModal() {
  showConfirmModal.value = false
  confirmAction.value = null
  declineReason.value = ''
}

const confirmModalTitle = computed(() => {
  if (!confirmAction.value) return ''
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'Отозвать оффер?'
    case 'confirm':
      return 'Подтвердить оффер?'
    case 'decline':
      return 'Отказаться от оффера?'
    default:
      return ''
  }
})

const confirmModalDescription = computed(() => {
  if (!confirmAction.value) return ''
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'Ваш оффер будет отозван и больше не будет виден заказчику.'
    case 'confirm':
      return 'После подтверждения будет создан заказ и вы станете перевозчиком.'
    case 'decline':
      return 'Вы отказываетесь от выбранного оффера. Заказчик сможет выбрать другого перевозчика.'
    default:
      return ''
  }
})

const confirmModalButtonText = computed(() => {
  if (!confirmAction.value) return ''
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'Отозвать'
    case 'confirm':
      return 'Подтвердить'
    case 'decline':
      return 'Отказаться'
    default:
      return ''
  }
})

const confirmModalButtonVariant = computed<'default' | 'destructive' | 'secondary'>(() => {
  if (!confirmAction.value) return 'default'
  switch (confirmAction.value.type) {
    case 'withdraw':
      return 'secondary'
    case 'confirm':
      return 'default'
    case 'decline':
      return 'destructive'
    default:
      return 'default'
  }
})

async function executeAction() {
  if (!confirmAction.value) return

  const { type, item } = confirmAction.value
  actionLoading.value = item.id
  actionError.value = null

  try {
    switch (type) {
      case 'withdraw':
        await offersApi.withdraw(item.freight_request_id, item.id)
        break
      case 'confirm':
        await offersApi.confirm(item.freight_request_id, item.id)
        break
      case 'decline':
        await offersApi.decline(item.freight_request_id, item.id, declineReason.value || undefined)
        break
    }
    closeConfirmModal()
    await loadItems()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Не удалось выполнить действие'
    logger.error('Failed to perform offer action', e)
  } finally {
    actionLoading.value = null
  }
}

// Watch filters
watch([statusFilter, ownershipFilter], () => {
  loadItems()
})

// Обновляем список когда приходит уведомление об изменении оффера
const notificationsStore = useNotificationsStore()
const lastOfferNotificationId = computed(() =>
  notificationsStore.notifications.find(
    n => getCategoryByType(n.notification_type) === 'offers'
  )?.id ?? null
)
watch(lastOfferNotificationId, (newId, oldId) => {
  if (newId && newId !== oldId) loadItems()
})

</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <PageHeader title="Предложения" class="mb-6" />

    <!-- Filters -->
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center gap-3">
      <TabsSlider v-model="ownershipFilter" :items="ownershipItems" />
      <div class="w-full sm:w-48">
        <SelectField
          v-model="statusFilter"
          :options="offerStatusOptions"
          sheet-label="Статус"
        />
      </div>
    </div>

    <!-- Action error -->
    <div
      v-if="actionError && !showConfirmModal"
      class="mb-6 flex items-center justify-between gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
    >
      <div class="flex items-center gap-2">
        <AlertCircle class="h-4 w-4 shrink-0" />
        {{ actionError }}
      </div>
      <Button variant="ghost" size="sm" @click="actionError = null">
        Закрыть
      </Button>
    </div>

    <!-- Loading -->
    <LoadingSpinner v-if="isLoading" text="Загрузка предложений..." />

    <!-- Error -->
    <ErrorBanner
      v-else-if="error"
      :message="error"
      @retry="loadItems"
    />

    <!-- Empty state -->
    <EmptyState
      v-else-if="(items ?? []).length === 0"
      :icon="HandCoins"
      title="Предложений пока нет"
      description="Вы ещё не делали предложений на заявки"
    >
      <template #action>
        <Button as-child>
          <router-link to="/requests">
            <Package class="mr-2 h-4 w-4" />
            Найти заявки
          </router-link>
        </Button>
      </template>
    </EmptyState>

    <!-- List -->
    <div v-else class="space-y-4">
      <Card
        v-for="item in (items ?? [])"
        :key="item.id"
        interactive
      >
        <CardContent class="p-4">
          <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <!-- Main info -->
            <div
              class="flex-1 cursor-pointer"
              @click="goToFreightRequest(item.freight_request_id)"
            >
              <!-- Status -->
              <div class="flex items-center gap-2 mb-2">
                <StatusBadge :status="item.status" :status-map="offerStatusMap" />
              </div>

              <!-- Route -->
              <div class="text-sm font-medium text-foreground mb-1 break-words">
                {{ formatRoute(item) }}
              </div>

              <!-- Details -->
              <div class="flex flex-wrap gap-4 text-sm text-muted-foreground">
                <span class="flex items-center gap-1">
                  <Package class="h-3.5 w-3.5" />
                  {{ formatWeight(item.cargo_weight) }}
                </span>
                <span>Ставка: {{ formatPrice(item.price_amount, item.price_currency) }}</span>
                <span class="flex items-center gap-1">
                  <Calendar class="h-3.5 w-3.5" />
                  {{ formatDate(item.created_at) }}
                </span>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex gap-2 flex-shrink-0">
              <!-- Withdraw (pending) -->
              <Button
                v-if="canWithdraw(item.status)"
                variant="secondary"
                size="sm"
                :disabled="actionLoading === item.id"
                @click.stop="openConfirmModal('withdraw', item)"
              >
                <Undo2 class="mr-1 h-4 w-4" />
                Отозвать
              </Button>

              <!-- Confirm (selected) -->
              <Button
                v-if="canConfirm(item.status)"
                size="sm"
                :disabled="actionLoading === item.id"
                @click.stop="openConfirmModal('confirm', item)"
              >
                <Check class="mr-1 h-4 w-4" />
                Подтвердить
              </Button>

              <!-- Decline (selected) -->
              <Button
                v-if="canDecline(item.status)"
                variant="destructive"
                size="sm"
                :disabled="actionLoading === item.id"
                @click.stop="openConfirmModal('decline', item)"
              >
                <XCircle class="mr-1 h-4 w-4" />
                Отказаться
              </Button>

              <!-- View -->
              <Button
                variant="outline"
                size="sm"
                @click.stop="goToFreightRequest(item.freight_request_id, 'details')"
              >
                К заявке
                <ArrowRight class="ml-1 h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Confirm Modal -->
    <Dialog v-model:open="showConfirmModal">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ confirmModalTitle }}</DialogTitle>
          <DialogDescription>
            {{ confirmModalDescription }}
          </DialogDescription>
        </DialogHeader>

        <!-- Decline reason -->
        <div v-if="confirmAction?.type === 'decline'" class="space-y-2">
          <Label>Причина отказа (необязательно)</Label>
          <Textarea
            v-model="declineReason"
            rows="2"
            placeholder="Укажите причину..."
          />
        </div>

        <!-- Action error in modal -->
        <div
          v-if="actionError"
          class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
        >
          <AlertCircle class="h-4 w-4 shrink-0" />
          {{ actionError }}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            :disabled="actionLoading !== null"
            @click="closeConfirmModal"
          >
            Отмена
          </Button>
          <Button
            :variant="confirmModalButtonVariant"
            :disabled="actionLoading !== null"
            @click="executeAction"
          >
            <template v-if="actionLoading">
              Выполнение...
            </template>
            <template v-else>
              {{ confirmModalButtonText }}
            </template>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
