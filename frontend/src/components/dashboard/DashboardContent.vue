<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useFreightFiltersStore } from '@/stores/freightFilters'
import { freightRequestsApi } from '@/api/freightRequests'
import type { CarrierInviteItem } from '@/types/freightRequest'
import { subscriptionToParams, useSubscriptionNavigation } from '@/composables/useSubscriptionNavigation'
import { offersApi, type MyOfferListItem } from '@/api/offers'
import { organizationsApi } from '@/api/organizations'
import { subscriptionsApi } from '@/api/subscriptions'
import type { FreightRequestListItem, FreightRequestStatus, OwnershipFilter } from '@/types/freightRequest'
import type { OrganizationStats, DashboardStats, OrganizationRating, PendingOfferItem } from '@/types/admin'
import type { FreightSubscription } from '@/types/subscription'
import { currencyLabels } from '@/types/freightRequest'
import { formatRelativeTime, pluralizeRu } from '@/utils/formatters'
import { logger } from '@/utils/logger'
import { Button } from '@/components/ui/button'
import { LoadingSpinner } from '@/components/shared'
import StarRating from '@/components/freight-request/StarRating.vue'
import { getMyTickets, type TicketListItem } from '@/api/support'
import { ChevronRight, TrendingUp, Star, Headset, HandCoins, Bell, Clock, Truck } from 'lucide-vue-next'
import { eventStream } from '@/services/eventStream'

const emit = defineEmits<{
  'go-to-new': []
  'go-to-list': [filtered?: boolean]
}>()

const router = useRouter()
const auth = useAuthStore()
const filtersStore = useFreightFiltersStore()

function goToListFiltered(statuses: FreightRequestStatus[], ownership: OwnershipFilter) {
  filtersStore.resetFilters()
  filtersStore.setFilters({ statuses, ownership })
  emit('go-to-list', true)
}


const orgStats = ref<OrganizationStats | null>(null)
const dashboardStats = ref<DashboardStats | null>(null)
const orgRating = ref<OrganizationRating | null>(null)
const selectedOffers = ref<MyOfferListItem[]>([])
const myActiveRequests = ref<FreightRequestListItem[]>([])
const pendingOffersOnMyRequests = ref<PendingOfferItem[]>([])
const openTickets = ref<TicketListItem[]>([])
const subscriptionMatches = ref<{ sub: FreightSubscription; items: FreightRequestListItem[] }[]>([])
const incomingInvitations = ref<CarrierInviteItem[]>([])

const INVITATIONS_SEEN_KEY = 'carrier_invitations_seen_ids'

function getSeenInvitationIds(): Set<string> {
  try {
    const v = localStorage.getItem(INVITATIONS_SEEN_KEY)
    return v ? new Set(JSON.parse(v) as string[]) : new Set()
  } catch {
    return new Set()
  }
}

function markInvitationSeen(id: string) {
  try {
    const seen = getSeenInvitationIds()
    seen.add(id)
    // Ограничиваем размер — оставляем последние 500
    const arr = Array.from(seen).slice(-500)
    localStorage.setItem(INVITATIONS_SEEN_KEY, JSON.stringify(arr))
  } catch {
    // localStorage может быть недоступен (квота, приватный режим)
  }
}

const isLoadingStats = ref(false)
const isLoadingAttention = ref(false)
const isLoadingSubscriptions = ref(false)

const partiallyCompletedRequests = computed(() =>
  myActiveRequests.value.filter(i => i.status === 'partially_completed')
)

const publishedRequests = computed(() =>
  myActiveRequests.value.filter(i => i.status === 'published')
)

const offersByRequestId = computed(() => {
  const map = new Map<string, typeof pendingOffersOnMyRequests.value[0]>()
  for (const offer of pendingOffersOnMyRequests.value) {
    map.set(offer.freight_request_id, offer)
  }
  return map
})


const expiringSoonRequests = computed(() =>
  myActiveRequests.value.filter(i => {
    if (i.status !== 'published') return false
    const diff = (new Date(i.expires_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
    return diff > 0 && diff <= 3
  })
)

const hasAttentionItems = computed(
  () => selectedOffers.value.length + partiallyCompletedRequests.value.length > 0
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

const totalActivePipeline = computed(() => {
  const s = dashboardStats.value
  return (
    (s?.as_customer_published ?? 0) +
    (s?.as_customer_selected ?? 0) +
    (s?.as_customer_confirmed ?? 0) +
    (hasCarrierActivity.value
      ? (s?.as_carrier_confirmed ?? 0) + (s?.as_carrier_partially_completed ?? 0)
      : 0)
  )
})

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
  return stages
})

const pipelineGridClass = computed(() =>
  pipelineStages.value.length === 5 ? 'sm:grid-cols-5' : 'sm:grid-cols-3'
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
  const isEmployee = auth.role === 'employee'
  const memberId = auth.memberId ?? undefined
  isLoadingStats.value = true
  isLoadingAttention.value = true
  try {
    const [statsRes, dashboardRes, ratingRes, selectedOffersRes, myActiveRes, pendingOffersRes, ticketsRes, invitationsRes] = await Promise.allSettled([
      organizationsApi.getStats(orgID),
      organizationsApi.getDashboardStats(orgID),
      organizationsApi.getRating(orgID),
      offersApi.listMy({ status: 'selected', limit: 10 }),
      freightRequestsApi.list(
        isEmployee
          ? { member_id: memberId, statuses: 'published,partially_completed', limit: 50 }
          : { customer_org_id: orgID, statuses: 'published,partially_completed', limit: 50 }
      ),
      organizationsApi.getPendingOffers(orgID, 50, isEmployee ? memberId : undefined),
      getMyTickets({ status: 'open' }),
      freightRequestsApi.listCarrierInvitations({ limit: 10 }),
    ])
    orgStats.value = fulfilled('stats', statsRes)
    dashboardStats.value = fulfilled('dashboard-stats', dashboardRes)
    orgRating.value = fulfilled('rating', ratingRes)
    selectedOffers.value = fulfilled('selected-offers', selectedOffersRes) ?? []
    myActiveRequests.value = fulfilled('my-active', myActiveRes)?.items ?? []
    pendingOffersOnMyRequests.value = fulfilled('pending-offers', pendingOffersRes) ?? []
    openTickets.value = fulfilled('tickets', ticketsRes) ?? []
    const allInvitations = fulfilled('invitations', invitationsRes)?.items ?? []
    const seenIds = getSeenInvitationIds()
    incomingInvitations.value = allInvitations.filter(inv => !seenIds.has(inv.id))
  } finally {
    isLoadingStats.value = false
    isLoadingAttention.value = false
  }
}

function formatOffersCount(n: number): string {
  return `${n} ${pluralizeRu(n, 'предложение', 'предложения', 'предложений')}`
}

function formatPrice(amount?: number | null, currency?: string | null): string {
  if (!amount || !currency) return ''
  const symbol = currencyLabels[currency as keyof typeof currencyLabels] || currency
  return `${(amount / 100).toLocaleString('ru-RU')} ${symbol}`
}

function daysUntilExpiry(expiresAt: string): number {
  return Math.ceil((new Date(expiresAt).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
}

function daysLabel(expiresAt: string): string {
  const d = daysUntilExpiry(expiresAt)
  if (d === 1) return `${d} день`
  if (d >= 2 && d <= 4) return `${d} дня`
  return `${d} дней`
}

function subPriceRange(items: typeof subscriptionMatches.value[0]['items']): string | null {
  const prices = items.map(i => i.price_amount).filter((p): p is number => !!p)
  if (!prices.length) return null
  const currency = items.find(i => i.price_amount)?.price_currency
  const fmt = (n: number) => (n / 100).toLocaleString('ru-RU')
  const symbol = currency ? (currencyLabels[currency as keyof typeof currencyLabels] ?? currency) : ''
  const min = Math.min(...prices)
  const max = Math.max(...prices)
  return min === max ? `${fmt(min)} ${symbol}` : `${fmt(min)} – ${fmt(max)} ${symbol}`
}

function subLastUpdated(items: typeof subscriptionMatches.value[0]['items']): string | null {
  if (!items.length) return null
  const latest = items.reduce((a, b) => a.created_at > b.created_at ? a : b)
  return formatRelativeTime(latest.created_at)
}

const { applyFilters } = useSubscriptionNavigation()

function goToSubscriptionResults(sub: FreightSubscription) {
  applyFilters(sub)
  emit('go-to-list', true)
}

async function loadSubscriptions() {
  isLoadingSubscriptions.value = true
  try {
    const subs = await subscriptionsApi.list()
    const active = subs.filter(s => s.is_active)
    if (!active.length) return
    const results = await Promise.allSettled(
      active.map(sub => freightRequestsApi.list(subscriptionToParams(sub)))
    )
    subscriptionMatches.value = active.flatMap((sub, i) => {
      const r = results[i]
      if (!r) return []
      const items = r.status === 'fulfilled' ? (r.value.items ?? []) : []
      return items.length > 0 ? [{ sub, items }] : []
    })
  } catch (e) {
    logger.error('Dashboard: failed to load subscription matches', e)
  } finally {
    isLoadingSubscriptions.value = false
  }
}

let refreshTimer: number | null = null

function scheduleRefresh() {
  if (refreshTimer !== null) window.clearTimeout(refreshTimer)
  refreshTimer = window.setTimeout(() => {
    refreshTimer = null
    loadData()
  }, 300)
}

onMounted(() => {
  loadData()
  loadSubscriptions()
  eventStream.on('freight_request', scheduleRefresh)
  eventStream.on('support_ticket', scheduleRefresh)
  eventStream.onConnected(loadData)
})

onUnmounted(() => {
  if (refreshTimer !== null) window.clearTimeout(refreshTimer)
  eventStream.off('freight_request', scheduleRefresh)
  eventStream.off('support_ticket', scheduleRefresh)
  eventStream.offConnected(loadData)
})
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
          <span class="text-xs font-semibold text-muted-foreground">{{ totalActivePipeline }} всего</span>
        </div>

        <!-- Пайплайн -->
        <div
          v-if="hasCustomerActivity || hasCarrierActivity"
          class="grid grid-cols-2 gap-px bg-border"
          :class="pipelineGridClass"
        >
          <button
            v-for="(stage, index) in pipelineStages"
            :key="stage.label"
            class="bg-card px-2 py-3 sm:px-3 sm:py-2.5 flex flex-col items-center gap-0.5 transition-colors"
            :class="[
              stage.value > 0 ? 'hover:bg-muted/60 cursor-pointer' : 'cursor-default',
              index === pipelineStages.length - 1 && pipelineStages.length % 2 !== 0 ? 'col-span-2 sm:col-span-1' : '',
            ]"
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

        <div v-else class="flex items-center justify-between">
          <span class="text-sm text-muted-foreground">Нет активных заявок</span>
          <Button size="sm" variant="outline" @click="emit('go-to-new')">Создать заявку</Button>
        </div>
      </div>

    </div>

    <!-- Action cards (urgent attention) -->
    <div v-if="!isLoadingAttention && hasAttentionItems" class="space-y-2">

      <div
        v-for="offer in selectedOffers"
        :key="offer.id"
        class="flex items-center gap-4 rounded-lg border border-primary/30 bg-primary/5 px-4 py-3.5 cursor-pointer hover:bg-primary/10 transition-colors"
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
        class="flex items-center gap-4 rounded-lg border border-amber-300/60 bg-amber-50 dark:bg-amber-500/10 dark:border-amber-500/30 px-4 py-3.5 cursor-pointer hover:bg-amber-100 dark:hover:bg-amber-500/20 transition-colors"
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


    </div>

    <!-- Main: offers feed + sidebar -->
    <div class="grid grid-cols-1 lg:grid-cols-[1fr_260px] gap-6 items-start">

      <!-- Офферы на мои заявки / Новые заявки на рынке -->
      <div class="space-y-4">

        <!-- Заявки по рассылкам -->
        <div v-if="isLoadingSubscriptions" class="rounded-lg border bg-card p-4">
          <div class="h-3 bg-muted animate-pulse rounded w-24 mb-3" />
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
            <div v-for="i in 3" :key="i" class="rounded-md border bg-muted/30 p-3 space-y-2">
              <div class="h-3 bg-muted animate-pulse rounded w-3/4" />
              <div class="h-4 bg-muted animate-pulse rounded w-1/2" />
            </div>
          </div>
        </div>

        <div v-else-if="subscriptionMatches.length > 0">
          <div class="flex items-center gap-2 mb-2">
            <Bell class="h-3.5 w-3.5 text-muted-foreground" />
            <span class="text-xs font-medium text-muted-foreground uppercase tracking-wide">Рассылки</span>
          </div>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
            <button
              v-for="match in subscriptionMatches"
              :key="match.sub.id"
              class="flex flex-col items-start gap-1.5 rounded-md border bg-card px-3 py-2.5 text-left hover:bg-muted/50 transition-colors"
              @click="goToSubscriptionResults(match.sub)"
            >
              <span class="text-xs font-medium text-foreground leading-tight line-clamp-2">{{ match.sub.name }}</span>
              <span class="text-xs font-semibold text-primary">
                {{ match.items.length >= 5 ? '5+' : match.items.length }}
                {{ pluralizeRu(match.items.length, 'заявка', 'заявки', 'заявок') }}
              </span>
              <span v-if="subPriceRange(match.items)" class="text-xs text-muted-foreground">
                {{ subPriceRange(match.items) }}
              </span>
              <span v-if="subLastUpdated(match.items)" class="text-[11px] text-muted-foreground/60">
                Обновлено {{ subLastUpdated(match.items) }}
              </span>
            </button>
          </div>
        </div>

        <!-- Заявки истекают скоро -->
        <div v-if="expiringSoonRequests.length > 0" class="rounded-lg border bg-card overflow-hidden">
          <div class="flex items-center gap-2.5 px-5 py-4 border-b">
            <Clock class="h-4 w-4 text-destructive shrink-0" />
            <h2 class="text-sm font-semibold text-foreground">Истекают скоро</h2>
            <span class="text-xs font-semibold px-1.5 py-0.5 rounded-full bg-destructive/10 text-destructive">{{ expiringSoonRequests.length }}</span>
          </div>
          <div>
            <div
              v-for="(item, index) in expiringSoonRequests"
              :key="item.id"
              class="flex items-center gap-3 px-5 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
              :class="index < expiringSoonRequests.length - 1 ? 'border-b' : ''"
              @click="router.push(`/freight-requests/${item.id}`)"
            >
              <div class="w-1.5 h-1.5 rounded-full bg-destructive shrink-0" />
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-1.5 text-sm">
                  <span class="text-xs text-muted-foreground shrink-0">#{{ item.request_number }}</span>
                  <span class="font-medium text-foreground truncate">
                    {{ item.origin_address || '—' }} → {{ item.destination_address || '—' }}
                  </span>
                </div>
                <div class="text-xs text-destructive mt-0.5">
                  Истекает через {{ daysLabel(item.expires_at) }}
                </div>
              </div>
              <ChevronRight class="h-4 w-4 text-muted-foreground shrink-0" />
            </div>
          </div>
        </div>

        <!-- Входящие приглашения на перевозку -->
        <div v-if="incomingInvitations.length > 0" class="rounded-lg border bg-card overflow-hidden">
          <div class="flex items-center justify-between px-5 py-4 border-b">
            <div class="flex items-center gap-2.5">
              <Truck class="h-4 w-4 text-muted-foreground shrink-0" />
              <h2 class="text-sm font-semibold text-foreground">Входящие приглашения</h2>
              <span class="text-xs font-semibold px-1.5 py-0.5 rounded-full bg-primary/10 text-primary">{{ incomingInvitations.length }}</span>
            </div>
            <button
              class="flex items-center gap-1 text-sm text-primary hover:text-primary/80 transition-colors"
              @click="router.push('/my-offers?tab=invitations')"
            >
              Все <ChevronRight class="h-4 w-4" />
            </button>
          </div>
          <div>
            <div
              v-for="(inv, index) in incomingInvitations"
              :key="inv.id"
              class="flex items-start gap-3 px-5 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
              :class="index < incomingInvitations.length - 1 ? 'border-b' : ''"
              @click="markInvitationSeen(inv.id); router.push(`/freight-requests/${inv.freight_request_id}`)"
            >
              <div class="w-1.5 h-1.5 rounded-full bg-primary mt-1.5 shrink-0" />
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-1.5 text-sm flex-wrap">
                  <span class="text-xs text-muted-foreground shrink-0">#{{ inv.request_number }}</span>
                  <span class="font-medium text-foreground truncate">
                    {{ inv.origin_address || '—' }} → {{ inv.destination_address || '—' }}
                  </span>
                </div>
                <div class="flex items-center gap-1 text-xs text-muted-foreground mt-0.5 flex-wrap">
                  <Truck class="h-3 w-3 shrink-0" />
                  <span class="font-medium text-foreground">
                    {{ [inv.vehicle_brand, inv.vehicle_model].filter(Boolean).join(' ') || 'Транспорт' }}
                  </span>
                  <template v-if="inv.vehicle_registration">
                    <span>·</span>
                    <span class="font-mono">{{ inv.vehicle_registration }}</span>
                  </template>
                  <template v-if="inv.customer_org_name">
                    <span>·</span>
                    <span>{{ inv.customer_org_name }}</span>
                  </template>
                </div>
              </div>
              <ChevronRight class="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />
            </div>
          </div>
        </div>

        <!-- Мои заявки и офферы на них -->
        <div class="rounded-lg border bg-card overflow-hidden">
          <div class="flex items-center justify-between px-5 py-4 border-b">
            <div class="flex items-center gap-2.5">
              <h2 class="text-sm font-semibold text-foreground">Мои заявки</h2>
              <span
                v-if="publishedRequests.length > 0"
                class="text-xs font-semibold px-1.5 py-0.5 rounded-full bg-muted text-muted-foreground"
              >{{ publishedRequests.length }}</span>
              <span
                v-if="(dashboardStats?.pending_offers_today ?? 0) > 0"
                class="flex items-center gap-0.5 text-xs font-medium text-emerald-600 dark:text-emerald-400"
              >
                <TrendingUp class="h-3 w-3" />
                +{{ dashboardStats!.pending_offers_today }} офферов за сутки
              </span>
            </div>
            <button
              class="flex items-center gap-1 text-sm text-primary hover:text-primary/80 transition-colors"
              @click="goToListFiltered(['published'], 'my_org')"
            >
              Все <ChevronRight class="h-4 w-4" />
            </button>
          </div>

          <div v-if="isLoadingStats" class="px-5 py-10 flex justify-center">
            <LoadingSpinner text="Загрузка..." />
          </div>

          <div v-else-if="publishedRequests.length === 0" class="px-5 py-10 text-center text-sm text-muted-foreground">
            Нет активных заявок на рынке
          </div>

          <div v-else>
            <div
              v-for="(item, index) in publishedRequests"
              :key="item.id"
              class="flex items-center gap-3 px-5 py-3.5 cursor-pointer hover:bg-muted/40 transition-colors"
              :class="index < publishedRequests.length - 1 ? 'border-b' : ''"
              @click="router.push(offersByRequestId.has(item.id) ? `/freight-requests/${item.id}?tab=offers` : `/freight-requests/${item.id}`)"
            >
              <div
                class="w-2 h-2 rounded-full shrink-0"
                :class="offersByRequestId.has(item.id) ? 'bg-primary' : 'bg-muted-foreground/30'"
              />
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-1.5 text-sm">
                  <span class="text-xs text-muted-foreground shrink-0">#{{ item.request_number }}</span>
                  <span class="font-medium text-foreground truncate">
                    {{ item.origin_address || '—' }} → {{ item.destination_address || '—' }}
                  </span>
                </div>
                <template v-if="offersByRequestId.has(item.id)">
                  <div class="flex items-center gap-1 text-xs text-muted-foreground mt-0.5">
                    <span class="font-medium text-primary">{{ formatOffersCount(offersByRequestId.get(item.id)!.offers_count) }}</span>
                    <span>·</span>
                    <span>{{ offersByRequestId.get(item.id)!.carrier_org_name }}</span>
                    <span>·</span>
                    <span>{{ formatRelativeTime(offersByRequestId.get(item.id)!.created_at) }}</span>
                  </div>
                </template>
                <div v-else class="text-xs text-muted-foreground/60 mt-0.5">
                  Нет предложений
                </div>
              </div>
              <div
                v-if="offersByRequestId.has(item.id)"
                class="text-sm font-semibold text-foreground shrink-0"
              >
                {{ formatPrice(offersByRequestId.get(item.id)!.price_amount, offersByRequestId.get(item.id)!.price_currency) || '—' }}
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
