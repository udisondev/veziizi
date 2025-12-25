<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { notificationsApi } from '@/api/notifications'
import type { FreightRequestSubscription } from '@/types/notification'
import {
  cargoTypeOptions,
  cargoTypeLabels,
  bodyTypeOptions,
  bodyTypeLabels,
  type CargoType,
  type BodyType,
} from '@/types/freightRequest'
import { useGeo, getDisplayName, type Country } from '@/composables/useGeo'
import { useToast } from '@/components/ui/toast/use-toast'

// UI Components
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'

// Shared Components
import { PageHeader, LoadingSpinner, BackLink } from '@/components/shared'

// Icons
import { Bell, Filter, Info } from 'lucide-vue-next'

const { toast } = useToast()

// Geo composable для загрузки стран
const geo = useGeo()

const isLoading = ref(true)
const isSaving = ref(false)

// Локальное состояние фильтров
const originCountryIds = ref<number[]>([])
const destinationCountryIds = ref<number[]>([])
const cargoTypes = ref<string[]>([])
const minWeight = ref<number | undefined>()
const maxWeight = ref<number | undefined>()
const bodyTypes = ref<string[]>([])

// Toggle функции для chip-кнопок
function toggleCargoType(type: CargoType) {
  const index = cargoTypes.value.indexOf(type)
  if (index === -1) {
    cargoTypes.value.push(type)
  } else {
    cargoTypes.value.splice(index, 1)
  }
}

function toggleBodyType(type: BodyType) {
  const index = bodyTypes.value.indexOf(type)
  if (index === -1) {
    bodyTypes.value.push(type)
  } else {
    bodyTypes.value.splice(index, 1)
  }
}

function toggleOriginCountry(countryId: number) {
  const index = originCountryIds.value.indexOf(countryId)
  if (index === -1) {
    originCountryIds.value.push(countryId)
  } else {
    originCountryIds.value.splice(index, 1)
  }
}

function toggleDestinationCountry(countryId: number) {
  const index = destinationCountryIds.value.indexOf(countryId)
  if (index === -1) {
    destinationCountryIds.value.push(countryId)
  } else {
    destinationCountryIds.value.splice(index, 1)
  }
}

function getSelectedCountryNames(ids: number[]): string {
  return ids
    .map(id => {
      const country = geo.countries.value.find(c => c.id === id)
      return country ? getDisplayName(country) : ''
    })
    .filter(Boolean)
    .join(', ')
}

async function loadSubscription() {
  isLoading.value = true
  try {
    // Загружаем страны и подписку параллельно
    const [subscription] = await Promise.all([
      notificationsApi.getSubscription(),
      geo.fetchCountries(),
    ])

    // Заполняем локальные значения
    originCountryIds.value = subscription.origin_country_ids || []
    destinationCountryIds.value = subscription.destination_country_ids || []
    cargoTypes.value = subscription.cargo_types || []
    minWeight.value = subscription.min_weight
    maxWeight.value = subscription.max_weight
    bodyTypes.value = subscription.body_types || []
  } catch {
    toast({
      title: 'Ошибка',
      description: 'Не удалось загрузить настройки подписки',
      variant: 'destructive',
    })
  } finally {
    isLoading.value = false
  }
}

async function save() {
  isSaving.value = true
  try {
    const data: Partial<FreightRequestSubscription> = {
      origin_country_ids: originCountryIds.value.length ? originCountryIds.value : undefined,
      destination_country_ids: destinationCountryIds.value.length ? destinationCountryIds.value : undefined,
      cargo_types: cargoTypes.value.length ? cargoTypes.value : undefined,
      min_weight: minWeight.value,
      max_weight: maxWeight.value,
      body_types: bodyTypes.value.length ? bodyTypes.value : undefined,
      unsubscribed: false,
    }

    await notificationsApi.updateSubscription(data)
    toast({
      title: 'Подписка сохранена',
      description: 'Вы будете получать уведомления о подходящих заявках',
    })
  } catch {
    toast({
      title: 'Ошибка',
      description: 'Не удалось сохранить настройки подписки',
      variant: 'destructive',
    })
  } finally {
    isSaving.value = false
  }
}

function clearFilters() {
  originCountryIds.value = []
  destinationCountryIds.value = []
  cargoTypes.value = []
  minWeight.value = undefined
  maxWeight.value = undefined
  bodyTypes.value = []
}

onMounted(() => {
  loadSubscription()
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <BackLink to="/freight-requests" label="К заявкам" class="mb-4" />

    <PageHeader title="Подписка на заявки" class="mb-6" />

    <LoadingSpinner v-if="isLoading" text="Загрузка настроек..." />

    <template v-else>
      <!-- Info Card -->
      <Card class="mb-6 bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
        <CardContent class="flex items-start gap-3 pt-6">
          <Info class="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
          <div class="text-sm text-blue-800 dark:text-blue-200">
            <p class="font-medium mb-1">Как это работает</p>
            <p>
              Настройте фильтры, чтобы получать уведомления только о подходящих заявках.
              Если не выбрать ни один фильтр — вы будете получать уведомления о <strong>всех</strong> новых заявках.
            </p>
          </div>
        </CardContent>
      </Card>

      <!-- Filters Card -->
      <Card>
        <CardHeader>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Filter class="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle class="text-lg">Фильтры подписки</CardTitle>
              <CardDescription>
                Выберите параметры заявок, которые вам интересны
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-8">
          <!-- Origin Countries -->
          <div>
            <Label class="text-base font-medium mb-3 block">Страны отправления</Label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="country in geo.countries.value"
                :key="country.id"
                type="button"
                :class="[
                  'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
                  originCountryIds.includes(country.id)
                    ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                    : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700',
                ]"
                @click="toggleOriginCountry(country.id)"
              >
                {{ getDisplayName(country) }}
              </button>
            </div>
            <p v-if="originCountryIds.length" class="mt-2 text-sm text-muted-foreground">
              Выбрано: {{ getSelectedCountryNames(originCountryIds) }}
            </p>
            <p v-else class="mt-2 text-sm text-muted-foreground">
              Не выбрано — все страны
            </p>
          </div>

          <!-- Destination Countries -->
          <div>
            <Label class="text-base font-medium mb-3 block">Страны назначения</Label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="country in geo.countries.value"
                :key="country.id"
                type="button"
                :class="[
                  'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
                  destinationCountryIds.includes(country.id)
                    ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                    : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700',
                ]"
                @click="toggleDestinationCountry(country.id)"
              >
                {{ getDisplayName(country) }}
              </button>
            </div>
            <p v-if="destinationCountryIds.length" class="mt-2 text-sm text-muted-foreground">
              Выбрано: {{ getSelectedCountryNames(destinationCountryIds) }}
            </p>
            <p v-else class="mt-2 text-sm text-muted-foreground">
              Не выбрано — все страны
            </p>
          </div>

          <!-- Cargo Types -->
          <div>
            <Label class="text-base font-medium mb-3 block">Типы груза</Label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="option in cargoTypeOptions"
                :key="option.value"
                type="button"
                :class="[
                  'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
                  cargoTypes.includes(option.value)
                    ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                    : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700',
                ]"
                @click="toggleCargoType(option.value)"
              >
                {{ option.label }}
              </button>
            </div>
            <p v-if="cargoTypes.length" class="mt-2 text-sm text-muted-foreground">
              Выбрано: {{ cargoTypes.map(t => cargoTypeLabels[t as CargoType]).join(', ') }}
            </p>
            <p v-else class="mt-2 text-sm text-muted-foreground">
              Не выбрано — все типы груза
            </p>
          </div>

          <!-- Weight Range -->
          <div>
            <Label class="text-base font-medium mb-3 block">Вес груза, т</Label>
            <div class="flex items-center gap-3 max-w-md">
              <div class="flex-1">
                <Input
                  type="number"
                  v-model.number="minWeight"
                  placeholder="от"
                  min="0"
                  step="0.1"
                />
              </div>
              <span class="text-muted-foreground">—</span>
              <div class="flex-1">
                <Input
                  type="number"
                  v-model.number="maxWeight"
                  placeholder="до"
                  min="0"
                  step="0.1"
                />
              </div>
            </div>
            <p v-if="minWeight || maxWeight" class="mt-2 text-sm text-muted-foreground">
              {{ minWeight ? `от ${minWeight} т` : '' }}{{ minWeight && maxWeight ? ' — ' : '' }}{{ maxWeight ? `до ${maxWeight} т` : '' }}
            </p>
            <p v-else class="mt-2 text-sm text-muted-foreground">
              Не указано — любой вес
            </p>
          </div>

          <!-- Body Types -->
          <div>
            <Label class="text-base font-medium mb-3 block">Типы кузова</Label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="option in bodyTypeOptions"
                :key="option.value"
                type="button"
                :class="[
                  'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
                  bodyTypes.includes(option.value)
                    ? 'bg-blue-100 border-blue-500 text-blue-700 dark:bg-blue-900 dark:border-blue-400 dark:text-blue-200'
                    : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700',
                ]"
                @click="toggleBodyType(option.value)"
              >
                {{ option.label }}
              </button>
            </div>
            <p v-if="bodyTypes.length" class="mt-2 text-sm text-muted-foreground">
              Выбрано: {{ bodyTypes.map(t => bodyTypeLabels[t as BodyType]).join(', ') }}
            </p>
            <p v-else class="mt-2 text-sm text-muted-foreground">
              Не выбрано — все типы кузова
            </p>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-4 pt-4 border-t">
            <Button :disabled="isSaving" @click="save">
              <Bell class="mr-2 h-4 w-4" />
              {{ isSaving ? 'Сохранение...' : 'Сохранить подписку' }}
            </Button>
            <Button variant="outline" @click="clearFilters">
              Сбросить фильтры
            </Button>
          </div>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
