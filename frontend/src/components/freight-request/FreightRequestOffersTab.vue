<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import type { Offer, FreightRequest } from '@/types/freightRequest'
import { vatTypeLabels, paymentMethodLabels } from '@/types/freightRequest'
import { offerStatusMap } from '@/constants/statusMaps'
import { formatDateTime, formatMoney } from '@/utils/formatters'

// UI Components
import { Button } from '@/components/ui/button'
import { StatusBadge } from '@/components/shared'
import { Clock, Check, Building2 } from 'lucide-vue-next'

interface Props {
  freightRequest: FreightRequest
  offers: Offer[]
  actionLoading?: boolean
}

interface Emits {
  (e: 'select', offerId: string): void
  (e: 'reject', offerId: string): void
  (e: 'withdraw', offerId: string): void
  (e: 'confirm', offerId: string): void
  (e: 'decline', offerId: string): void
}

const props = withDefaults(defineProps<Props>(), {
  actionLoading: false,
})

const emit = defineEmits<Emits>()

const auth = useAuthStore()
const permissions = usePermissions()

const isOwner = computed(() => {
  return permissions.isFreightRequestOwner(props.freightRequest.customer_org_id)
})

const myOffers = computed(() => {
  return props.offers.filter((o) => o.carrier_org_id === auth.organizationId)
})

const visibleOffers = computed(() => {
  if (isOwner.value) {
    return props.offers
  }
  return myOffers.value
})

const canManageOffers = computed(() => {
  return permissions.canSelectOffer(
    props.freightRequest.customer_org_id,
    props.freightRequest.customer_member_id
  )
})

function formatPrice(amount: number, currency: 'RUB' | 'EUR' | 'USD'): string {
  return formatMoney({ amount, currency })
}
</script>

<template>
  <div class="space-y-6">
    <div v-if="visibleOffers.length === 0" class="text-center py-8 text-muted-foreground">
      Пока нет предложений
    </div>

    <div v-else class="space-y-4">
      <div
        v-for="offer in visibleOffers"
        :key="offer.id"
        :class="[
          'border rounded-lg p-4',
          offer.status === 'selected' ? 'border-info/50 bg-info/5' :
          offer.status === 'confirmed' ? 'border-success/50 bg-success/5' :
          'border-border'
        ]"
      >
        <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-3 mb-2">
              <span class="text-xl font-semibold text-foreground">
                {{ formatPrice(offer.price.amount, offer.price.currency) }}
              </span>
              <StatusBadge :status="offer.status" :status-map="offerStatusMap" />
            </div>
            <div v-if="offer.carrier_org_name || offer.carrier_member_name" class="mb-2 flex flex-wrap items-center gap-x-2">
              <router-link
                v-if="isOwner && offer.carrier_org_name"
                :to="`/organizations/${offer.carrier_org_id}`"
                class="text-primary hover:underline font-medium truncate"
              >
                <Building2 class="inline h-4 w-4 mr-1" />
                {{ offer.carrier_org_name }}
              </router-link>
              <span v-if="isOwner && offer.carrier_org_name && offer.carrier_member_name" class="text-muted-foreground">•</span>
              <router-link
                v-if="offer.carrier_member_name"
                :to="`/members/${offer.carrier_member_id}`"
                class="text-primary hover:underline truncate"
              >
                {{ offer.carrier_member_name }}
              </router-link>
            </div>
            <div class="text-sm text-muted-foreground space-y-1">
              <p>
                <span class="text-muted-foreground/70">НДС:</span>
                {{ vatTypeLabels[offer.vat_type] }}
              </p>
              <p>
                <span class="text-muted-foreground/70">Способ оплаты:</span>
                {{ paymentMethodLabels[offer.payment_method] }}
              </p>
              <p v-if="offer.comment" class="break-words">
                <span class="text-muted-foreground/70">Комментарий:</span>
                {{ offer.comment }}
              </p>
              <p class="text-xs flex items-center gap-1">
                <Clock class="h-3 w-3" />
                {{ formatDateTime(offer.created_at) }}
              </p>
            </div>
          </div>

          <!-- Owner/Admin actions -->
          <div v-if="canManageOffers && offer.status === 'pending' && freightRequest.status === 'published'" class="flex gap-2 shrink-0">
            <Button
              size="sm"
              :disabled="actionLoading"
              @click="emit('select', offer.id)"
            >
              <Check class="mr-1 h-4 w-4" />
              Выбрать
            </Button>
            <Button
              variant="outline"
              size="sm"
              :disabled="actionLoading"
              @click="emit('reject', offer.id)"
            >
              Отклонить
            </Button>
          </div>

          <!-- Carrier actions (own offer) -->
          <div v-if="!isOwner && offer.carrier_org_id === auth.organizationId" class="flex gap-2 shrink-0">
            <template v-if="offer.status === 'pending' && permissions.canWithdrawOffer(offer.carrier_org_id, offer.carrier_member_id)">
              <Button
                variant="outline"
                size="sm"
                :disabled="actionLoading"
                @click="emit('withdraw', offer.id)"
              >
                Отозвать
              </Button>
            </template>
            <template v-if="offer.status === 'selected' && permissions.canConfirmOffer(offer.carrier_org_id, offer.carrier_member_id)">
              <Button
                size="sm"
                :disabled="actionLoading"
                @click="emit('confirm', offer.id)"
              >
                <Check class="mr-1 h-4 w-4" />
                Подтвердить
              </Button>
              <Button
                variant="outline"
                size="sm"
                :disabled="actionLoading"
                @click="emit('decline', offer.id)"
              >
                Отказаться
              </Button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
