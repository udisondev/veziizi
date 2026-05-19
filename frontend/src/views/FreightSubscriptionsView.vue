<script setup lang="ts">
import { onMounted, ref, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useSubscriptionsStore } from '@/stores/subscriptions'
import { storeToRefs } from 'pinia'
import { MAX_SUBSCRIPTIONS_PER_MEMBER } from '@/types/subscription'
import type { FreightSubscription } from '@/types/subscription'
import { freightRequestsApi } from '@/api/freightRequests'
import { subscriptionToParams, useSubscriptionNavigation } from '@/composables/useSubscriptionNavigation'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

// Shared Components
import { PageHeader, LoadingSpinner, EmptyState } from '@/components/shared'

// Subscription Components
import SubscriptionCard from '@/components/subscriptions/SubscriptionCard.vue'

// Icons
import { Plus, Bell, Info } from 'lucide-vue-next'

const router = useRouter()
const store = useSubscriptionsStore()
const { subscriptions, isLoading, canCreateMore, subscriptionsCount, activeCount } = storeToRefs(store)
const { goToSubscriptionResults } = useSubscriptionNavigation()

const matchCounts = ref<Record<string, number>>({})

// Отслеживаем только ID и is_active — перезагружаем счётчики при изменении состава или активности
const subscriptionKeys = computed(() =>
  subscriptions.value.map(s => `${s.id}:${s.is_active}`).join(',')
)

async function loadMatchCounts() {
  const active = subscriptions.value.filter(s => s.is_active)
  matchCounts.value = {}
  if (!active.length) return
  const results = await Promise.allSettled(active.map(sub => freightRequestsApi.list(subscriptionToParams(sub))))
  active.forEach((sub, i) => {
    const r = results[i]
    if (r?.status === 'fulfilled') matchCounts.value[sub.id] = r.value.items?.length ?? 0
  })
}

function goToCreate() {
  router.push({ name: 'subscription-create' })
}

function goToEdit(subscription: FreightSubscription) {
  router.push({ name: 'subscription-edit', params: { id: subscription.id } })
}

async function handleDelete(id: string) {
  await store.deleteSubscription(id)
}

function handleToggleActive(id: string, value: boolean) {
  store.toggleActive(id, value)
}

watch(subscriptionKeys, () => {
  loadMatchCounts()
}, { immediate: true })

onMounted(() => {
  store.fetchSubscriptions()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Рассылка" class="mb-6">
      <template #actions>
        <Button v-if="canCreateMore" @click="goToCreate">
          <Plus class="h-4 w-4 mr-2" />
          Создать подписку
        </Button>
        <span v-else class="text-sm text-warning font-medium">
          Достигнут лимит ({{ MAX_SUBSCRIPTIONS_PER_MEMBER }})
        </span>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="isLoading" text="Загрузка подписок..." />

    <template v-else>
      <!-- Info Card -->
      <Card class="mb-6 bg-accent border-primary/20">
        <CardContent class="flex items-start gap-3 pt-6">
          <Info class="h-5 w-5 text-primary mt-0.5 flex-shrink-0" />
          <div class="text-sm text-accent-foreground">
            <p class="font-medium mb-1">Как это работает</p>
            <p>
              Создайте подписки с нужными фильтрами, и вы будете получать уведомления
              только о подходящих заявках. Можно создать до {{ MAX_SUBSCRIPTIONS_PER_MEMBER }} подписок.
              Если фильтр не указан — подходят любые значения этого параметра.
            </p>
          </div>
        </CardContent>
      </Card>

      <!-- Stats -->
      <div class="flex items-center gap-6 mb-6 text-sm text-muted-foreground">
        <span>Всего подписок: <strong class="text-foreground">{{ subscriptionsCount }}</strong></span>
        <span>Активных: <strong class="text-success">{{ activeCount }}</strong></span>
      </div>

      <!-- Subscriptions List -->
      <div v-if="subscriptions.length > 0" class="grid gap-4 md:grid-cols-2">
        <SubscriptionCard
          v-for="subscription in subscriptions"
          :key="subscription.id"
          :subscription="subscription"
          :match-count="matchCounts[subscription.id]"
          @edit="goToEdit"
          @delete="handleDelete"
          @toggle-active="handleToggleActive"
          @view-matches="goToSubscriptionResults(subscription)"
        />
      </div>

      <!-- Empty State -->
      <EmptyState
        v-else
        title="Нет подписок"
        description="Создайте первую подписку, чтобы получать уведомления о подходящих заявках"
      >
        <template #icon>
          <Bell class="h-12 w-12 text-muted-foreground/50" />
        </template>
        <template #action>
          <Button @click="goToCreate">
            <Plus class="h-4 w-4 mr-2" />
            Создать подписку
          </Button>
        </template>
      </EmptyState>
    </template>
  </div>
</template>
