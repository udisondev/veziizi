<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSubscriptionsStore } from '@/stores/subscriptions'
import { storeToRefs } from 'pinia'
import { MAX_SUBSCRIPTIONS_PER_MEMBER } from '@/types/subscription'
import type { FreightSubscription } from '@/types/subscription'

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
        <span v-else class="text-sm text-muted-foreground">
          Достигнут лимит ({{ MAX_SUBSCRIPTIONS_PER_MEMBER }})
        </span>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="isLoading" text="Загрузка подписок..." />

    <template v-else>
      <!-- Info Card -->
      <Card class="mb-6 bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
        <CardContent class="flex items-start gap-3 pt-6">
          <Info class="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
          <div class="text-sm text-blue-800 dark:text-blue-200">
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
        <span>Активных: <strong class="text-green-600">{{ activeCount }}</strong></span>
      </div>

      <!-- Subscriptions List -->
      <div v-if="subscriptions.length > 0" class="grid gap-4 md:grid-cols-2">
        <SubscriptionCard
          v-for="subscription in subscriptions"
          :key="subscription.id"
          :subscription="subscription"
          @edit="goToEdit"
          @delete="handleDelete"
          @toggle-active="handleToggleActive"
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
