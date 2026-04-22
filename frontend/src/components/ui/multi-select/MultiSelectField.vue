<script setup lang="ts" generic="T extends string">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BottomSheet from '@/components/shared/BottomSheet.vue'
import HighlightText from '@/components/shared/HighlightText.vue'
import { X, ChevronDown, Check, Search } from 'lucide-vue-next'

interface Option {
  value: T
  label: string
}

interface Props {
  options: Option[]
  placeholder?: string
  sheetLabel?: string
  searchPlaceholder?: string
  emptyText?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: 'Выберите...',
  sheetLabel: 'Выберите',
  searchPlaceholder: 'Поиск...',
  emptyText: 'Не найдено',
})

const modelValue = defineModel<T[]>({ default: () => [] })

const { isMobile } = useBreakpoint()

// Desktop dropdown
const dropdownOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const searchQuery = ref('')

// Mobile sheet
const sheetOpen = ref(false)
const sheetSearch = ref('')

const filteredOptions = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  if (!q) return props.options
  return props.options.filter(o => o.label.toLowerCase().includes(q))
})

const filteredSheetOptions = computed(() => {
  const q = sheetSearch.value.toLowerCase().trim()
  if (!q) return props.options
  return props.options.filter(o => o.label.toLowerCase().includes(q))
})

const selectedLabels = computed(() =>
  modelValue.value.map(v => props.options.find(o => o.value === v)?.label ?? v)
)

function isSelected(value: T): boolean {
  return modelValue.value.includes(value)
}

function toggle(value: T) {
  const current = [...modelValue.value]
  const idx = current.indexOf(value)
  if (idx === -1) {
    current.push(value)
  } else {
    current.splice(idx, 1)
  }
  modelValue.value = current
}

function remove(value: T, event?: MouseEvent) {
  event?.stopPropagation()
  modelValue.value = modelValue.value.filter(v => v !== value)
}

const dropdownStyle = ref({ top: '0px', left: '0px', width: '0px' })
const dropdownRef = ref<HTMLElement | null>(null)

function openDesktop() {
  if (containerRef.value) {
    const rect = containerRef.value.getBoundingClientRect()
    dropdownStyle.value = {
      top: `${rect.bottom + 4}px`,
      left: `${rect.left}px`,
      width: `${rect.width}px`,
    }
  }
  dropdownOpen.value = true
  searchQuery.value = ''
}

function closeDesktop() {
  dropdownOpen.value = false
  searchQuery.value = ''
}

function handleClickOutside(e: MouseEvent) {
  const target = e.target as Node
  if (!containerRef.value?.contains(target) && !dropdownRef.value?.contains(target)) {
    closeDesktop()
  }
}

watch(dropdownOpen, (open) => {
  document.removeEventListener('click', handleClickOutside)
  if (open) document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

function openSheet() {
  sheetOpen.value = true
  sheetSearch.value = ''
}
</script>

<template>
  <!-- Desktop -->
  <template v-if="!isMobile">
    <div ref="containerRef" class="relative">
      <!-- Trigger -->
      <button
        type="button"
        class="w-full min-h-11 px-3 py-2 border border-input rounded-lg bg-white shadow-sm text-base text-left flex items-center gap-2 hover:border-primary/50 transition-colors focus:outline-none focus:border-primary"
        @click="dropdownOpen ? closeDesktop() : openDesktop()"
      >
        <span class="flex-1 flex flex-wrap gap-1 min-w-0">
          <span v-if="modelValue.length === 0" class="text-muted-foreground">{{ placeholder }}</span>
          <span
            v-for="(label, i) in selectedLabels"
            :key="modelValue[i]"
            class="inline-flex items-center gap-1 px-2 py-1 rounded bg-primary/10 text-primary text-sm font-medium"
          >
            {{ label }}
            <button
              type="button"
              tabindex="-1"
              class="hover:text-destructive transition-colors"
              @click.stop="remove(modelValue[i] as T)"
            >
              <X class="h-3 w-3" />
            </button>
          </span>
        </span>
        <ChevronDown
          class="h-4 w-4 text-muted-foreground shrink-0 transition-transform duration-200"
          :class="dropdownOpen ? 'rotate-180' : ''"
        />
      </button>

      <!-- Dropdown -->
      <Teleport to="body">
      <div
        v-if="dropdownOpen"
        ref="dropdownRef"
        class="fixed z-[200] bg-white border border-border rounded-lg shadow-lg flex flex-col"
        :style="{ ...dropdownStyle, maxHeight: '300px' }"
      >
        <!-- Search -->
        <div v-if="props.options.length > 5" class="p-2 border-b border-border flex-shrink-0">
          <div class="relative">
            <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            <input
              v-model="searchQuery"
              type="text"
              class="w-full pl-7 pr-2 py-2 text-base bg-white border border-input rounded-md outline-none transition-colors hover:border-primary/50 focus:border-primary"
              :placeholder="searchPlaceholder"
              @keydown.escape="closeDesktop"
            />
          </div>
        </div>

        <!-- Options -->
        <div class="overflow-y-auto flex-1">
          <button
            v-for="option in filteredOptions"
            :key="option.value"
            type="button"
            class="w-full px-3 py-2.5 text-left text-base flex items-center gap-2 hover:bg-accent transition-colors"
            @click="toggle(option.value)"
          >
            <span
              class="w-4 h-4 shrink-0 rounded border flex items-center justify-center transition-colors"
              :class="isSelected(option.value)
                ? 'bg-primary border-primary'
                : 'border-input bg-background'"
            >
              <Check v-if="isSelected(option.value)" class="h-3 w-3 text-primary-foreground" />
            </span>
            <HighlightText :text="option.label" :query="searchQuery" />
          </button>
          <div v-if="filteredOptions.length === 0" class="px-3 py-4 text-sm text-muted-foreground text-center">
            {{ emptyText }}
          </div>
        </div>
      </div>
      </Teleport>
    </div>
  </template>

  <!-- Mobile -->
  <template v-else>
    <div class="space-y-2">
      <button
        type="button"
        class="w-full min-h-11 px-3 py-2 border border-input rounded-lg bg-white shadow-sm text-base text-left flex items-center gap-2"
        @click="openSheet"
      >
        <span class="flex-1 text-muted-foreground">
          {{ modelValue.length === 0 ? placeholder : `Выбрано: ${modelValue.length}` }}
        </span>
        <ChevronDown class="h-4 w-4 text-muted-foreground shrink-0" />
      </button>

      <!-- Selected tags -->
      <div v-if="modelValue.length > 0" class="flex flex-wrap gap-2">
        <span
          v-for="(label, i) in selectedLabels"
          :key="modelValue[i]"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-primary/10 text-primary text-sm font-medium"
        >
          {{ label }}
          <button
            type="button"
            class="hover:text-destructive transition-colors"
            @click="remove(modelValue[i] as T)"
          >
            <X class="h-3.5 w-3.5" />
          </button>
        </span>
      </div>
    </div>

    <BottomSheet v-model="sheetOpen" :label="sheetLabel">
      <div v-if="props.options.length > 5" class="px-4 py-2 border-b flex-shrink-0">
        <input
          v-model="sheetSearch"
          type="text"
          class="w-full px-3 py-2.5 text-base border border-input rounded-lg bg-white transition-colors hover:border-primary/50 focus:outline-none focus:border-primary"
          :placeholder="searchPlaceholder"
        />
      </div>

      <div class="overflow-y-auto flex-1 divide-y divide-border/50">
        <button
          v-for="option in filteredSheetOptions"
          :key="option.value"
          type="button"
          class="w-full px-4 py-3.5 text-left text-base flex items-center gap-3 active:bg-accent"
          :class="isSelected(option.value) ? 'bg-accent' : ''"
          @click="toggle(option.value)"
        >
          <span
            class="w-4 h-4 shrink-0 rounded border flex items-center justify-center"
            :class="isSelected(option.value) ? 'bg-primary border-primary' : 'border-input bg-white'"
          >
            <Check v-if="isSelected(option.value)" class="h-3 w-3 text-primary-foreground" />
          </span>
          <span :class="isSelected(option.value) ? 'text-primary font-medium' : 'text-foreground'">
            <HighlightText :text="option.label" :query="sheetSearch" />
          </span>
        </button>
        <div v-if="filteredSheetOptions.length === 0" class="px-4 py-6 text-sm text-muted-foreground text-center">
          {{ emptyText }}
        </div>
      </div>
    </BottomSheet>
  </template>
</template>
