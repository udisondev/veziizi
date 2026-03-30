<script setup lang="ts">
import { ref, watch, onMounted, computed, nextTick } from 'vue'
import { useGeo, type City, searchCities } from '@/composables/useGeo'
import { useCityDropdown } from '@/composables/useCityDropdown'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Trash2, GripVertical } from 'lucide-vue-next'
import type { AcceptableValue } from 'reka-ui'

interface RoutePointData {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
}

interface Props {
  point: RoutePointData
  index: number
  canRemove?: boolean
  canMove?: boolean
}

interface Emits {
  (e: 'update', updates: Partial<RoutePointData>): void
  (e: 'remove'): void
}

const props = withDefaults(defineProps<Props>(), {
  canRemove: true,
  canMove: true,
})

const emit = defineEmits<Emits>()

const {
  countries,
  isLoadingCountries,
  fetchCountries,
} = useGeo()

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
  <div class="bg-white border rounded-lg p-4 shadow-sm border-l-4 border-l-primary">
    <div class="flex items-start gap-3">
      <!-- Drag handle -->
      <div
        v-if="canMove"
        class="drag-handle cursor-move text-muted-foreground hover:text-foreground pt-1"
      >
        <GripVertical class="h-5 w-5" />
      </div>

      <!-- Point number -->
      <div class="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-sm font-medium shrink-0 mt-1">
        {{ index + 1 }}
      </div>

      <!-- Country & City selection -->
      <div class="flex-1 space-y-3">
        <!-- Country Select -->
        <div>
          <Label class="text-sm font-medium text-gray-700 mb-1">Страна *</Label>
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
              <!-- Search input inside dropdown -->
              <div class="px-2 py-1 bg-white border-b z-10">
                <input
                  v-model="countrySearch"
                  type="text"
                  class="w-full px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-blue-500"
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
                  <span v-if="country.name_ru && country.name !== country.name_ru" class="text-gray-500 text-xs ml-1">
                    ({{ country.name }})
                  </span>
                </SelectItem>
                <div v-if="filteredCountries.length === 0" class="px-2 py-3 text-sm text-gray-500 text-center">
                  Страны не найдены
                </div>
              </div>
            </SelectContent>
          </Select>
        </div>

        <!-- City Autocomplete -->
        <div v-if="point.countryId">
          <Label class="text-sm font-medium text-gray-700 mb-1">Город (опционально)</Label>
          <div class="relative">
            <input
              ref="cityInputRef"
              type="text"
              :value="citySearch"
              class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 text-sm"
              placeholder="Любой город"
              autocomplete="off"
              @input="handleCityInput"
              @keydown="handleCityKeydown"
            />

            <!-- Loading indicator -->
            <div v-if="isLoadingCities" class="absolute right-3 top-1/2 -translate-y-1/2">
              <svg
                class="animate-spin h-5 w-5 text-gray-400"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                />
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
            </div>

            <!-- Selected indicator -->
            <div
              v-else-if="point.cityId"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-green-500"
              title="Город выбран"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fill-rule="evenodd"
                  d="M5.05 4.05a7 7 0 119.9 9.9L10 18.9l-4.95-4.95a7 7 0 010-9.9zM10 11a2 2 0 100-4 2 2 0 000 4z"
                  clip-rule="evenodd"
                />
              </svg>
            </div>

            <!-- City dropdown -->
            <div
              v-if="isCityDropdownOpen && cities.length > 0"
              ref="cityDropdownRef"
              :class="[
                'absolute z-50 w-full bg-white border border-gray-200 rounded-md shadow-lg max-h-60 overflow-auto',
                dropdownDirection === 'down' ? 'top-full mt-1' : 'bottom-full mb-1'
              ]"
            >
              <button
                v-for="(city, idx) in cities"
                :key="city.id"
                type="button"
                :class="[
                  'w-full px-3 py-2 text-left text-sm hover:bg-gray-100',
                  idx === highlightedIndex ? 'bg-blue-50' : '',
                ]"
                @click="handleCitySelect(city)"
                @mouseenter="highlightedIndex = idx"
              >
                <div class="font-medium text-gray-900">
                  {{ city.name_ru || city.name }}
                </div>
                <div v-if="city.name_ru && city.name !== city.name_ru" class="text-gray-500 text-xs">
                  {{ city.name }}
                </div>
              </button>
            </div>

            <!-- No results -->
            <div
              v-if="isCityDropdownOpen && cities.length === 0 && !isLoadingCities && citySearch.length > 0"
              :class="[
                'absolute z-50 w-full bg-white border border-gray-200 rounded-md shadow-lg p-3 text-sm text-gray-500 text-center',
                dropdownDirection === 'down' ? 'top-full mt-1' : 'bottom-full mb-1'
              ]"
            >
              Города не найдены
            </div>
          </div>
        </div>
      </div>

      <!-- Remove button -->
      <button
        v-if="canRemove"
        type="button"
        class="text-muted-foreground hover:text-destructive transition-colors p-1"
        title="Удалить точку"
        @click="$emit('remove')"
      >
        <Trash2 class="h-5 w-5" />
      </button>
    </div>
  </div>
</template>
