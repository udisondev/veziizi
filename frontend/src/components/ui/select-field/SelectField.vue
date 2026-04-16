<script setup lang="ts" generic="T extends string">
import { ref, computed } from 'vue'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BottomSheet from '@/components/shared/BottomSheet.vue'
import { ChevronDown, Check } from 'lucide-vue-next'

interface Option {
  value: T
  label: string
}

interface Props {
  options: Option[]
  placeholder?: string
  sheetLabel?: string
  hasError?: boolean
  disabled?: boolean
  clearable?: boolean
  clearLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: 'Выберите...',
  sheetLabel: 'Выберите',
  hasError: false,
  disabled: false,
  clearable: false,
  clearLabel: 'Не выбрано',
})

const modelValue = defineModel<T | undefined | null>()

const { isMobile } = useBreakpoint()

const dropdownOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const sheetOpen = ref(false)

const selectedLabel = computed(
  () => props.options.find(o => o.value === modelValue.value)?.label ?? props.placeholder
)

const isSelected = (value: T) => modelValue.value === value

function select(value: T | undefined) {
  modelValue.value = value
  dropdownOpen.value = false
  sheetOpen.value = false
}

function onFocusOut(e: FocusEvent) {
  const related = e.relatedTarget as Node | null
  if (!containerRef.value?.contains(related)) {
    dropdownOpen.value = false
  }
}
</script>

<template>
  <!-- Desktop -->
  <template v-if="!isMobile">
    <div ref="containerRef" class="relative" @focusout="onFocusOut">
      <button
        type="button"
        :disabled="disabled"
        class="w-full h-11 px-3 py-2 border rounded-lg bg-white shadow-sm text-base text-left flex items-center justify-between gap-2 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 disabled:cursor-not-allowed"
        :class="[
          hasError ? 'border-destructive' : 'border-input',
          !disabled ? 'hover:bg-accent/30' : '',
        ]"
        @click="dropdownOpen = !dropdownOpen"
      >
        <span :class="modelValue ? 'text-foreground' : 'text-muted-foreground'">
          {{ selectedLabel }}
        </span>
        <ChevronDown
          class="h-4 w-4 text-muted-foreground shrink-0 transition-transform duration-200"
          :class="dropdownOpen ? 'rotate-180' : ''"
        />
      </button>

      <div
        v-if="dropdownOpen"
        class="absolute z-50 top-full mt-1 w-full bg-white border border-border rounded-lg shadow-lg overflow-hidden"
        @mousedown.prevent
      >
        <button
          v-if="clearable"
          type="button"
          class="w-full px-3 py-2.5 text-left text-base flex items-center gap-2 hover:bg-accent transition-colors text-muted-foreground"
          @click="select(undefined)"
        >
          <span class="w-4 h-4 shrink-0" />
          {{ clearLabel }}
        </button>
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          class="w-full px-3 py-2.5 text-left text-base flex items-center gap-2 hover:bg-accent transition-colors"
          :class="isSelected(option.value) ? 'bg-accent' : ''"
          @click="select(option.value)"
        >
          <Check
            class="h-4 w-4 shrink-0 text-primary"
            :class="isSelected(option.value) ? 'opacity-100' : 'opacity-0'"
          />
          <span :class="isSelected(option.value) ? 'font-medium text-primary' : 'text-foreground'">
            {{ option.label }}
          </span>
        </button>
      </div>
    </div>
  </template>

  <!-- Mobile -->
  <template v-else>
    <button
      type="button"
      :disabled="disabled"
      class="w-full h-11 px-3 py-2 border rounded-lg bg-white shadow-sm text-base text-left flex items-center justify-between gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
      :class="hasError ? 'border-destructive' : 'border-input'"
      @click="sheetOpen = true"
    >
      <span :class="modelValue ? 'text-foreground' : 'text-muted-foreground'">
        {{ selectedLabel }}
      </span>
      <ChevronDown class="h-4 w-4 text-muted-foreground shrink-0" />
    </button>

    <BottomSheet v-model="sheetOpen" :label="sheetLabel">
      <div class="overflow-y-auto flex-1">
        <button
          v-if="clearable"
          type="button"
          class="w-full px-4 py-3.5 text-left text-base border-b border-border text-muted-foreground active:bg-accent"
          @click="select(undefined)"
        >
          {{ clearLabel }}
        </button>
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          class="w-full px-4 py-3.5 text-left text-base border-b border-border flex items-center gap-3 active:bg-accent"
          :class="isSelected(option.value) ? 'bg-accent' : ''"
          @click="select(option.value)"
        >
          <Check
            class="h-4 w-4 shrink-0 text-primary"
            :class="isSelected(option.value) ? 'opacity-100' : 'opacity-0'"
          />
          <span :class="isSelected(option.value) ? 'font-medium text-primary' : 'text-foreground'">
            {{ option.label }}
          </span>
        </button>
      </div>
    </BottomSheet>
  </template>
</template>
