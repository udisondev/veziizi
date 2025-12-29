/**
 * Composable for city dropdown UI behavior
 * Handles keyboard navigation and click-outside detection
 */

import { ref, onMounted, onUnmounted, type Ref } from 'vue'
import type { City } from '@/composables/useGeo'

export interface UseCityDropdownOptions {
  /** Cities list ref */
  cities: Ref<City[]>
  /** Dropdown open state ref */
  isOpen: Ref<boolean>
  /** Callback when city is selected via keyboard */
  onSelect: (city: City) => void
  /** Callback to close dropdown */
  onClose: () => void
}

export interface UseCityDropdownReturn {
  /** Currently highlighted index */
  highlightedIndex: Ref<number>
  /** Ref to attach to city input element */
  cityInputRef: Ref<HTMLInputElement | null>
  /** Ref to attach to dropdown element */
  cityDropdownRef: Ref<HTMLDivElement | null>
  /** Handle keyboard events on input */
  handleKeydown: (event: KeyboardEvent) => void
  /** Reset highlighted index */
  resetHighlight: () => void
}

/**
 * Provides keyboard navigation and click-outside handling for city dropdown
 *
 * @example
 * ```ts
 * const { cities, isCityDropdownOpen } = useGeo()
 *
 * const {
 *   highlightedIndex,
 *   cityInputRef,
 *   cityDropdownRef,
 *   handleKeydown,
 *   resetHighlight,
 * } = useCityDropdown({
 *   cities,
 *   isOpen: isCityDropdownOpen,
 *   onSelect: (city) => selectCity(city),
 *   onClose: () => closeCityDropdown(),
 * })
 * ```
 */
export function useCityDropdown(options: UseCityDropdownOptions): UseCityDropdownReturn {
  const { cities, isOpen, onSelect, onClose } = options

  const highlightedIndex = ref(-1)
  const cityInputRef = ref<HTMLInputElement | null>(null)
  const cityDropdownRef = ref<HTMLDivElement | null>(null)

  function handleKeydown(event: KeyboardEvent): void {
    if (!isOpen.value || cities.value.length === 0) return

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
        if (highlightedIndex.value >= 0) {
          const selected = cities.value[highlightedIndex.value]
          if (selected) {
            onSelect(selected)
          }
        }
        break
      case 'Escape':
        onClose()
        break
    }
  }

  function handleClickOutside(event: MouseEvent): void {
    const target = event.target as Node
    if (
      cityInputRef.value &&
      !cityInputRef.value.contains(target) &&
      cityDropdownRef.value &&
      !cityDropdownRef.value.contains(target)
    ) {
      onClose()
    }
  }

  function resetHighlight(): void {
    highlightedIndex.value = -1
  }

  onMounted(() => {
    document.addEventListener('click', handleClickOutside)
  })

  onUnmounted(() => {
    document.removeEventListener('click', handleClickOutside)
  })

  return {
    highlightedIndex,
    cityInputRef,
    cityDropdownRef,
    handleKeydown,
    resetHighlight,
  }
}
