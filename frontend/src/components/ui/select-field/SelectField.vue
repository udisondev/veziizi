<script setup lang="ts" generic="T extends string">
import { ref, computed, watch, onUnmounted } from 'vue'
import type { Component } from 'vue'
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
  icon?: Component
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: 'Выберите...',
  sheetLabel: 'Выберите',
  hasError: false,
  disabled: false,
  clearable: false,
  clearLabel: 'Не выбрано',
  icon: undefined,
})

const modelValue = defineModel<T | undefined | null>()

const { isMobile } = useBreakpoint()

const dropdownOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const sheetOpen = ref(false)

const dropdownStyle = ref({ top: '0px', left: '0px', width: '0px', maxHeight: '280px' })
const dropdownRef = ref<HTMLElement | null>(null)

function openDropdown() {
  if (!containerRef.value) return
  const rect = containerRef.value.getBoundingClientRect()
  dropdownStyle.value = {
    top: `${rect.bottom + 4}px`,
    left: `${rect.left}px`,
    width: `${rect.width}px`,
    maxHeight: `${Math.min(window.innerHeight - rect.bottom - 8, 280)}px`,
  }
  dropdownOpen.value = true
}

const selectedLabel = computed(
  () => props.options.find(o => o.value === modelValue.value)?.label ?? props.placeholder
)

const isSelected = (value: T) => modelValue.value === value

function select(value: T | undefined) {
  modelValue.value = value
  dropdownOpen.value = false
  sheetOpen.value = false
}

function handleClickOutside(e: MouseEvent) {
  const target = e.target as Node
  if (!containerRef.value?.contains(target) && !dropdownRef.value?.contains(target)) {
    dropdownOpen.value = false
  }
}

watch(dropdownOpen, (open) => {
  document.removeEventListener('click', handleClickOutside)
  if (open) document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <!-- Desktop -->
  <template v-if="!isMobile">
    <div ref="containerRef" class="relative">
      <button
        type="button"
        :disabled="disabled"
        class="w-full h-11 px-3 py-2 border rounded-lg bg-white shadow-sm text-base text-left flex items-center justify-between gap-2 transition-colors hover:border-primary/50 focus:outline-none focus:border-primary disabled:opacity-50 disabled:cursor-not-allowed"
        :class="[
          hasError ? 'border-destructive' : 'border-input',
        ]"
        @click="dropdownOpen ? dropdownOpen = false : openDropdown()"
      >
        <span class="flex items-center gap-1.5 min-w-0">
          <component :is="props.icon" v-if="props.icon" class="h-4 w-4 text-muted-foreground shrink-0" />
          <span class="truncate" :class="modelValue ? 'text-foreground' : 'text-muted-foreground'">
            {{ selectedLabel }}
          </span>
        </span>
        <ChevronDown
          class="h-4 w-4 text-muted-foreground shrink-0 transition-transform duration-200"
          :class="dropdownOpen ? 'rotate-180' : ''"
        />
      </button>

      <Teleport to="body">
        <div
          v-if="dropdownOpen"
          ref="dropdownRef"
          class="fixed z-[200] bg-white border border-border rounded-lg shadow-lg overflow-y-auto"
          :style="dropdownStyle"
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
      </Teleport>
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
      <span class="flex items-center gap-1.5 min-w-0">
        <component :is="props.icon" v-if="props.icon" class="h-4 w-4 text-muted-foreground shrink-0" />
        <span class="truncate" :class="modelValue ? 'text-foreground' : 'text-muted-foreground'">
          {{ selectedLabel }}
        </span>
      </span>
      <ChevronDown class="h-4 w-4 text-muted-foreground shrink-0" />
    </button>

    <BottomSheet v-model="sheetOpen" :label="sheetLabel">
      <div class="overflow-y-auto flex-1 divide-y divide-border/50">
        <button
          v-if="clearable"
          type="button"
          class="w-full px-4 py-3.5 text-left text-base text-muted-foreground active:bg-accent"
          @click="select(undefined)"
        >
          {{ clearLabel }}
        </button>
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          class="w-full px-4 py-3.5 text-left text-base flex items-center gap-3 active:bg-accent"
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
