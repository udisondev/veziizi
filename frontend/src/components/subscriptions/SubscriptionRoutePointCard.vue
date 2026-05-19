<script setup lang="ts">
import { ref, watch, watchEffect, onMounted, computed, nextTick } from 'vue'
import { useGeo, type City, searchCities } from '@/composables/useGeo'
import { useCityDropdown } from '@/composables/useCityDropdown'
import { useBreakpoint } from '@/composables/useBreakpoint'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import InputVue from '@/components/ui/input/Input.vue'
import BottomSheet from '@/components/shared/BottomSheet.vue'
import { Trash2, GripVertical, Navigation, MapPin } from 'lucide-vue-next'
import { Tooltip } from '@/components/ui/tooltip'
import type { AcceptableValue } from 'reka-ui'
import type { RoutePointData } from '@/types/routePoint'

interface Props {
  point: RoutePointData
  index: number
  total?: number
  canRemove?: boolean
  canMove?: boolean
  plain?: boolean
}

interface Emits {
  (e: 'update', updates: Partial<RoutePointData>): void
  (e: 'remove'): void
}

const props = withDefaults(defineProps<Props>(), {
  total: 1,
  canRemove: true,
  canMove: true,
  plain: false,
})

const emit = defineEmits<Emits>()

const isFirst  = computed(() => props.index === 0)
const isLast   = computed(() => props.index === props.total - 1)
const isMiddle = computed(() => props.total > 2 && !isFirst.value && !isLast.value)
const showOriginBtn      = computed(() => props.total === 1 || isFirst.value)
const showDestinationBtn = computed(() => props.total === 1 || isLast.value)

// Сбрасываем роль у средних точек автоматически
watch(isMiddle, (middle) => {
  if (middle && props.point.role && props.point.role !== 'any') {
    emit('update', { role: 'any' })
  }
}, { immediate: true })

const {
  countries,
  isLoadingCountries,
  fetchCountries,
} = useGeo()

const { isMobile, isBelowLg } = useBreakpoint()

// Mobile country sheet
const countrySheetOpen = ref(false)
const countrySheetSearch = ref('')

const filteredSheetCountries = computed(() => {
  if (!countrySheetSearch.value) return countries.value
  const search = countrySheetSearch.value.toLowerCase()
  return countries.value.filter(
    (c) =>
      c.name.toLowerCase().includes(search) ||
      (c.name_ru && c.name_ru.toLowerCase().includes(search)) ||
      c.iso2.toLowerCase().includes(search) ||
      (c.native_name && c.native_name.toLowerCase().includes(search))
  )
})

function handleSheetCountrySelect(countryId: number) {
  const country = countries.value.find(c => c.id === countryId)
  emit('update', {
    countryId,
    countryName: country?.name_ru || country?.name,
    cityId: undefined,
    cityName: undefined,
  })
  citySearch.value = ''
  cities.value = []
  countrySheetOpen.value = false
  countrySheetSearch.value = ''
}

// Local state for city search
const citySearch = ref('')
const cities = ref<City[]>([])
const isLoadingCities = ref(false)
const isCityDropdownOpen = ref(false)

const {
  highlightedIndex,
  cityInputRef,
  cityDropdownRef,
  handleKeydown: handleCityKeydown,
  resetHighlight,
} = useCityDropdown({
  cities,
  isOpen: isCityDropdownOpen,
  onSelect: (city) => handleCitySelect(city),
  onClose: () => { isCityDropdownOpen.value = false },
})

// Ref to the Input component instance — used to sync cityInputRef with the inner DOM element
const cityInputComponent = ref<InstanceType<typeof InputVue> | null>(null)
watchEffect(() => {
  cityInputRef.value = cityInputComponent.value?.inputEl ?? null
})

// Template refs used in template via ref="..." (vue-tsc false positive workaround)
void cityDropdownRef

// Dropdown direction (up or down)
const dropdownDirection = ref<'down' | 'up'>('down')

function calculateDropdownDirection() {
  if (!cityInputRef.value) return

  const inputRect = cityInputRef.value.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const dropdownMaxHeight = 240 // max-h-60 = 15rem = 240px

  const spaceBelow = viewportHeight - inputRect.bottom
  const spaceAbove = inputRect.top

  // Открываем вверх если снизу недостаточно места, а сверху больше
  if (spaceBelow < dropdownMaxHeight && spaceAbove > spaceBelow) {
    dropdownDirection.value = 'up'
  } else {
    dropdownDirection.value = 'down'
  }
}

// Recalculate direction when dropdown opens
watch(isCityDropdownOpen, (isOpen) => {
  if (isOpen) {
    nextTick(() => calculateDropdownDirection())
  }
})

// Filter countries by search
const countrySearch = ref('')
const filteredCountries = computed(() => {
  if (!countrySearch.value) return countries.value
  const search = countrySearch.value.toLowerCase()
  return countries.value.filter(
    (c) =>
      c.name.toLowerCase().includes(search) ||
      (c.name_ru && c.name_ru.toLowerCase().includes(search)) ||
      c.iso2.toLowerCase().includes(search) ||
      (c.native_name && c.native_name.toLowerCase().includes(search))
  )
})

// Initialize city search with current city name
watch(
  () => props.point.cityName,
  (newVal) => {
    if (newVal && !citySearch.value) {
      citySearch.value = newVal
    }
  },
  { immediate: true }
)

// Handle country change
function handleCountryChange(value: AcceptableValue) {
  if (value === undefined || value === null) return
  const countryId = typeof value === 'string' ? parseInt(value, 10) : Number(value)
  if (!isNaN(countryId)) {
    const country = countries.value.find(c => c.id === countryId)
    emit('update', {
      countryId,
      countryName: country?.name_ru || country?.name,
      cityId: undefined,
      cityName: undefined,
    })
    citySearch.value = ''
    cities.value = []
  }
}

// Handle city search
async function handleCityInput(event: Event) {
  const value = (event.target as HTMLInputElement).value
  citySearch.value = value
  resetHighlight()

  if (!props.point.countryId || value.length < 1) {
    cities.value = []
    isCityDropdownOpen.value = false
    // Clear city selection when input is empty
    if (props.point.cityId !== undefined) {
      emit('update', { cityId: undefined, cityName: undefined })
    }
    return
  }

  isLoadingCities.value = true
  try {
    const results = await searchCities(props.point.countryId, value, 10)
    cities.value = results
    isCityDropdownOpen.value = results.length > 0
  } catch {
    cities.value = []
  } finally {
    isLoadingCities.value = false
  }
}

// Handle city selection
function handleCitySelect(city: City) {
  emit('update', {
    cityId: city.id,
    cityName: city.name_ru || city.name,
  })
  citySearch.value = city.name_ru || city.name
  isCityDropdownOpen.value = false
}

onMounted(async () => {
  await fetchCountries()
})
</script>

<template>
  <div :class="plain ? '' : 'bg-white border rounded-lg p-4 shadow-sm border-l-4 border-l-primary flex items-start gap-3'">

    <!-- plain: header row (drag + number + remove) above fields -->
    <template v-if="plain">
      <div class="flex items-center gap-2 mb-3">
        <div v-if="canMove" class="drag-handle cursor-move text-muted-foreground hover:text-foreground">
          <GripVertical class="h-5 w-5" />
        </div>
        <div
          class="flex items-center justify-center w-6 h-6 rounded-full text-white text-sm font-medium shrink-0"
          :class="point.role === 'destination' ? 'bg-emerald-500' : 'bg-primary'"
        >
          {{ index + 1 }}
        </div>
        <div v-if="!isMiddle" class="flex gap-1 ml-1">
          <!-- Desktop: иконки + тултипы -->
          <template v-if="!isBelowLg">
            <Tooltip v-if="showOriginBtn" text="Начало маршрута" side="top">
              <button
                type="button"
                class="flex justify-center rounded-md border p-1.5 transition-colors"
                :class="point.role === 'origin' ? 'bg-primary border-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-muted/50'"
                @click="emit('update', { role: point.role === 'origin' ? 'any' : 'origin' })"
              >
                <Navigation class="h-3.5 w-3.5" />
              </button>
            </Tooltip>
            <Tooltip v-if="showDestinationBtn" text="Конец маршрута" side="top">
              <button
                type="button"
                class="flex justify-center rounded-md border p-1.5 transition-colors"
                :class="point.role === 'destination' ? 'bg-emerald-500 border-emerald-500 text-white' : 'border-border text-muted-foreground hover:bg-muted/50'"
                @click="emit('update', { role: point.role === 'destination' ? 'any' : 'destination' })"
              >
                <MapPin class="h-3.5 w-3.5" />
              </button>
            </Tooltip>
          </template>
          <!-- Mobile/Tablet: текстовые кнопки -->
          <template v-else>
            <button
              v-if="showOriginBtn"
              type="button"
              class="rounded-md border px-2 py-1 text-xs font-medium transition-colors"
              :class="point.role === 'origin' ? 'bg-primary border-primary text-primary-foreground' : 'border-border text-muted-foreground hover:bg-muted/50'"
              @click="emit('update', { role: point.role === 'origin' ? 'any' : 'origin' })"
            >
              Начало маршрута
            </button>
            <button
              v-if="showDestinationBtn"
              type="button"
              class="rounded-md border px-2 py-1 text-xs font-medium transition-colors"
              :class="point.role === 'destination' ? 'bg-emerald-500 border-emerald-500 text-white' : 'border-border text-muted-foreground hover:bg-muted/50'"
              @click="emit('update', { role: point.role === 'destination' ? 'any' : 'destination' })"
            >
              Конец маршрута
            </button>
          </template>
        </div>
        <button v-if="canRemove" type="button" class="ml-auto text-muted-foreground hover:text-destructive transition-colors p-1" title="Удалить точку" @click="emit('remove')">
          <Trash2 class="h-5 w-5" />
        </button>
      </div>
    </template>

    <!-- full: drag handle and number on the left (flex children) -->
    <template v-else>
      <div v-if="canMove" class="drag-handle cursor-move text-muted-foreground hover:text-foreground pt-1">
        <GripVertical class="h-5 w-5" />
      </div>
      <div class="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-sm font-medium shrink-0 mt-1">
        {{ index + 1 }}
      </div>
    </template>

    <!-- shared: country + city fields -->
    <div :class="plain ? 'space-y-3' : 'flex-1 space-y-3'">
      <!-- Country Select -->
      <div>
        <Label class="text-foreground">Страна *</Label>

        <!-- Desktop -->
        <template v-if="!isMobile">
          <Select
            :model-value="point.countryId?.toString()"
            :disabled="isLoadingCountries"
            @update:model-value="handleCountryChange"
          >
            <SelectTrigger class="w-full">
              <SelectValue placeholder="Выберите страну" />
            </SelectTrigger>
            <SelectContent
              class="!w-[var(--reka-select-trigger-width)] !min-w-0 !max-h-[50vh] overflow-hidden"
              :side-offset="4"
            >
              <div class="px-2 py-1 bg-white border-b z-10">
                <Input
                  v-model="countrySearch"
                  type="text"
                  placeholder="Поиск страны..."
                  @click.stop
                  @keydown.stop
                />
              </div>
              <div class="max-h-[40vh] overflow-y-auto">
                <SelectItem
                  v-for="country in filteredCountries"
                  :key="country.id"
                  :value="country.id.toString()"
                >
                  {{ country.name_ru || country.name }}
                  <span v-if="country.name_ru && country.name !== country.name_ru" class="text-muted-foreground text-xs ml-1">
                    ({{ country.name }})
                  </span>
                </SelectItem>
                <div v-if="filteredCountries.length === 0" class="px-2 py-3 text-sm text-muted-foreground text-center">
                  Страны не найдены
                </div>
              </div>
            </SelectContent>
          </Select>
        </template>

        <!-- Mobile -->
        <template v-else>
          <button
            type="button"
            :disabled="isLoadingCountries"
            class="appearance-none block w-full px-3 py-2.5 border border-input rounded-md text-base text-left bg-white disabled:bg-muted disabled:text-muted-foreground"
            @click="countrySheetOpen = true"
          >
            <span v-if="point.countryId" class="text-foreground">{{ point.countryName }}</span>
            <span v-else class="text-muted-foreground">Выберите страну</span>
          </button>

          <BottomSheet v-model="countrySheetOpen" label="Страна">
            <div class="px-4 py-2 border-b flex-shrink-0">
              <Input
                v-model="countrySheetSearch"
                type="text"
                placeholder="Поиск страны..."
              />
            </div>
            <div class="overflow-y-auto flex-1">
              <button
                v-for="country in filteredSheetCountries"
                :key="country.id"
                type="button"
                :class="[
                  'w-full px-4 py-3.5 text-left text-base border-b border-border/30 active:bg-muted',
                  country.id === point.countryId ? 'bg-accent text-accent-foreground font-medium' : 'text-foreground',
                ]"
                @click="handleSheetCountrySelect(country.id)"
              >
                {{ country.name_ru || country.name }}
                <span v-if="country.name_ru && country.name !== country.name_ru" class="text-muted-foreground text-xs ml-1">
                  ({{ country.name }})
                </span>
              </button>
              <div v-if="filteredSheetCountries.length === 0" class="px-4 py-6 text-sm text-muted-foreground text-center">
                Страны не найдены
              </div>
            </div>
          </BottomSheet>
        </template>
      </div>

      <!-- City Autocomplete -->
      <div v-if="point.countryId">
        <Label class="text-foreground">Город (опционально)</Label>
        <div class="relative">
          <Input
            ref="cityInputComponent"
            :model-value="citySearch"
            placeholder="Любой город"
            autocomplete="off"
            @input="handleCityInput"
            @keydown="handleCityKeydown"
          />

          <!-- Loading indicator -->
          <div v-if="isLoadingCities" class="absolute right-3 top-1/2 -translate-y-1/2">
            <svg class="animate-spin h-5 w-5 text-muted-foreground" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
          </div>

          <!-- Selected indicator -->
          <div v-else-if="point.cityId" class="absolute right-3 top-1/2 -translate-y-1/2 text-success" title="Город выбран">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M5.05 4.05a7 7 0 119.9 9.9L10 18.9l-4.95-4.95a7 7 0 010-9.9zM10 11a2 2 0 100-4 2 2 0 000 4z" clip-rule="evenodd" />
            </svg>
          </div>

          <!-- City dropdown -->
          <div
            v-if="isCityDropdownOpen && cities.length > 0"
            ref="cityDropdownRef"
            :class="[
              'absolute z-50 w-full bg-white border border-border rounded-md shadow-lg max-h-60 overflow-auto',
              dropdownDirection === 'down' ? 'top-full mt-1' : 'bottom-full mb-1'
            ]"
          >
            <button
              v-for="(city, idx) in cities"
              :key="city.id"
              type="button"
              :class="['w-full px-3 py-2.5 text-left text-base hover:bg-muted', idx === highlightedIndex ? 'bg-accent' : '']"
              @click="handleCitySelect(city)"
              @mouseenter="highlightedIndex = idx"
            >
              <div class="font-medium text-foreground">{{ city.name_ru || city.name }}</div>
              <div v-if="city.name_ru && city.name !== city.name_ru" class="text-muted-foreground text-xs">{{ city.name }}</div>
            </button>
          </div>

          <!-- No results -->
          <div
            v-if="isCityDropdownOpen && cities.length === 0 && !isLoadingCities && citySearch.length > 0"
            :class="[
              'absolute z-50 w-full bg-white border border-border rounded-md shadow-lg p-3 text-sm text-muted-foreground text-center',
              dropdownDirection === 'down' ? 'top-full mt-1' : 'bottom-full mb-1'
            ]"
          >
            Города не найдены
          </div>
        </div>
      </div>
    </div>

    <!-- full: remove button on the right (flex child) -->
    <button v-if="!plain && canRemove" type="button" class="text-muted-foreground hover:text-destructive transition-colors p-1" title="Удалить точку" @click="emit('remove')">
      <Trash2 class="h-5 w-5" />
    </button>

  </div>
</template>
