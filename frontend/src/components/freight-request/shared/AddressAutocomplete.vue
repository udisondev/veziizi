<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { useAddressSearch, type AddressSearchResult } from '@/composables/useAddressSearch'
import type { Coordinates } from '@/types/freightRequest'

interface Props {
  modelValue: string
  coordinates?: Coordinates
  placeholder?: string
  error?: string | null
  disabled?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: string): void
  (e: 'update:coordinates', value: Coordinates | undefined): void
  (e: 'select', result: AddressSearchResult): void
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: 'Начните вводить адрес...',
  disabled: false,
})

const emit = defineEmits<Emits>()

const { results, isLoading, isOpen, setQuery, close } = useAddressSearch()

const inputRef = ref<HTMLInputElement | null>(null)
const dropdownRef = ref<HTMLDivElement | null>(null)
const highlightedIndex = ref(-1)

function handleInput(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', value)
  setQuery(value)
  highlightedIndex.value = -1
}

function handleSelect(result: AddressSearchResult) {
  emit('update:modelValue', result.displayName)
  emit('update:coordinates', result.coordinates)
  emit('select', result)
  close()
}

function handleKeydown(event: KeyboardEvent) {
  if (!isOpen.value || results.value.length === 0) return

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      highlightedIndex.value = Math.min(
        highlightedIndex.value + 1,
        results.value.length - 1
      )
      break
    case 'ArrowUp':
      event.preventDefault()
      highlightedIndex.value = Math.max(highlightedIndex.value - 1, 0)
      break
    case 'Enter':
      event.preventDefault()
      const selected = results.value[highlightedIndex.value]
      if (highlightedIndex.value >= 0 && selected) {
        handleSelect(selected)
      }
      break
    case 'Escape':
      close()
      break
  }
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Node
  if (
    inputRef.value &&
    !inputRef.value.contains(target) &&
    dropdownRef.value &&
    !dropdownRef.value.contains(target)
  ) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

watch(
  () => props.modelValue,
  () => {
    // Если значение изменилось извне, можно очистить координаты
    // (пользователь вручную редактирует адрес)
  }
)
</script>

<template>
  <div class="relative">
    <input
      ref="inputRef"
      type="text"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :class="[
        'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500',
        error ? 'border-red-300' : 'border-gray-300',
        disabled ? 'bg-gray-100 cursor-not-allowed' : '',
      ]"
      autocomplete="off"
      @input="handleInput"
      @keydown="handleKeydown"
      @focus="modelValue.length >= 3 && setQuery(modelValue)"
    />

    <!-- Loading indicator -->
    <div
      v-if="isLoading"
      class="absolute right-3 top-1/2 -translate-y-1/2"
    >
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

    <!-- Coordinates indicator -->
    <div
      v-else-if="coordinates"
      class="absolute right-3 top-1/2 -translate-y-1/2 text-green-500"
      title="Координаты определены"
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

    <!-- Dropdown -->
    <div
      v-if="isOpen && results.length > 0"
      ref="dropdownRef"
      class="absolute z-50 w-full mt-1 bg-white border border-gray-200 rounded-md shadow-lg max-h-60 overflow-auto"
    >
      <button
        v-for="(result, index) in results"
        :key="result.placeId"
        type="button"
        :class="[
          'w-full px-3 py-2 text-left text-sm hover:bg-gray-100',
          index === highlightedIndex ? 'bg-blue-50' : '',
        ]"
        @click="handleSelect(result)"
        @mouseenter="highlightedIndex = index"
      >
        <div class="font-medium text-gray-900 truncate">
          {{ result.shortName }}
        </div>
        <div class="text-gray-500 text-xs truncate">
          {{ result.displayName }}
        </div>
      </button>
    </div>

    <!-- Error message -->
    <p v-if="error" class="mt-1 text-sm text-red-600">
      {{ error }}
    </p>
  </div>
</template>
