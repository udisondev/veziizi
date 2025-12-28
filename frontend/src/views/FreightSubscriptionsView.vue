<script setup lang="ts">
import { onMounted, ref } from 'vue'
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
import SubscriptionFormDialog from '@/components/subscriptions/SubscriptionFormDialog.vue'

// Icons
import { Plus, Bell, Info } from 'lucide-vue-next'

const store = useSubscriptionsStore()
const { subscriptions, isLoading, canCreateMore, subscriptionsCount, activeCount } = storeToRefs(store)

const isFormOpen = ref(false)
const editingSubscription = ref<FreightSubscription | null>(null)

function openCreateForm() {
  editingSubscription.value = null
  isFormOpen.value = true
}

function openEditForm(subscription: FreightSubscription) {
  editingSubscription.value = subscription
  isFormOpen.value = true
}

function closeForm() {
  isFormOpen.value = false
  editingSubscription.value = null
}

async function handleFormSuccess() {
  closeForm()
  await store.fetchSubscriptions()
}

async function handleDelete(id: string) {
  await store.deleteSubscription(id)
}

async function handleToggleActive(id: string) {
  await store.toggleActive(id)
}

onMounted(() => {
  store.fetchSubscriptions()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Рассылка" class="mb-6">
      <template #actions>
        <Button v-if="canCreateMore" @click="openCreateForm">
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
          @edit="openEditForm"
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
          <Button @click="openCreateForm">
            <Plus class="h-4 w-4 mr-2" />
            Создать подписку
          </Button>
        </template>
      </EmptyState>
    </template>

    <!-- Form Dialog -->
    <SubscriptionFormDialog
      v-model:open="isFormOpen"
      :subscription="editingSubscription"
      @success="handleFormSuccess"
      @cancel="closeForm"
    />
  </div>
</template>
