import { ref } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import type { Coordinates } from '@/types/freightRequest'

export interface NominatimResult {
  place_id: number
  display_name: string
  lat: string
  lon: string
  type: string
  address?: {
    city?: string
    town?: string
    village?: string
    state?: string
    country?: string
  }
}

export interface AddressSearchResult {
  displayName: string
  shortName: string
  coordinates: Coordinates
  placeId: number
}

function formatShortAddress(result: NominatimResult): string {
  const parts = result.display_name.split(', ')
  // Берём первые 2-3 части адреса для краткого отображения
  return parts.slice(0, 3).join(', ')
}

function toSearchResult(result: NominatimResult): AddressSearchResult {
  return {
    displayName: result.display_name,
    shortName: formatShortAddress(result),
    coordinates: {
      latitude: parseFloat(result.lat),
      longitude: parseFloat(result.lon),
    },
    placeId: result.place_id,
  }
}

export function useAddressSearch() {
  const query = ref('')
  const results = ref<AddressSearchResult[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const isOpen = ref(false)

  const search = useDebounceFn(async (q: string) => {
    if (q.length < 3) {
      results.value = []
      isOpen.value = false
      return
    }

    isLoading.value = true
    error.value = null

    try {
      const params = new URLSearchParams({
        format: 'json',
        q: q,
        limit: '7',
        countrycodes: 'ru,kz,by',
        addressdetails: '1',
      })

      const response = await fetch(
        `https://nominatim.openstreetmap.org/search?${params}`,
        {
          headers: {
            'Accept-Language': 'ru',
          },
        }
      )

      if (!response.ok) {
        throw new Error('Ошибка поиска')
      }

      const data: NominatimResult[] = await response.json()
      results.value = data.map(toSearchResult)
      isOpen.value = results.value.length > 0
    } catch (e) {
      error.value = 'Ошибка поиска адреса'
      results.value = []
    } finally {
      isLoading.value = false
    }
  }, 300)

  function setQuery(value: string) {
    query.value = value
    search(value)
  }

  function clear() {
    query.value = ''
    results.value = []
    isOpen.value = false
    error.value = null
  }

  function close() {
    isOpen.value = false
  }

  return {
    query,
    results,
    isLoading,
    error,
    isOpen,
    setQuery,
    clear,
    close,
  }
}

// Reverse geocoding - получение адреса по координатам
export async function reverseGeocode(coordinates: Coordinates): Promise<string | null> {
  try {
    const params = new URLSearchParams({
      format: 'json',
      lat: coordinates.latitude.toString(),
      lon: coordinates.longitude.toString(),
    })

    const response = await fetch(
      `https://nominatim.openstreetmap.org/reverse?${params}`,
      {
        headers: {
          'Accept-Language': 'ru',
        },
      }
    )

    if (!response.ok) {
      return null
    }

    const data: NominatimResult = await response.json()
    return data.display_name || null
  } catch {
    return null
  }
}
