<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import { freightRequestsApi } from '@/api/freightRequests'
import type { FreightRequestListItem, FreightRequestStatus, FreightRequestStatusFilter, OwnershipFilter, Country, CountryFilter } from '@/types/freightRequest'
import {
  cargoTypeLabels,
  bodyTypeLabels,
  currencyLabels,
  ownershipOptions,
  statusOptions,
  countryOptions,
  countryLabels,
} from '@/types/freightRequest'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
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
import { Plus, Clock, Building2, Package, Bell } from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()
const { canCreateFreightRequest } = usePermissions()

const items = ref<FreightRequestListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const showFilters = ref(false)

// Filters (applied state)
const ownershipFilter = ref<OwnershipFilter>('all')
const statusFilter = ref<FreightRequestStatusFilter>('all')
const orgNameFilter = ref('')
const orgINNFilter = ref('')
const orgCountryFilter = ref<CountryFilter>('all')

// Temp filters for sheet
const tempOwnership = ref<OwnershipFilter>('all')
const tempStatus = ref<FreightRequestStatusFilter>('all')
const tempOrgName = ref('')
const tempOrgINN = ref('')
const tempOrgCountry = ref<CountryFilter>('all')

// Status map for StatusBadge
const freightStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  published: { label: 'Опубликована', variant: 'success' },
  selected: { label: 'Выбран исполнитель', variant: 'warning' },
  confirmed: { label: 'Подтверждена', variant: 'info' },
  cancelled: { label: 'Отменена', variant: 'destructive' },
  expired: { label: 'Истекла', variant: 'secondary' },
}

// Computed
const hasActiveFilters = computed(() =>
  ownershipFilter.value !== 'all' ||
  statusFilter.value !== 'all' ||
  orgNameFilter.value !== '' ||
  orgINNFilter.value !== '' ||
  orgCountryFilter.value !== 'all'
)

const activeFiltersCount = computed(() => {
  let count = 0
  if (ownershipFilter.value !== 'all') count++
  if (statusFilter.value !== 'all') count++
  if (orgNameFilter.value !== '') count++
  if (orgINNFilter.value !== '') count++
  if (orgCountryFilter.value !== 'all') count++
  return count
})

// Sheet functions
function openFilters() {
  tempOwnership.value = ownershipFilter.value
  tempStatus.value = statusFilter.value
  tempOrgName.value = orgNameFilter.value
  tempOrgINN.value = orgINNFilter.value
  tempOrgCountry.value = orgCountryFilter.value
  showFilters.value = true
}

function applyFilters() {
  ownershipFilter.value = tempOwnership.value
  statusFilter.value = tempStatus.value
  orgNameFilter.value = tempOrgName.value
  orgINNFilter.value = tempOrgINN.value
  orgCountryFilter.value = tempOrgCountry.value
  showFilters.value = false
}

function resetFilters() {
  tempOwnership.value = 'all'
  tempStatus.value = 'all'
  tempOrgName.value = ''
  tempOrgINN.value = ''
  tempOrgCountry.value = 'all'
}

function resetAllFilters() {
  ownershipFilter.value = 'all'
  statusFilter.value = 'all'
  orgNameFilter.value = ''
  orgINNFilter.value = ''
  orgCountryFilter.value = 'all'
}

// Load data with filters
async function loadItems() {
  isLoading.value = true
  error.value = null

  try {
    const params: Parameters<typeof freightRequestsApi.list>[0] = {}

    if (ownershipFilter.value === 'my_org' && auth.organizationId) {
      params.customer_org_id = auth.organizationId
    } else if (ownershipFilter.value === 'my' && auth.memberId) {
      params.member_id = auth.memberId
    }

    if (statusFilter.value !== 'all') params.status = statusFilter.value as FreightRequestStatus
    if (orgNameFilter.value) params.org_name = orgNameFilter.value
    if (orgINNFilter.value) params.org_inn = orgINNFilter.value
    if (orgCountryFilter.value !== 'all') params.org_country = orgCountryFilter.value as Country

    items.value = await freightRequestsApi.list(params)
  } catch (e) {
    error.value = 'Не удалось загрузить заявки'
    console.error(e)
  } finally {
    isLoading.value = false
  }
}

function goToDetail(id: string) {
  router.push(`/freight-requests/${id}`)
}

function goToCreate() {
  router.push('/freight-requests/new')
}

function goToSubscriptions() {
  router.push('/notifications/subscriptions')
}

function formatPrice(amount?: number, currency?: string): string {
  if (!amount || !currency) return '—'
  const value = amount / 100
  const symbol = currencyLabels[currency as keyof typeof currencyLabels] || currency
  return `${value.toLocaleString('ru-RU')} ${symbol}`
}

function formatWeight(weight?: number): string {
  if (!weight) return '—'
  return `${weight.toLocaleString('ru-RU')} т`
}

function getTransitPointsCount(item: FreightRequestListItem): number {
  if (!item.route?.points || item.route.points.length <= 2) return 0
  return item.route.points.length - 2
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function formatBodyTypes(types?: string[]): string {
  if (!types || types.length === 0) return '—'
  return types
    .map(t => bodyTypeLabels[t as keyof typeof bodyTypeLabels] || t)
    .join(', ')
}

function isExpiringSoon(expiresAt: string): boolean {
  const expires = new Date(expiresAt)
  const now = new Date()
  const diffDays = (expires.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  return diffDays > 0 && diffDays <= 3
}

// Watch filters and reload
watch(
  [ownershipFilter, statusFilter, orgNameFilter, orgINNFilter, orgCountryFilter],
  () => loadItems()
)

onMounted(() => {
  loadItems()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <PageHeader title="Заявки на перевозку" class="mb-6">
      <template #actions>
        <!-- Subscription Bell -->
        <Button variant="outline" size="icon" @click="goToSubscriptions" title="Настроить подписку на заявки">
          <Bell class="h-4 w-4" />
        </Button>

        <!-- Filters Sheet -->
        <FilterSheet
          v-model:open="showFilters"
          :active-filters-count="activeFiltersCount"
          description="Настройте параметры поиска заявок"
          @open="openFilters"
          @apply="applyFilters"
          @reset="resetFilters"
        >
          <!-- Ownership -->
          <div class="space-y-2">
            <Label>Принадлежность</Label>
            <Select v-model="tempOwnership">
              <SelectTrigger>
                <SelectValue placeholder="Выберите..." />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in ownershipOptions"
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
                  v-for="opt in statusOptions"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <!-- Organization name -->
          <div class="space-y-2">
            <Label>Название организации</Label>
            <Input
              v-model="tempOrgName"
              placeholder="Поиск по названию"
            />
          </div>

          <!-- INN -->
          <div class="space-y-2">
            <Label>ИНН</Label>
            <Input
              v-model="tempOrgINN"
              placeholder="Поиск по ИНН"
            />
          </div>

          <!-- Country -->
          <div class="space-y-2">
            <Label>Страна</Label>
            <Select v-model="tempOrgCountry">
              <SelectTrigger>
                <SelectValue placeholder="Все страны" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in countryOptions"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </FilterSheet>

        <Button v-if="canCreateFreightRequest" @click="goToCreate">
          <Plus class="mr-2 h-4 w-4" />
          Новая заявка
        </Button>
      </template>
    </PageHeader>

    <!-- Active filters indicator -->
    <Card v-if="hasActiveFilters" class="mb-6 border-primary/20 bg-primary/5">
      <CardContent class="flex items-center justify-between py-3">
        <div class="text-sm text-primary flex flex-wrap gap-x-2 gap-y-1">
          <span v-if="ownershipFilter !== 'all'">
            {{ ownershipOptions.find(o => o.value === ownershipFilter)?.label }}
          </span>
          <span v-if="statusFilter !== 'all'">
            <span v-if="ownershipFilter !== 'all'">, </span>
            Статус: {{ statusOptions.find(o => o.value === statusFilter)?.label }}
          </span>
          <span v-if="orgNameFilter">
            <span v-if="ownershipFilter !== 'all' || statusFilter !== 'all'">, </span>
            Организация: "{{ orgNameFilter }}"
          </span>
          <span v-if="orgINNFilter">
            <span v-if="ownershipFilter !== 'all' || statusFilter !== 'all' || orgNameFilter">, </span>
            ИНН: "{{ orgINNFilter }}"
          </span>
          <span v-if="orgCountryFilter !== 'all'">
            <span v-if="ownershipFilter !== 'all' || statusFilter !== 'all' || orgNameFilter || orgINNFilter">, </span>
            Страна: {{ countryLabels[orgCountryFilter as Country] }}
          </span>
        </div>
        <Button variant="ghost" size="sm" @click="resetAllFilters">
          Сбросить
        </Button>
      </CardContent>
    </Card>

    <!-- Loading -->
    <LoadingSpinner v-if="isLoading" text="Загрузка заявок..." />

    <!-- Error -->
    <ErrorBanner
      v-else-if="error"
      :message="error"
      @retry="loadItems"
    />

    <!-- Empty state -->
    <EmptyState
      v-else-if="items.length === 0"
      :icon="Package"
      title="Заявок пока нет"
      :description="hasActiveFilters ? 'Нет заявок по заданным фильтрам' : 'Создайте первую заявку на перевозку'"
      :action-label="canCreateFreightRequest && !hasActiveFilters ? 'Создать заявку' : undefined"
      @action="goToCreate"
    />

    <!-- List -->
    <div v-else class="space-y-4">
      <Card
        v-for="item in items"
        :key="item.id"
        class="hover:shadow-md transition-shadow cursor-pointer"
        @click="goToDetail(item.id)"
      >
        <CardContent class="p-4">
          <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <!-- Route -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <StatusBadge :status="item.status" :status-map="freightStatusMap" />
                <span
                  v-if="item.status === 'published' && isExpiringSoon(item.expires_at)"
                  class="inline-flex items-center gap-1 text-xs text-warning"
                >
                  <Clock class="h-3 w-3" />
                  Истекает скоро
                </span>
              </div>

              <!-- Route with vertical dashed line -->
              <div class="flex items-stretch gap-3">
                <!-- Vertical line with dots -->
                <div class="flex flex-col items-center py-1">
                  <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
                  <div class="w-px flex-1 border-l border-dashed border-muted-foreground/40 min-h-2" />
                  <div
                    v-if="getTransitPointsCount(item) > 0"
                    class="text-xs text-muted-foreground bg-background px-1 shrink-0"
                  >
                    +{{ getTransitPointsCount(item) }}
                  </div>
                  <div
                    v-if="getTransitPointsCount(item) > 0"
                    class="w-px flex-1 border-l border-dashed border-muted-foreground/40 min-h-2"
                  />
                  <div class="w-2 h-2 rounded-full bg-primary shrink-0" />
                </div>
                <!-- Addresses -->
                <div class="flex flex-col justify-between flex-1 min-w-0 gap-1">
                  <div class="text-lg font-medium text-foreground truncate">
                    {{ item.origin_address || 'Не указан' }}
                  </div>
                  <div class="text-lg font-medium text-foreground truncate">
                    {{ item.destination_address || 'Не указан' }}
                  </div>
                </div>
              </div>

              <!-- Organization info -->
              <div v-if="item.customer_org_name" class="flex items-center gap-1 text-sm text-muted-foreground mt-2">
                <Building2 class="h-4 w-4" />
                {{ item.customer_org_name }}
                <span v-if="item.customer_org_country" class="text-muted-foreground/70">
                  ({{ countryLabels[item.customer_org_country as Country] || item.customer_org_country }})
                </span>
              </div>
            </div>

            <!-- Details -->
            <div class="flex flex-wrap gap-4 lg:gap-6 text-sm">
              <!-- Cargo -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Груз</div>
                <div class="font-medium">
                  {{ item.cargo_type ? cargoTypeLabels[item.cargo_type] : '—' }}
                </div>
                <div class="text-muted-foreground">{{ formatWeight(item.cargo_weight) }}</div>
              </div>

              <!-- Vehicle -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Кузов</div>
                <div class="font-medium truncate max-w-32">
                  {{ formatBodyTypes(item.body_types) }}
                </div>
              </div>

              <!-- Price -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Ставка</div>
                <div class="font-medium text-success">
                  {{ formatPrice(item.price_amount, item.price_currency) }}
                </div>
              </div>

              <!-- Date -->
              <div class="min-w-24">
                <div class="text-muted-foreground">Создана</div>
                <div class="font-medium">{{ formatDate(item.created_at) }}</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
