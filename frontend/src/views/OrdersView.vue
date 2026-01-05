<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ordersApi } from '@/api/orders'
import type { OrderListItem, OrderStatus, OrderStatusFilter, ViewMode } from '@/types/order'
import {
  orderStatusOptions,
  viewModeOptions,
  viewModeLabels,
} from '@/types/order'
import { orderStatusMapSimple as orderStatusMap } from '@/constants/statusMaps'
import { formatDateShort } from '@/utils/formatters'
import { logger } from '@/utils/logger'

// UI Components
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// Shared Components
import {
  PageHeader,
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
  FilterSheet,
} from '@/components/shared'

// Icons
import { ClipboardList, Calendar } from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()

const items = ref<OrderListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const showFilters = ref(false)

// Filters (applied state)
const viewMode = ref<ViewMode>('all')
const statusFilter = ref<OrderStatusFilter>('all')

// Temp filters for sheet
const tempViewMode = ref<ViewMode>('all')
const tempStatus = ref<OrderStatusFilter>('all')


// Computed
const hasActiveFilters = computed(() =>
  viewMode.value !== 'all' || statusFilter.value !== 'all'
)

const activeFiltersCount = computed(() => {
  let count = 0
  if (viewMode.value !== 'all') count++
  if (statusFilter.value !== 'all') count++
  return count
})

async function loadItems() {
  isLoading.value = true
  error.value = null

  try {
    const params: Parameters<typeof ordersApi.list>[0] = {}

    if (viewMode.value === 'as_customer' && auth.organizationId) {
      params.customer_org_id = auth.organizationId
    } else if (viewMode.value === 'as_carrier' && auth.organizationId) {
      params.carrier_org_id = auth.organizationId
    }

    if (statusFilter.value !== 'all') {
      params.status = statusFilter.value as OrderStatus
    }

    items.value = await ordersApi.list(params)
  } catch (e) {
    error.value = 'Не удалось загрузить заказы'
    logger.error('Failed to load orders', e)
  } finally {
    isLoading.value = false
  }
}

function goToDetail(id: string) {
  router.push(`/orders/${id}`)
}

function getRole(item: OrderListItem): string {
  if (item.customer_org_id === auth.organizationId) return 'Заказчик'
  if (item.carrier_org_id === auth.organizationId) return 'Перевозчик'
  return ''
}

// Sheet functions
function openFilters() {
  tempViewMode.value = viewMode.value
  tempStatus.value = statusFilter.value
  showFilters.value = true
}

function applyFilters() {
  viewMode.value = tempViewMode.value
  statusFilter.value = tempStatus.value
  showFilters.value = false
}

function resetFilters() {
  tempViewMode.value = 'all'
  tempStatus.value = 'all'
}

function resetAllFilters() {
  viewMode.value = 'all'
  statusFilter.value = 'all'
}

// Watch filters
watch([viewMode, statusFilter], () => {
  loadItems()
})

onMounted(() => {
  loadItems()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <PageHeader title="Заказы" class="mb-6">
      <template #actions>
        <!-- Filters Sheet -->
        <FilterSheet
          v-model:open="showFilters"
          :active-filters-count="activeFiltersCount"
          description="Настройте параметры отображения заказов"
          @open="openFilters"
          @apply="applyFilters"
          @reset="resetFilters"
        >
          <!-- View Mode -->
          <div class="space-y-2">
            <Label>Отображение</Label>
            <Select v-model="tempViewMode">
              <SelectTrigger>
                <SelectValue placeholder="Выберите..." />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in viewModeOptions"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <!-- Status -->
          <div class="space-y-2">
            <Label>Статус</Label>
            <Select v-model="tempStatus">
              <SelectTrigger>
                <SelectValue placeholder="Все статусы" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in orderStatusOptions"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </FilterSheet>
      </template>
    </PageHeader>

    <!-- Active filters indicator -->
    <Card v-if="hasActiveFilters" class="mb-6 border-primary/20 bg-primary/5">
      <CardContent class="flex items-center justify-between py-3">
        <div class="text-sm text-primary flex flex-wrap gap-x-2 gap-y-1">
          <span v-if="viewMode !== 'all'">
            {{ viewModeLabels[viewMode] }}
          </span>
          <span v-if="statusFilter !== 'all'">
            <span v-if="viewMode !== 'all'">, </span>
            Статус: {{ orderStatusOptions.find(o => o.value === statusFilter)?.label }}
          </span>
        </div>
        <Button variant="ghost" size="sm" @click="resetAllFilters">
          Сбросить
        </Button>
      </CardContent>
    </Card>

    <!-- Loading -->
    <LoadingSpinner v-if="isLoading" text="Загрузка заказов..." />

    <!-- Error -->
    <ErrorBanner
      v-else-if="error"
      :message="error"
      @retry="loadItems"
    />

    <!-- Empty state -->
    <EmptyState
      v-else-if="items.length === 0"
      :icon="ClipboardList"
      title="Заказов пока нет"
      description="Заказы появятся после подтверждения офферов на заявки"
    />

    <!-- List -->
    <div v-else class="space-y-4">
      <Card
        v-for="(item, idx) in items"
        :key="item.id"
        :data-tutorial="idx === 0 ? 'order-card' : undefined"
        class="hover:shadow-md transition-shadow cursor-pointer"
        @click="goToDetail(item.id)"
      >
        <CardContent class="p-4">
          <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-2">
                <StatusBadge :status="item.status" :status-map="orderStatusMap" />
                <Badge v-if="getRole(item)" variant="secondary">
                  {{ getRole(item) }}
                </Badge>
              </div>
              <div class="text-sm text-muted-foreground">
                Заказ #{{ item.order_number }}
              </div>
            </div>

            <div class="flex items-center gap-1 text-sm text-muted-foreground">
              <Calendar class="h-4 w-4" />
              {{ formatDateShort(item.created_at) }}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
