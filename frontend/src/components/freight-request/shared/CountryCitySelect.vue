<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useGeo, type City } from '@/composables/useGeo'
import type { Coordinates } from '@/types/freightRequest'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import type { AcceptableValue } from 'reka-ui'

interface Props {
  countryId?: number
  cityId?: number
  error?: string | null
  disabled?: boolean
}

interface Emits {
  (e: 'update:countryId', value: number | undefined): void
  (e: 'update:cityId', value: number | undefined): void
  (e: 'update:coordinates', value: Coordinates | undefined): void
  (e: 'update:displayAddress', value: string): void
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
})

const emit = defineEmits<Emits>()

const {
  countries,
  cities,
  isLoadingCountries,
  isLoadingCities,
  selectedCountryId,
  selectedCityId,
  selectedCity,
  citySearch,
  isCityDropdownOpen,
  coordinates,
  fetchCountries,
  setCitySearch,
  selectCountry,
  selectCity,
  loadCityById,
  closeCityDropdown,
} = useGeo()

const cityInputRef = ref<HTMLInputElement | null>(null)
const cityDropdownRef = ref<HTMLDivElement | null>(null)
const highlightedIndex = ref(-1)

// Filter countries by search (English, Russian, and ISO codes)
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

// Display address for legacy compatibility (uses Russian names)
const displayAddress = computed(() => {
  if (selectedCity.value) {
    const cityName = selectedCity.value.name_ru || selectedCity.value.name
    const countryName = selectedCity.value.country_name_ru || selectedCity.value.country_name
    const parts = [cityName]
    if (countryName) {
      parts.push(countryName)
    }
    return parts.join(', ')
  }
  return ''
})

// Country trigger class
const countryTriggerClass = computed(() => {
  const classes = ['w-full']
  if (props.error && !selectedCountryId.value) {
    classes.push('border-red-300')
  }
  return classes.join(' ')
})

// Sync props with internal state
watch(
  () => props.countryId,
  (newVal) => {
    if (newVal !== selectedCountryId.value) {
      selectedCountryId.value = newVal
    }
  },
  { immediate: true }
)

watch(
  () => props.cityId,
  async (newVal) => {
    if (newVal && newVal !== selectedCityId.value) {
      await loadCityById(newVal)
    }
  },
  { immediate: true }
)

// Emit changes
watch(selectedCountryId, (newVal) => {
  emit('update:countryId', newVal)
})

watch(selectedCityId, (newVal) => {
  emit('update:cityId', newVal)
})

watch(coordinates, (newVal) => {
  emit('update:coordinates', newVal)
})

watch(displayAddress, (newVal) => {
  emit('update:displayAddress', newVal)
})

// Handle country change
function handleCountryChange(value: AcceptableValue) {
  if (value === undefined || value === null) return
  const countryId = typeof value === 'string' ? parseInt(value, 10) : Number(value)
  if (!isNaN(countryId)) {
    selectCountry(countryId)
    highlightedIndex.value = -1
  }
}

// Handle city input
function handleCityInput(event: Event) {
  const value = (event.target as HTMLInputElement).value
  setCitySearch(value)
  highlightedIndex.value = -1
}

// Handle city selection
function handleCitySelect(city: City) {
  selectCity(city)
}

// Keyboard navigation for city dropdown
function handleCityKeydown(event: KeyboardEvent) {
  if (!isCityDropdownOpen.value || cities.value.length === 0) return

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      highlightedIndex.value = Math.min(highlightedIndex.value + 1, cities.value.length - 1)
      break
    case 'ArrowUp':
      event.preventDefault()
      highlightedIndex.value = Math.max(highlightedIndex.value - 1, 0)
      break
    case 'Enter':
      event.preventDefault()
      const selected = cities.value[highlightedIndex.value]
      if (highlightedIndex.value >= 0 && selected) {
        handleCitySelect(selected)
      }
      break
    case 'Escape':
      closeCityDropdown()
      break
  }
}

// Click outside handler
function handleClickOutside(event: MouseEvent) {
  const target = event.target as Node
  if (
    cityInputRef.value &&
    !cityInputRef.value.contains(target) &&
    cityDropdownRef.value &&
    !cityDropdownRef.value.contains(target)
  ) {
    closeCityDropdown()
  }
}

onMounted(async () => {
  await fetchCountries()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div class="space-y-3">
    <!-- Country Select -->
    <div>
      <Label class="block text-sm font-medium text-gray-700 mb-1">Страна</Label>
      <Select
        :model-value="selectedCountryId?.toString()"
        :disabled="disabled || isLoadingCountries"
        @update:model-value="handleCountryChange"
      >
        <SelectTrigger :class="countryTriggerClass">
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
    <div v-if="selectedCountryId">
      <Label class="block text-sm font-medium text-gray-700 mb-1">Город</Label>
      <div class="relative">
        <input
          ref="cityInputRef"
          type="text"
          :value="citySearch"
          :disabled="disabled"
          :class="[
            'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500',
            error && !selectedCityId ? 'border-red-300' : 'border-gray-300',
            disabled ? 'bg-gray-100 cursor-not-allowed' : '',
          ]"
          placeholder="Начните вводить город..."
          autocomplete="off"
          @input="handleCityInput"
          @keydown="handleCityKeydown"
          @focus="setCitySearch(citySearch)"
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
          v-else-if="selectedCityId"
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
          class="absolute z-50 w-full mt-1 bg-white border border-gray-200 rounded-md shadow-lg max-h-60 overflow-auto"
        >
          <button
            v-for="(city, index) in cities"
            :key="city.id"
            type="button"
            :class="[
              'w-full px-3 py-2 text-left text-sm hover:bg-gray-100',
              index === highlightedIndex ? 'bg-blue-50' : '',
            ]"
            @click="handleCitySelect(city)"
            @mouseenter="highlightedIndex = index"
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
          class="absolute z-50 w-full mt-1 bg-white border border-gray-200 rounded-md shadow-lg p-3 text-sm text-gray-500 text-center"
        >
          Города не найдены
        </div>
      </div>
    </div>

    <!-- Error message -->
    <p v-if="error" class="text-sm text-red-600">
      {{ error }}
    </p>
  </div>
</template>
