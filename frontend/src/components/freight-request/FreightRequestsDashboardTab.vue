<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import { freightRequestsApi } from '@/api/freightRequests'
import { offersApi, type MyOfferListItem } from '@/api/offers'
import { organizationsApi } from '@/api/organizations'
import type { FreightRequestListItem, FreightRequestStatus, OwnershipFilter } from '@/types/freightRequest'
import type { OrganizationStats, DashboardStats, OrganizationRating, PendingOfferItem } from '@/types/admin'
import { currencyLabels } from '@/types/freightRequest'
import { formatRelativeTime } from '@/utils/formatters'
import { logger } from '@/utils/logger'
import { Button } from '@/components/ui/button'
import { LoadingSpinner } from '@/components/shared'
import StarRating from '@/components/freight-request/StarRating.vue'
import { getMyTickets, type TicketListItem } from '@/api/support'
import { ChevronRight, TrendingUp, Star, Headset, HandCoins } from 'lucide-vue-next'

const emit = defineEmits<{
  'go-to-new': []
  'go-to-list': [filtered?: boolean]
}>()

const router = useRouter()
const auth = useAuthStore()
const filtersStore = useFreightFiltersStore()

function goToListFiltered(statuses: FreightRequestStatus[], ownership: OwnershipFilter) {
  filtersStore.setFilters({ statuses, ownership })
  emit('go-to-list', true)
}

function goToPendingOffers() {
  filtersStore.setFilters({ ownership: 'my_org', hasPendingOffers: true })
  emit('go-to-list', true)
}

const orgStats = ref<OrganizationStats | null>(null)
const dashboardStats = ref<DashboardStats | null>(null)
const orgRating = ref<OrganizationRating | null>(null)
const recentItems = ref<FreightRequestListItem[]>([])
const selectedOffers = ref<MyOfferListItem[]>([])
const myActiveRequests = ref<FreightRequestListItem[]>([])
const pendingOffersOnMyRequests = ref<PendingOfferItem[]>([])
const openTickets = ref<TicketListItem[]>([])

const isLoadingStats = ref(false)
const isLoadingRecent = ref(false)
const isLoadingAttention = ref(false)

const partiallyCompletedRequests = computed(() =>
  myActiveRequests.value.filter(i => i.status === 'partially_completed')
)

const expiringSoonRequests = computed(() =>
  myActiveRequests.value.filter(i => {
    if (i.status !== 'published') return false
    const diff = (new Date(i.expires_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
    return diff > 0 && diff <= 3
  })
)

const hasAttentionItems = computed(
  () => selectedOffers.value.length + partiallyCompletedRequests.value.length + expiringSoonRequests.value.length > 0
)

const ratingValue = computed(() => orgRating.value?.average_rating ?? 0)
const hasRating = computed(() => (orgRating.value?.total_reviews ?? 0) > 0)

const hasCarrierStats = computed(() => (orgStats.value?.total_offers_made ?? 0) > 0)
const offerSuccessRate = computed(() => {
  const total = orgStats.value?.total_offers_made ?? 0
  const success = orgStats.value?.successful_offers ?? 0
  if (total === 0) return null
  return Math.round((success / total) * 100)
})

const hasCustomerActivity = computed(() =>
  !!dashboardStats.value && (dashboardStats.value.as_customer_published + dashboardStats.value.as_customer_selected + dashboardStats.value.as_customer_confirmed) > 0
)

const hasCarrierActivity = computed(() =>
  !!dashboardStats.value && (dashboardStats.value.as_carrier_confirmed + dashboardStats.value.as_carrier_partially_completed) > 0
)

const activeAsCustomer = computed(() =>
  (dashboardStats.value?.as_customer_published ?? 0) +
  (dashboardStats.value?.as_customer_selected ?? 0) +
  (dashboardStats.value?.as_customer_confirmed ?? 0)
)

const pipelineStages = computed(() => {
  const s = dashboardStats.value
  const stages: { label: string; value: number; color: string; statuses: FreightRequestStatus[]; ownership: OwnershipFilter }[] = [
    { label: 'Ищут перевозчика',  value: s?.as_customer_published ?? 0,          color: 'text-primary',      statuses: ['published'],           ownership: 'my_org' },
    { label: 'Перевозчик выбран', value: s?.as_customer_selected ?? 0,            color: 'text-amber-500',    statuses: ['selected'],            ownership: 'my_org' },
    { label: 'В пути',            value: s?.as_customer_confirmed ?? 0,           color: 'text-emerald-500',  statuses: ['confirmed'],           ownership: 'my_org' },
  ]
  if (hasCarrierActivity.value) {
    stages.push({ label: 'Везу сейчас',     value: s?.as_carrier_confirmed ?? 0,          color: 'text-emerald-500', statuses: ['confirmed'],           ownership: 'my_as_carrier' })
    stages.push({ label: 'Нужно завершить', value: s?.as_carrier_partially_completed ?? 0, color: 'text-amber-500',  statuses: ['partially_completed'], ownership: 'my_as_carrier' })
  }
  stages.push({ label: 'Истекают скоро', value: expiringSoonRequests.value.length, color: 'text-destructive', statuses: ['published'], ownership: 'my_org' })
  return stages
})

const pipelineGridClass = computed(() =>
  pipelineStages.value.length === 6 ? 'sm:grid-cols-6' : 'sm:grid-cols-4'
)


const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 12) return 'Доброе утро'
  if (h < 18) return 'Добрый день'
  return 'Добрый вечер'
})

const formattedDate = computed(() =>
  new Date().toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
)

function fulfilled<T>(label: string, r: PromiseSettledResult<T>): T | null {
  if (r.status === 'fulfilled') return r.value
  logger.error(`Dashboard: failed to load ${label}`, r.reason)
  return null
}

async function loadData() {
  if (!auth.organizationId) return
  const orgID = auth.organizationId
  isLoadingStats.value = true
  isLoadingRecent.value = true
  isLoadingAttention.value = true
  try {
    const [statsRes, dashboardRes, ratingRes, recentRes, selectedOffersRes, myActiveRes, pendingOffersRes, ticketsRes] = await Promise.allSettled([
      organizationsApi.getStats(orgID),
      organizationsApi.getDashboardStats(orgID),
      organizationsApi.getRating(orgID),
      freightRequestsApi.list({ statuses: 'published', limit: 10 }),
      offersApi.listMy({ status: 'selected', limit: 10 }),
      freightRequestsApi.list({
        customer_org_id: orgID,
        statuses: 'published,partially_completed',
        limit: 50,
      }),
      organizationsApi.getPendingOffers(orgID, 15),
      getMyTickets({ status: 'open' }),
    ])
    orgStats.value = fulfilled('stats', statsRes)
    dashboardStats.value = fulfilled('dashboard-stats', dashboardRes)
    orgRating.value = fulfilled('rating', ratingRes)
    recentItems.value = fulfilled('recent', recentRes)?.items ?? []
    selectedOffers.value = fulfilled('selected-offers', selectedOffersRes) ?? []
    myActiveRequests.value = fulfilled('my-active', myActiveRes)?.items ?? []
    pendingOffersOnMyRequests.value = fulfilled('pending-offers', pendingOffersRes) ?? []
    openTickets.value = fulfilled('tickets', ticketsRes) ?? []
  } finally {
    isLoadingStats.value = false
    isLoadingRecent.value = false
    isLoadingAttention.value = false
  }
}

function formatPrice(amount?: number | null, currency?: string | null): string {
  if (!amount || !currency) return ''
  const symbol = currencyLabels[currency as keyof typeof currencyLabels] || currency
  return `${(amount / 100).toLocaleString('ru-RU')} ${symbol}`
}

function daysUntilExpiry(expiresAt: string): number {
  return Math.ceil((new Date(expiresAt).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-6">

    <!-- Header -->
    <div>
      <p class="text-sm text-muted-foreground mb-1">{{ formattedDate }}</p>
      <h1 class="text-2xl font-bold text-foreground">{{ greeting }}, {{ auth.name }}</h1>
    </div>

    <!-- KPI cards -->
    <div v-if="isLoadingStats" class="rounded-lg border-t-[3px] border border-t-muted bg-card p-5 space-y-4">
      <div class="flex justify-between">
        <div class="h-3 bg-muted animate-pulse rounded w-28" />
        <div class="h-3 bg-muted animate-pulse rounded w-12" />
      </div>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-px bg-border">
        <div v-for="j in 4" :key="j" class="bg-card px-3 py-2.5 flex flex-col items-center gap-2">
          <div class="h-6 bg-muted animate-pulse rounded w-6" />
          <div class="h-2.5 bg-muted animate-pulse rounded w-16" />
        </div>
      </div>
    </div>

    <div v-else>

      <!-- Мои активные заявки — полная ширина -->
      <div class="rounded-lg border-t-[3px] border border-t-primary bg-card p-5 text-left">
        <div class="flex items-center justify-between mb-4">
          <span class="text-xs text-muted-foreground">Мои активные заявки</span>
          <span class="text-xs font-semibold text-muted-foreground">{{ activeAsCustomer }} всего</span>
        </div>

        <!-- Пайплайн -->
        <div
          v-if="hasCustomerActivity || hasCarrierActivity"
          class="grid grid-cols-2 gap-px bg-border"
          :class="pipelineGridClass"
        >
          <button
            v-for="stage in pipelineStages"
            :key="stage.label"
            class="bg-card px-2 py-3 sm:px-3 sm:py-2.5 flex flex-col items-center gap-0.5 transition-colors"
            :class="stage.value > 0 ? 'hover:bg-muted/60 cursor-pointer' : 'cursor-default'"
            :disabled="stage.value === 0"
            @click.stop="stage.value > 0 && goToListFiltered(stage.statuses, stage.ownership)"
          >
            <span
              class="text-xl sm:text-2xl font-bold leading-none"
              :class="stage.value > 0 ? stage.color : 'text-muted-foreground/30'"
            >{{ stage.value }}</span>
            <span class="text-[10px] sm:text-[11px] text-muted-foreground text-center leading-tight mt-1">{{ stage.label }}</span>
          </button>
        </div>

        <div v-else class="text-sm text-muted-foreground">
          Нет активных заявок
        </div>
      </div>

    </div>

    <!-- Action cards (urgent attention) -->
    <div v-if="!isLoadingAttention && hasAttentionItems" class="space-y-2">

      <div
        v-for="offer in selectedOffers"
        :key="offer.id"
        class="flex items-center gap-4 rounded-lg border bg-card px-4 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
        @click="router.push(`/freight-requests/${offer.freight_request_id}?tab=offers`)"
      >
        <div class="w-1 self-stretch rounded-full bg-primary shrink-0" />
        <div class="flex-1 min-w-0">
          <div class="text-sm font-semibold text-foreground">Ваше предложение выбрано</div>
          <div class="text-xs text-muted-foreground mt-0.5 truncate">
            {{ offer.origin_address }} → {{ offer.destination_address }}
            <template v-if="offer.price_amount"> · {{ formatPrice(offer.price_amount, offer.price_currency) }}</template>
          </div>
        </div>
        <Button size="sm" variant="outline" class="shrink-0">Подтвердить</Button>
      </div>

      <div
        v-for="item in partiallyCompletedRequests"
        :key="item.id"
        class="flex items-center gap-4 rounded-lg border bg-card px-4 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
        @click="router.push(`/freight-requests/${item.id}`)"
      >
        <div class="w-1 self-stretch rounded-full bg-amber-500 shrink-0" />
        <div class="flex-1 min-w-0">
          <div class="text-sm font-semibold text-foreground">Подтвердите завершение</div>
          <div class="text-xs text-muted-foreground mt-0.5 truncate">
            #{{ item.request_number }} · {{ item.origin_address }} → {{ item.destination_address }}
          </div>
        </div>
        <Button size="sm" variant="outline" class="shrink-0">Открыть</Button>
      </div>

      <div
        v-for="item in expiringSoonRequests"
        :key="item.id"
        class="flex items-center gap-4 rounded-lg border bg-card px-4 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
        @click="router.push(`/freight-requests/${item.id}`)"
      >
        <div class="w-1 self-stretch rounded-full bg-destructive shrink-0" />
        <div class="flex-1 min-w-0">
          <div class="text-sm font-semibold text-foreground">
            Заявка истекает через {{ daysUntilExpiry(item.expires_at) }} {{ daysUntilExpiry(item.expires_at) === 1 ? 'день' : 'дня' }}
          </div>
          <div class="text-xs text-muted-foreground mt-0.5 truncate">
            #{{ item.request_number }} · {{ item.origin_address }} → {{ item.destination_address }}
          </div>
        </div>
        <Button size="sm" variant="outline" class="shrink-0">Посмотреть</Button>
      </div>

    </div>

    <!-- Main: offers feed + sidebar -->
    <div class="grid grid-cols-1 lg:grid-cols-[1fr_260px] gap-6 items-start">

      <!-- Офферы на мои заявки / Новые заявки на рынке -->
      <div class="space-y-4">

        <!-- Офферы на мои заявки -->
        <div class="rounded-lg border bg-card overflow-hidden">
          <div class="flex items-center justify-between px-5 py-4 border-b">
            <div class="flex items-center gap-2.5">
              <h2 class="text-sm font-semibold text-foreground">Офферы на мои заявки</h2>
              <span
                v-if="(dashboardStats?.pending_offers_count ?? 0) > 0"
                class="text-xs font-semibold px-1.5 py-0.5 rounded-full bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400"
              >{{ dashboardStats!.pending_offers_count }}</span>
              <span
                v-if="(dashboardStats?.pending_offers_today ?? 0) > 0"
                class="flex items-center gap-0.5 text-xs font-medium text-emerald-600 dark:text-emerald-400"
              >
                <TrendingUp class="h-3 w-3" />
                +{{ dashboardStats!.pending_offers_today }} за сутки
              </span>
            </div>
            <button
              class="flex items-center gap-1 text-sm text-primary hover:text-primary/80 transition-colors"
              @click="goToPendingOffers()"
            >
              Все <ChevronRight class="h-4 w-4" />
            </button>
          </div>

          <div v-if="isLoadingStats" class="px-5 py-10 flex justify-center">
            <LoadingSpinner text="Загрузка..." />
          </div>

          <div v-else-if="pendingOffersOnMyRequests.length === 0" class="px-5 py-10 text-center text-sm text-muted-foreground">
            Нет входящих предложений
          </div>

          <div v-else>
            <div
              v-for="(offer, index) in pendingOffersOnMyRequests"
              :key="offer.id"
              class="flex items-center gap-3 px-5 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
              :class="index < pendingOffersOnMyRequests.length - 1 ? 'border-b' : ''"
              @click="router.push(`/freight-requests/${offer.freight_request_id}?tab=offers`)"
            >
              <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-1.5 text-sm">
                  <span class="text-xs text-muted-foreground shrink-0">#{{ offer.request_number }}</span>
                  <span class="font-medium text-foreground truncate">
                    {{ offer.origin_address || '—' }} → {{ offer.destination_address || '—' }}
                  </span>
                </div>
                <div class="flex items-center gap-1 text-xs text-muted-foreground mt-0.5">
                  <span class="font-medium text-foreground/70">{{ offer.carrier_org_name || 'Перевозчик' }}</span>
                  <span>·</span>
                  <span>{{ formatRelativeTime(offer.created_at) }}</span>
                  <template v-if="offer.offers_count > 1">
                    <span>·</span>
                    <span class="text-primary font-medium">{{ offer.offers_count }} предложения</span>
                  </template>
                </div>
              </div>
              <div class="text-sm font-semibold text-foreground shrink-0">
                {{ formatPrice(offer.price_amount, offer.price_currency) || '—' }}
              </div>
              <ChevronRight class="h-4 w-4 text-muted-foreground shrink-0" />
            </div>
          </div>
        </div>

        <!-- Новые заявки на рынке -->
        <div class="rounded-lg border bg-card overflow-hidden">
          <div class="flex items-center justify-between px-5 py-4 border-b">
            <div class="flex items-center gap-2.5">
              <h2 class="text-sm font-semibold text-foreground">Новые заявки на рынке</h2>
              <span
                v-if="(dashboardStats?.market_published_today ?? 0) > 0"
                class="text-xs font-semibold px-1.5 py-0.5 rounded-full bg-primary/10 text-primary"
              >{{ dashboardStats!.market_published_today }}</span>
              <span
                v-if="(dashboardStats?.market_published_today ?? 0) > 0"
                class="flex items-center gap-0.5 text-xs font-medium text-emerald-600 dark:text-emerald-400"
              >
                <TrendingUp class="h-3 w-3" />
                за сутки
              </span>
            </div>
            <button
              class="flex items-center gap-1 text-sm text-primary hover:text-primary/80 transition-colors"
              @click="emit('go-to-list')"
            >
              Все <ChevronRight class="h-4 w-4" />
            </button>
          </div>

          <div v-if="isLoadingRecent" class="px-5 py-10 flex justify-center">
            <LoadingSpinner text="Загрузка..." />
          </div>

          <div v-else-if="recentItems.length === 0" class="px-5 py-10 text-center text-sm text-muted-foreground">
            Нет опубликованных заявок
          </div>

          <div v-else>
            <div
              v-for="(item, index) in recentItems"
              :key="item.id"
              class="flex items-center gap-3 px-5 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
              :class="index < recentItems.length - 1 ? 'border-b' : ''"
              @click="router.push(`/freight-requests/${item.id}`)"
            >
              <div class="w-1.5 h-1.5 rounded-full bg-muted-foreground/40 shrink-0" />
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-1.5 text-sm">
                  <span class="text-xs text-muted-foreground shrink-0">#{{ item.request_number }}</span>
                  <span class="font-medium text-foreground truncate">
                    {{ item.origin_address || '—' }} → {{ item.destination_address || '—' }}
                  </span>
                </div>
                <div v-if="item.customer_org_name" class="flex items-center gap-1 text-xs text-muted-foreground mt-0.5">
                  <span>{{ item.customer_org_name }}</span>
                  <span>·</span>
                  <span>{{ formatRelativeTime(item.created_at) }}</span>
                </div>
              </div>
              <div class="text-sm font-semibold text-foreground shrink-0">
                {{ formatPrice(item.price_amount, item.price_currency) || '—' }}
              </div>
              <ChevronRight class="h-4 w-4 text-muted-foreground shrink-0" />
            </div>
          </div>
        </div>

      </div>

      <!-- Sidebar -->
      <div class="space-y-3">

        <!-- Skeleton -->
        <div v-if="isLoadingStats" class="rounded-lg border bg-card p-4 space-y-3">
          <div class="h-3 bg-muted animate-pulse rounded w-20" />
          <div class="h-4 bg-muted animate-pulse rounded w-32" />
          <div class="h-4 bg-muted animate-pulse rounded w-28" />
          <div class="h-4 bg-muted animate-pulse rounded w-24" />
        </div>

        <!-- Репутация -->
        <div v-if="!isLoadingStats" class="rounded-lg border bg-card p-4 space-y-3">
          <div class="flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase tracking-wide">
            <Star class="h-3.5 w-3.5" />
            Репутация
          </div>

          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Завершённых сделок</span>
            <span class="text-sm font-semibold">{{ orgStats?.completed_deals ?? '—' }}</span>
          </div>

          <template v-if="hasRating">
            <div class="border-t pt-3 flex items-center justify-between">
              <div>
                <div class="text-sm font-bold text-foreground">{{ ratingValue.toFixed(1) }}</div>
                <StarRating :model-value="Math.round(ratingValue)" readonly size="sm" class="mt-0.5" />
              </div>
              <span
                class="text-xs text-primary hover:underline cursor-pointer"
                @click="router.push(`/organizations/${auth.organizationId}`)"
              >{{ orgRating?.total_reviews }} отзывов</span>
            </div>
          </template>

          <div v-else class="border-t pt-3 text-xs text-muted-foreground">
            Отзывов пока нет
          </div>
        </div>

        <!-- Мои предложения (только если была активность как перевозчик) -->
        <div v-if="!isLoadingStats && hasCarrierStats" class="rounded-lg border bg-card p-4 space-y-3">
          <div class="flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase tracking-wide">
            <HandCoins class="h-3.5 w-3.5" />
            Мои предложения
          </div>

          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Всего сделано</span>
            <span class="text-sm font-semibold">{{ orgStats?.total_offers_made }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Принято</span>
            <span class="text-sm font-semibold">{{ orgStats?.successful_offers }}</span>
          </div>
          <div v-if="offerSuccessRate !== null" class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Конверсия</span>
            <span class="text-sm font-semibold" :class="offerSuccessRate >= 50 ? 'text-emerald-600' : 'text-muted-foreground'">
              {{ offerSuccessRate }}%
            </span>
          </div>

          <div class="border-t pt-3">
            <button
              class="text-xs text-primary hover:underline"
              @click="router.push('/my-offers')"
            >Все мои предложения →</button>
          </div>
        </div>

        <!-- Поддержка -->
        <div class="rounded-lg border bg-card p-4 space-y-3">
          <div class="flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase tracking-wide">
            <Headset class="h-3.5 w-3.5" />
            Поддержка
          </div>

          <div v-if="openTickets.length > 0" class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Открытых тикетов</span>
            <span class="text-sm font-semibold text-amber-600">{{ openTickets.length }}</span>
          </div>
          <div v-else class="text-sm text-muted-foreground">
            Открытых тикетов нет
          </div>

          <div class="border-t pt-3 flex flex-col gap-2">
            <button
              class="text-xs text-primary hover:underline text-left"
              @click="router.push('/support')"
            >Написать в поддержку →</button>
            <button
              v-if="openTickets.length > 0"
              class="text-xs text-muted-foreground hover:underline text-left"
              @click="router.push('/support')"
            >Посмотреть тикеты →</button>
          </div>
        </div>

      </div>

    </div>

  </div>
</template>
