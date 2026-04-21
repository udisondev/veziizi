<script setup lang="ts">
import { ref, watch, onMounted, computed, nextTick } from 'vue'
import { useGeo, type City } from '@/composables/useGeo'
import { useCityDropdown } from '@/composables/useCityDropdown'
import type { Coordinates } from '@/types/freightRequest'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import type { AcceptableValue } from 'reka-ui'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BottomSheet from '@/components/shared/BottomSheet.vue'

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

const {
  highlightedIndex,
  cityInputRef,
  cityDropdownRef,
  handleKeydown: handleCityKeydown,
  resetHighlight,
} = useCityDropdown({
  cities,
  isOpen: isCityDropdownOpen,
  onSelect: (city) => selectCity(city),
  onClose: () => closeCityDropdown(),
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

// Recalculate direction when dropdown opens (desktop only)
watch(isCityDropdownOpen, (isOpen) => {
  if (isOpen && !isMobile) {
    nextTick(() => calculateDropdownDirection())
  }
})

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
  resetHighlight()
}

// Handle city selection
function handleCitySelect(city: City) {
  selectCity(city)
}

onMounted(async () => {
  await fetchCountries()
})

const { isMobile } = useBreakpoint()

const countrySheetOpen = ref(false)
const citySheetOpen = ref(false)

const selectedCountryName = computed(() => {
  if (!selectedCountryId.value) return null
  const country = countries.value.find(c => c.id === selectedCountryId.value)
  return country ? (country.name_ru || country.name) : null
})

function openCountrySheet() {
  countrySearch.value = ''
  countrySheetOpen.value = true
}

function handleCountrySheetSelect(value: AcceptableValue) {
  handleCountryChange(value)
  countrySheetOpen.value = false
}

function openCitySheet() {
  citySheetOpen.value = true
  nextTick(() => {
    cityInputRef.value?.focus()
  })
}

watch(citySheetOpen, (open) => {
  if (!open) closeCityDropdown()
})

function handleCitySheetSelect(city: City) {
  handleCitySelect(city)
  citySheetOpen.value = false
}
</script>

<template>
  <div>
  <div class="space-y-3">
    <!-- Country Select -->
    <div>
      <Label class="text-sm text-foreground">Страна</Label>

      <!-- Desktop -->
      <template v-if="!isMobile">
        <Select
          :model-value="selectedCountryId?.toString()"
          :disabled="disabled || isLoadingCountries"
          @update:model-value="handleCountryChange"
        >
          <SelectTrigger :class="error && !selectedCountryId ? 'w-full border-destructive' : 'w-full'">
            <SelectValue placeholder="Выберите страну" />
          </SelectTrigger>
          <SelectContent
            class="!w-[var(--reka-select-trigger-width)] !min-w-0 !max-h-[50vh] overflow-hidden"
            :side-offset="4"
          >
            <div class="px-2 py-2 bg-white border-b z-10">
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

      <!-- Mobile: триггер + bottom sheet -->
      <template v-else>
        <button
          type="button"
          :disabled="disabled || isLoadingCountries"
          :class="[
            'appearance-none block w-full px-3 py-2 border rounded-md text-sm text-left',
            error && !selectedCountryId ? 'border-destructive' : 'border-input',
            disabled ? 'bg-muted text-muted-foreground' : 'bg-white text-foreground',
          ]"
          @click="openCountrySheet"
        >
          <span v-if="selectedCountryName">{{ selectedCountryName }}</span>
          <span v-else class="text-muted-foreground">Выберите страну</span>
        </button>

        <BottomSheet v-model="countrySheetOpen" label="Выберите страну">
          <div class="px-4 pt-3 pb-2 border-b">
            <Input
              v-model="countrySearch"
              type="text"
              placeholder="Поиск страны..."
            />
          </div>
          <div class="overflow-y-auto flex-1">
            <button
              v-for="country in filteredCountries"
              :key="country.id"
              type="button"
              class="w-full px-4 py-3 text-left text-sm border-b border-border/30 active:bg-muted"
              :class="country.id === selectedCountryId ? 'bg-accent text-accent-foreground font-medium' : 'text-foreground'"
              @click="handleCountrySheetSelect(country.id)"
            >
              {{ country.name_ru || country.name }}
              <span v-if="country.name_ru && country.name !== country.name_ru" class="text-muted-foreground text-xs ml-1">
                ({{ country.name }})
              </span>
            </button>
            <div v-if="filteredCountries.length === 0" class="px-4 py-6 text-sm text-muted-foreground text-center">
              Страны не найдены
            </div>
          </div>
        </BottomSheet>
      </template>
    </div>

    <!-- City Autocomplete -->
    <div v-if="selectedCountryId">
      <Label class="text-sm text-foreground">Город</Label>

      <!-- Desktop -->
      <template v-if="!isMobile">
        <div class="relative">
          <input
            ref="cityInputRef"
            type="text"
            :value="citySearch"
            :disabled="disabled"
            :class="[
              'appearance-none block w-full px-3 py-2.5 text-base border rounded-lg focus:outline-none focus:ring-2 focus:ring-ring',
              error && !selectedCityId ? 'border-destructive' : 'border-input',
              disabled ? 'bg-muted cursor-not-allowed' : 'bg-white',
            ]"
            placeholder="Начните вводить город..."
            autocomplete="off"
            @input="handleCityInput"
            @keydown="handleCityKeydown"
            @focus="setCitySearch(citySearch)"
          />

          <div v-if="isLoadingCities" class="absolute right-3 top-1/2 -translate-y-1/2">
            <svg class="animate-spin h-5 w-5 text-muted-foreground" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
          </div>
          <div v-else-if="selectedCityId" class="absolute right-3 top-1/2 -translate-y-1/2 text-success" title="Город выбран">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M5.05 4.05a7 7 0 119.9 9.9L10 18.9l-4.95-4.95a7 7 0 010-9.9zM10 11a2 2 0 100-4 2 2 0 000 4z" clip-rule="evenodd" />
            </svg>
          </div>

          <div
            v-if="isCityDropdownOpen && cities.length > 0"
            ref="cityDropdownRef"
            data-tutorial-popup
            :class="[
              'absolute z-50 w-full bg-white border border-border rounded-md shadow-lg max-h-60 overflow-auto',
              dropdownDirection === 'down' ? 'top-full mt-1' : 'bottom-full mb-1'
            ]"
          >
            <button
              v-for="(city, index) in cities"
              :key="city.id"
              type="button"
              :class="['w-full px-3 py-2 text-left text-sm hover:bg-muted', index === highlightedIndex ? 'bg-accent' : '']"
              @click="handleCitySelect(city)"
              @mouseenter="highlightedIndex = index"
            >
              <div class="font-medium text-foreground">{{ city.name_ru || city.name }}</div>
              <div v-if="city.name_ru && city.name !== city.name_ru" class="text-muted-foreground text-xs">{{ city.name }}</div>
            </button>
          </div>

          <div
            v-if="isCityDropdownOpen && cities.length === 0 && !isLoadingCities && citySearch.length > 0"
            data-tutorial-popup
            :class="[
              'absolute z-50 w-full bg-white border border-border rounded-md shadow-lg p-3 text-sm text-muted-foreground text-center',
              dropdownDirection === 'down' ? 'top-full mt-1' : 'bottom-full mb-1'
            ]"
          >
            Города не найдены
          </div>
        </div>
      </template>

      <!-- Mobile: триггер + bottom sheet -->
      <template v-else>
        <button
          type="button"
          :disabled="disabled"
          :class="[
            'appearance-none block w-full px-3 py-2 border rounded-md text-sm text-left',
            error && !selectedCityId ? 'border-destructive' : 'border-input',
            disabled ? 'bg-muted text-muted-foreground' : 'bg-white',
          ]"
          @click="openCitySheet"
        >
          <span v-if="citySearch" class="text-foreground">{{ citySearch }}</span>
          <span v-else class="text-muted-foreground">Начните вводить город...</span>
        </button>

        <BottomSheet v-model="citySheetOpen" label="Выберите город">
          <div class="px-4 pt-3 pb-2 border-b">
            <Input
              ref="cityInputRef"
              type="text"
              :value="citySearch"
              placeholder="Начните вводить город..."
              autocomplete="off"
              @input="handleCityInput"
              @keydown="handleCityKeydown"
            />
          </div>
          <div class="overflow-y-auto flex-1">
            <div v-if="isLoadingCities" class="flex justify-center py-6">
              <svg class="animate-spin h-6 w-6 text-muted-foreground" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </div>
            <template v-else>
              <button
                v-for="(city, index) in cities"
                :key="city.id"
                type="button"
                :class="[
                  'w-full px-4 py-3 text-left text-sm border-b border-border/30 active:bg-muted',
                  index === highlightedIndex ? 'bg-accent' : 'text-foreground',
                ]"
                @click="handleCitySheetSelect(city)"
              >
                <div class="font-medium">{{ city.name_ru || city.name }}</div>
                <div v-if="city.name_ru && city.name !== city.name_ru" class="text-muted-foreground text-xs">{{ city.name }}</div>
              </button>
              <div v-if="cities.length === 0 && citySearch.length > 0" class="px-4 py-6 text-sm text-muted-foreground text-center">
                Города не найдены
              </div>
              <div v-if="cities.length === 0 && citySearch.length === 0" class="px-4 py-6 text-sm text-muted-foreground text-center">
                Введите название города
              </div>
            </template>
          </div>
        </BottomSheet>
      </template>
    </div>

  </div>
  <!-- Error message -->
  <p v-if="error" class="mt-1 text-sm text-destructive">
    {{ error }}
  </p>
  </div>
</template>
