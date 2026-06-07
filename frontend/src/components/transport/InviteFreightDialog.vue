<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { freightRequestsApi } from '@/api/freightRequests'
import { ApiError } from '@/api/errors'
import type { FreightRequestListItem, Currency } from '@/types/freightRequest'
import { currencyLabels } from '@/types/freightRequest'
import type { VehicleListItem } from '@/types/vehicle'
import { useToast } from '@/components/ui/toast/use-toast'
import { logger } from '@/utils/logger'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { LoadingSpinner, EmptyState, ErrorBanner } from '@/components/shared'
import { FileText, MapPin, Send } from 'lucide-vue-next'

const props = defineProps<{
  vehicle: VehicleListItem | null
}>()

const open = defineModel<boolean>('open', { required: true })

const router = useRouter()
const auth = useAuthStore()
const { toast } = useToast()

const requests = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const submittingId = ref<string | null>(null)
// Заявки, уже предложенные этой организации (в рамках открытой модалки)
const invitedIds = ref<Set<string>>(new Set())

watch(open, (isOpen) => {
  if (isOpen) {
    invitedIds.value = new Set()
    loadRequests()
  }
})

async function loadRequests() {
  if (!auth.organizationId) return
  isLoading.value = true
  error.value = null
  try {
    const res = await freightRequestsApi.list({
      customer_org_id: auth.organizationId,
      statuses: 'published',
      sort_by: 'created_at_desc',
      limit: 50,
    })
    requests.value = res.items ?? []
  } catch (e) {
    error.value = 'Не удалось загрузить ваши заявки'
    logger.error('Failed to load own freight requests for invite', e)
  } finally {
    isLoading.value = false
  }
}

function formatRoute(fr: FreightRequestListItem): string {
  const from = fr.origin_address ?? '—'
  const to = fr.destination_address ?? '—'
  return `${from} → ${to}`
}

function formatPrice(fr: FreightRequestListItem): string | null {
  if (!fr.price_amount) return null
  const value = (fr.price_amount / 100).toLocaleString('ru-RU')
  const symbol = fr.price_currency ? currencyLabels[fr.price_currency as Currency] ?? fr.price_currency : '₽'
  return `${value} ${symbol}`
}

function inviteErrorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.status === 409 && e.message.includes('already been invited')) {
      return 'Этой организации уже предложена данная заявка'
    }
    if (e.status === 409 && e.message.includes('expired')) {
      return 'Срок действия заявки истёк'
    }
    if (e.status === 409 && e.message.includes('not published')) {
      return 'Заявка не опубликована'
    }
    if (e.status === 422 && e.message.includes('limit')) {
      return 'Достигнут лимит приглашений по этой заявке'
    }
    if (e.status === 422 && e.message.includes('not verified')) {
      return 'Транспорт не прошёл подтверждение'
    }
    if (e.status === 400 && e.message.includes('own organization')) {
      return 'Нельзя предложить заявку своей организации'
    }
  }
  return 'Не удалось предложить заявку'
}

async function invite(fr: FreightRequestListItem) {
  if (!props.vehicle || submittingId.value) return
  submittingId.value = fr.id
  try {
    await freightRequestsApi.inviteCarrier(fr.id, props.vehicle.id)
    invitedIds.value.add(fr.id)
    toast({
      title: 'Заявка предложена перевозчику',
      description: props.vehicle.org_name
        ? `Организация «${props.vehicle.org_name}» получит уведомление`
        : undefined,
      variant: 'success',
    })
  } catch (e) {
    logger.error('Failed to invite carrier', e)
    toast({ title: inviteErrorMessage(e), variant: 'destructive' })
  } finally {
    submittingId.value = null
  }
}

function goToNewRequest() {
  open.value = false
  router.push('/freight-requests/new')
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>Предложить заявку</DialogTitle>
        <DialogDescription>
          <template v-if="vehicle">
            {{ vehicle.registration_number }}<template v-if="vehicle.org_name"> · {{ vehicle.org_name }}</template>
            — выберите заявку, которую хотите предложить
          </template>
        </DialogDescription>
      </DialogHeader>

      <LoadingSpinner v-if="isLoading" text="Загрузка заявок..." />

      <ErrorBanner v-else-if="error" :message="error" @retry="loadRequests" />

      <EmptyState
        v-else-if="requests.length === 0"
        :icon="FileText"
        title="Нет опубликованных заявок"
        description="Создайте заявку на перевозку, чтобы предложить её перевозчику"
        action-label="Новая заявка"
        @action="goToNewRequest"
      />

      <div v-else class="space-y-2 max-h-80 overflow-y-auto pr-1">
        <div
          v-for="fr in requests"
          :key="fr.id"
          class="flex items-center justify-between gap-3 rounded-md border p-3"
        >
          <div class="min-w-0">
            <div class="text-sm font-medium">Заявка #{{ fr.request_number }}</div>
            <div class="flex items-center gap-1 text-xs text-muted-foreground truncate">
              <MapPin class="h-3 w-3 shrink-0" />
              <span class="truncate">{{ formatRoute(fr) }}</span>
            </div>
            <div v-if="formatPrice(fr)" class="text-xs text-muted-foreground mt-0.5">
              {{ formatPrice(fr) }}
            </div>
          </div>
          <Button
            size="sm"
            class="shrink-0"
            :disabled="submittingId === fr.id || invitedIds.has(fr.id)"
            @click="invite(fr)"
          >
            <Send class="h-3.5 w-3.5 mr-1.5" />
            {{ invitedIds.has(fr.id) ? 'Предложено' : submittingId === fr.id ? 'Отправка...' : 'Предложить' }}
          </Button>
        </div>
      </div>

      <DialogFooter>
        <Button variant="ghost" @click="open = false">Закрыть</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
