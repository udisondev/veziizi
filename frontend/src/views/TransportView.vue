<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { vehiclesApi } from '@/api/vehicles'
import type { VehicleListItem } from '@/types/vehicle'
import {
  vehicleSubTypeLabels,
  allVehicleSubTypeOptions,
  loadingTypeOptions,
  loadingTypeLabels,
  vehicleTypeLabels,
} from '@/types/freightRequest'
import type { VehicleSubType, LoadingType } from '@/types/freightRequest'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import { logger } from '@/utils/logger'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadingSpinner, EmptyState, ErrorBanner, VehicleVerifiedBadge } from '@/components/shared'
import { RangeInput, ChipButtonGroup } from '@/components/filters'
import { MultiSelectField } from '@/components/ui/multi-select'
import InviteFreightDialog from '@/components/transport/InviteFreightDialog.vue'
import {
  SlidersHorizontal,
  Weight,
  Package,
  Building2,
  Truck,
  Thermometer,
  Shield,
  CheckSquare,
  Send,
} from 'lucide-vue-next'

const PAGE_SIZE = 20

const router = useRouter()
const auth = useAuthStore()

// Предложение заявки владельцу транспорта
const inviteDialogOpen = ref(false)
const inviteVehicle = ref<VehicleListItem | null>(null)

function openInviteDialog(item: VehicleListItem) {
  inviteVehicle.value = item
  inviteDialogOpen.value = true
}

// Фильтры
const vehicleSubTypes = ref<VehicleSubType[]>([])
const minCapacity = ref<number | undefined>()
const maxCapacity = ref<number | undefined>()
const minVolume = ref<number | undefined>()
const maxVolume = ref<number | undefined>()
const loadingTypes = ref<LoadingType[]>([])
const requiresAdr = ref(false)
const thermograph = ref(false)
const orgName = ref('')

const mobileFiltersOpen = ref(false)

const hasActiveFilters = computed(() =>
  vehicleSubTypes.value.length > 0 ||
  minCapacity.value !== undefined ||
  maxCapacity.value !== undefined ||
  minVolume.value !== undefined ||
  maxVolume.value !== undefined ||
  loadingTypes.value.length > 0 ||
  requiresAdr.value ||
  thermograph.value ||
  orgName.value !== ''
)

function resetFilters() {
  vehicleSubTypes.value = []
  minCapacity.value = undefined
  maxCapacity.value = undefined
  minVolume.value = undefined
  maxVolume.value = undefined
  loadingTypes.value = []
  requiresAdr.value = false
  thermograph.value = false
  orgName.value = ''
}

// Данные
const items = ref<VehicleListItem[]>([])
const isLoading = ref(false)
const isLoadingMore = ref(false)
const error = ref<string | null>(null)
const cursor = ref<string | undefined>()
const hasMore = ref(false)

function buildParams() {
  const params: Parameters<typeof vehiclesApi.listAll>[0] = { limit: PAGE_SIZE }
  if (vehicleSubTypes.value.length > 0) params.vehicle_subtypes = vehicleSubTypes.value.join(',')
  if (minCapacity.value !== undefined) params.min_capacity = minCapacity.value
  if (maxCapacity.value !== undefined) params.max_capacity = maxCapacity.value
  if (minVolume.value !== undefined) params.min_volume = minVolume.value
  if (maxVolume.value !== undefined) params.max_volume = maxVolume.value
  if (loadingTypes.value.length > 0) params.loading_types = loadingTypes.value.join(',')
  if (requiresAdr.value) params.requires_adr = true
  if (thermograph.value) params.thermograph = true
  if (orgName.value) params.org_name = orgName.value
  return params
}

async function loadItems() {
  isLoading.value = true
  error.value = null
  cursor.value = undefined
  items.value = []
  try {
    const res = await vehiclesApi.listAll(buildParams())
    items.value = res.items ?? []
    cursor.value = res.next_cursor
    hasMore.value = res.has_more
  } catch (e) {
    error.value = 'Не удалось загрузить транспорт'
    logger.error('Failed to load transport', e)
  } finally {
    isLoading.value = false
  }
}

async function loadMoreItems() {
  if (!hasMore.value || isLoadingMore.value || !cursor.value) return
  isLoadingMore.value = true
  try {
    const params = buildParams()
    params.cursor = cursor.value
    const res = await vehiclesApi.listAll(params)
    items.value.push(...(res.items ?? []))
    cursor.value = res.next_cursor
    hasMore.value = res.has_more
  } catch (e) {
    logger.error('Failed to load more transport', e)
  } finally {
    isLoadingMore.value = false
  }
}

const canLoadMore = computed(() => hasMore.value && !isLoadingMore.value && cursor.value !== undefined)
const { sentinelRef, reset: resetInfiniteScroll } = useInfiniteScroll(loadMoreItems, {
  threshold: 300,
  enabled: canLoadMore,
})
void sentinelRef

// Форматирование
function formatCapacity(capacity?: number): string {
  if (!capacity) return '—'
  return `${capacity} т`
}

function formatVolumeDisplay(volume?: number): string {
  if (!volume) return '—'
  return `${volume} м³`
}

function formatVehicle(item: VehicleListItem): string {
  const type = vehicleTypeLabels[item.vehicle_type] ?? item.vehicle_type
  const subtype = vehicleSubTypeLabels[item.vehicle_subtype] ?? item.vehicle_subtype
  return `${type} · ${subtype}`
}

function formatTemp(item: VehicleListItem): string | null {
  if (!item.temperature) return null
  return `${item.temperature.min}°C — ${item.temperature.max}°C`
}

// Debounce перезагрузки при смене фильтров
let debounceTimer: ReturnType<typeof setTimeout> | null = null
watch(
  [vehicleSubTypes, minCapacity, maxCapacity, minVolume, maxVolume, loadingTypes, requiresAdr, thermograph, orgName],
  () => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      await loadItems()
      await nextTick()
      resetInfiniteScroll()
    }, 300)
  },
  { deep: true }
)

onUnmounted(() => { if (debounceTimer) clearTimeout(debounceTimer) })
onMounted(() => { loadItems() })
</script>

<template>
  <div class="max-w-7xl mx-auto pt-4 pb-6 px-4 sm:px-6 lg:px-8">
    <!-- Mobile filter toggle -->
    <div class="lg:hidden mb-2">
      <Button variant="outline" class="w-full" @click="mobileFiltersOpen = !mobileFiltersOpen">
        <SlidersHorizontal class="h-4 w-4 mr-2" />
        Фильтры
        <span
          v-if="hasActiveFilters"
          class="ml-2 inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium"
        >
          ●
        </span>
      </Button>
    </div>

    <!-- 2-column layout -->
    <div class="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-6 items-start">

      <!-- Filters -->
      <div class="lg:sticky lg:top-16">
        <div :class="['accordion-grid filters-accordion', mobileFiltersOpen ? 'is-open' : '']">
          <div class="overflow-hidden lg:overflow-visible">
            <div class="space-y-4 pb-2">

              <!-- Организация -->
              <div>
                <Label>Организация</Label>
                <Input v-model="orgName" placeholder="Название компании" class="mt-2" />
              </div>

              <!-- Тип кузова -->
              <MultiSelectField
                v-model="vehicleSubTypes"
                :options="allVehicleSubTypeOptions"
                placeholder="Все типы кузова"
                sheet-label="Тип кузова"
                search-placeholder="Поиск типа кузова..."
              />

              <!-- Грузоподъёмность -->
              <RangeInput
                :min-value="minCapacity"
                :max-value="maxCapacity"
                label="Грузоподъёмность, т"
                :min="0"
                :step="0.5"
                @update:min-value="minCapacity = $event"
                @update:max-value="maxCapacity = $event"
              />

              <!-- Объём -->
              <RangeInput
                :min-value="minVolume"
                :max-value="maxVolume"
                label="Объём кузова, м³"
                :min="0"
                :step="1"
                @update:min-value="minVolume = $event"
                @update:max-value="maxVolume = $event"
              />

              <!-- Тип погрузки -->
              <ChipButtonGroup
                v-model="loadingTypes"
                :options="loadingTypeOptions"
                label="Тип погрузки"
              />

              <!-- Доп. параметры -->
              <div class="space-y-2">
                <Label>Доп. параметры</Label>
                <button
                  type="button"
                  class="flex items-center gap-2 rounded-md border px-3 py-2 text-sm font-medium w-full transition-colors"
                  :class="requiresAdr ? 'bg-accent border-primary text-accent-foreground' : 'bg-white border-input text-foreground hover:bg-muted'"
                  @click="requiresAdr = !requiresAdr"
                >
                  <Shield class="h-4 w-4" />
                  Опасный груз (ADR)
                </button>
                <button
                  type="button"
                  class="flex items-center gap-2 rounded-md border px-3 py-2 text-sm font-medium w-full transition-colors"
                  :class="thermograph ? 'bg-accent border-primary text-accent-foreground' : 'bg-white border-input text-foreground hover:bg-muted'"
                  @click="thermograph = !thermograph"
                >
                  <CheckSquare class="h-4 w-4" />
                  Термописец
                </button>
              </div>

              <div v-if="hasActiveFilters" class="flex justify-center pt-1">
                <Button variant="ghost" size="sm" @click="resetFilters">Сбросить фильтры</Button>
              </div>

            </div>
          </div>
        </div>
      </div>

      <!-- List -->
      <div>
        <LoadingSpinner v-if="isLoading" text="Загрузка транспорта..." />

        <ErrorBanner v-else-if="error" :message="error" @retry="loadItems" />

        <EmptyState
          v-else-if="items.length === 0"
          :icon="Truck"
          title="Транспорт не найден"
          :description="hasActiveFilters ? 'Нет транспорта по заданным фильтрам' : 'Зарегистрированный транспорт пока не добавлен'"
        />

        <div v-else class="space-y-3">
          <Card
            v-for="item in items"
            :key="item.id"
            interactive
            @click="router.push(`/organizations/${item.org_id}`)"
          >
            <CardContent class="p-4">

              <!-- Header: тип + рег. номер -->
              <div class="flex items-start justify-between gap-3 mb-3">
                <div class="flex items-center gap-2 flex-wrap">
                  <Truck class="h-4 w-4 text-muted-foreground shrink-0" />
                  <span class="text-sm font-medium">{{ formatVehicle(item) }}</span>
                  <VehicleVerifiedBadge :verified="item.verified" />
                </div>
                <span class="font-mono text-sm font-semibold shrink-0">{{ item.registration_number }}</span>
              </div>

              <!-- Brand/model + org -->
              <div class="flex items-center gap-4 mb-3 text-sm text-muted-foreground flex-wrap">
                <span v-if="item.brand || item.model">
                  {{ [item.brand, item.model].filter(Boolean).join(' ') }}
                </span>
                <span v-if="item.org_name" class="flex items-center gap-1">
                  <Building2 class="h-3.5 w-3.5 shrink-0" />
                  {{ item.org_name }}
                </span>
              </div>

              <!-- Характеристики -->
              <div class="flex items-center gap-3 flex-wrap text-xs text-muted-foreground pt-2.5 border-t">
                <span v-if="item.capacity" class="flex items-center gap-1">
                  <Weight class="h-3.5 w-3.5 shrink-0" />
                  {{ formatCapacity(item.capacity) }}
                </span>
                <span v-if="item.volume" class="flex items-center gap-1">
                  <Package class="h-3.5 w-3.5 shrink-0" />
                  {{ formatVolumeDisplay(item.volume) }}
                </span>
                <span v-if="formatTemp(item)" class="flex items-center gap-1">
                  <Thermometer class="h-3.5 w-3.5 shrink-0" />
                  {{ formatTemp(item) }}
                </span>
                <span
                  v-for="lt in (item.loading_types ?? [])"
                  :key="lt"
                  class="inline-flex items-center rounded-full bg-muted px-2 py-0.5"
                >
                  {{ loadingTypeLabels[lt as LoadingType] }}
                </span>
                <Badge v-if="item.requires_adr" variant="destructive" class="text-xs">ADR</Badge>
                <Badge v-if="item.thermograph" variant="secondary" class="text-xs">Термописец</Badge>

                <Button
                  v-if="item.verified && item.org_id !== auth.organizationId"
                  size="sm"
                  variant="outline"
                  class="ml-auto shrink-0"
                  @click.stop="openInviteDialog(item)"
                >
                  <Send class="h-3.5 w-3.5 mr-1.5" />
                  Предложить заявку
                </Button>
              </div>

            </CardContent>
          </Card>

          <!-- Infinite scroll sentinel -->
          <div ref="sentinelRef" class="h-16 flex items-center justify-center">
            <template v-if="isLoadingMore">
              <LoadingSpinner text="Загрузка..." />
            </template>
            <template v-else-if="!hasMore && items.length > 0">
              <span class="text-sm text-muted-foreground">Весь транспорт загружен ({{ items.length }})</span>
            </template>
          </div>
        </div>
      </div>

    </div>

    <InviteFreightDialog v-model:open="inviteDialogOpen" :vehicle="inviteVehicle" />
  </div>
</template>
