import { ref, computed } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import type { Coordinates } from '@/types/freightRequest'

// Types matching backend projections.Country and projections.City
export interface Country {
  id: number
  name: string
  name_ru?: string
  iso2: string
  iso3?: string
  phone_code?: string
  native_name?: string
  latitude?: number
  longitude?: number
}

export interface City {
  id: number
  name: string
  name_ru?: string
  country_id: number
  state_name?: string
  state_code?: string
  latitude: number
  longitude: number
  country_name?: string
  country_name_ru?: string
  country_iso2?: string
}

// Helper to get display name (Russian preferred, fallback to English)
export function getDisplayName(item: { name: string; name_ru?: string }): string {
  return item.name_ru || item.name
}

// Helper to get city display name (without region code)
export function getCityDisplayName(city: City): string {
  return city.name_ru || city.name
}

// Cached countries (loaded once)
let countriesCache: Country[] | null = null
let countriesPromise: Promise<Country[]> | null = null

/**
 * Load all countries (cached)
 */
export async function loadCountries(): Promise<Country[]> {
  if (countriesCache) {
    return countriesCache
  }

  if (countriesPromise) {
    return countriesPromise
  }

  countriesPromise = fetch('/api/v1/geo/countries')
    .then(async (res) => {
      if (!res.ok) {
        throw new Error('Failed to load countries')
      }
      const data: Country[] = await res.json()
      countriesCache = data
      return data
    })
    .finally(() => {
      countriesPromise = null
    })

  return countriesPromise
}

/**
 * Search cities in a country
 */
export async function searchCities(
  countryId: number,
  search: string,
  limit: number = 20
): Promise<City[]> {
  const params = new URLSearchParams({
    search,
    limit: limit.toString(),
  })

  const res = await fetch(`/api/v1/geo/countries/${countryId}/cities?${params}`)
  if (!res.ok) {
    throw new Error('Failed to search cities')
  }

  const data = await res.json()
  return data ?? []
}

/**
 * Get city by ID
 */
export async function getCity(cityId: number): Promise<City> {
  const res = await fetch(`/api/v1/geo/cities/${cityId}`)
  if (!res.ok) {
    throw new Error('City not found')
  }

  return res.json()
}

/**
 * Composable for country/city selection
 */
export function useGeo() {
  const countries = ref<Country[]>([])
  const cities = ref<City[]>([])
  const isLoadingCountries = ref(false)
  const isLoadingCities = ref(false)
  const error = ref<string | null>(null)

  // Selected values
  const selectedCountryId = ref<number | undefined>(undefined)
  const selectedCityId = ref<number | undefined>(undefined)
  const selectedCity = ref<City | undefined>(undefined)

  // City search
  const citySearch = ref('')
  const isCityDropdownOpen = ref(false)

  // Computed
  const selectedCountry = computed(() =>
    countries.value.find((c) => c.id === selectedCountryId.value)
  )

  const coordinates = computed<Coordinates | undefined>(() => {
    if (selectedCity.value) {
      return {
        latitude: selectedCity.value.latitude,
        longitude: selectedCity.value.longitude,
      }
    }
    return undefined
  })

  // Load countries on first use
  async function fetchCountries() {
    if (countries.value.length > 0) return

    isLoadingCountries.value = true
    error.value = null

    try {
      countries.value = await loadCountries()
    } catch (e) {
      error.value = 'Ошибка загрузки стран'
    } finally {
      isLoadingCountries.value = false
    }
  }

  // Search cities with debounce
  const debouncedSearchCities = useDebounceFn(async (query: string) => {
    if (!selectedCountryId.value) {
      cities.value = []
      return
    }

    if (query.length < 1) {
      // Load initial cities without filter
      isLoadingCities.value = true
      try {
        cities.value = await searchCities(selectedCountryId.value, '', 20)
        isCityDropdownOpen.value = cities.value.length > 0
      } catch (e) {
        error.value = 'Ошибка загрузки городов'
      } finally {
        isLoadingCities.value = false
      }
      return
    }

    isLoadingCities.value = true
    error.value = null

    try {
      cities.value = await searchCities(selectedCountryId.value, query, 20)
      isCityDropdownOpen.value = cities.value.length > 0
    } catch (e) {
      error.value = 'Ошибка поиска городов'
    } finally {
      isLoadingCities.value = false
    }
  }, 300)

  function setCitySearch(value: string) {
    citySearch.value = value
    debouncedSearchCities(value)
  }

  // Select country
  function selectCountry(countryId: number | undefined) {
    selectedCountryId.value = countryId
    // Reset city when country changes
    selectedCityId.value = undefined
    selectedCity.value = undefined
    citySearch.value = ''
    cities.value = []

    // Load initial cities for new country
    if (countryId) {
      debouncedSearchCities('')
    }
  }

  // Select city
  function selectCity(city: City) {
    selectedCityId.value = city.id
    selectedCity.value = city
    citySearch.value = getCityDisplayName(city)
    isCityDropdownOpen.value = false
  }

  // Load city by ID (for editing existing data)
  async function loadCityById(cityId: number) {
    try {
      const city = await getCity(cityId)
      selectedCityId.value = city.id
      selectedCity.value = city
      selectedCountryId.value = city.country_id
      citySearch.value = getCityDisplayName(city)

      // Make sure country is in the list
      await fetchCountries()
    } catch (e) {
      error.value = 'Город не найден'
    }
  }

  // Reset
  function reset() {
    selectedCountryId.value = undefined
    selectedCityId.value = undefined
    selectedCity.value = undefined
    citySearch.value = ''
    cities.value = []
    isCityDropdownOpen.value = false
    error.value = null
  }

  function closeCityDropdown() {
    isCityDropdownOpen.value = false
  }

  return {
    // State
    countries,
    cities,
    isLoadingCountries,
    isLoadingCities,
    error,

    // Selected values
    selectedCountryId,
    selectedCityId,
    selectedCountry,
    selectedCity,
    coordinates,

    // City search
    citySearch,
    isCityDropdownOpen,

    // Actions
    fetchCountries,
    setCitySearch,
    selectCountry,
    selectCity,
    loadCityById,
    reset,
    closeCityDropdown,
  }
}
