<script setup lang="ts">
import { ref } from 'vue'
import type { AcceptableValue } from 'reka-ui'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BottomSheet from '@/components/shared/BottomSheet.vue'

interface Option {
  value: string
  label: string
}

interface Props {
  options: Option[]
  placeholder?: string
  sheetLabel: string
  hasError?: boolean
  disabled?: boolean
  clearable?: boolean
  clearLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: 'Выберите...',
  hasError: false,
  disabled: false,
  clearable: false,
  clearLabel: 'Не выбран',
})

const modelValue = defineModel<string | undefined | null>()

const { isMobile } = useBreakpoint()
const sheetOpen = ref(false)

function handleDesktopChange(value: AcceptableValue) {
  if (!value || typeof value !== 'string') return
  modelValue.value = (props.clearable && value === '__none__') ? undefined : value
}

function handleSheetSelect(value: string | undefined) {
  modelValue.value = value
  sheetOpen.value = false
}
</script>

<template>
  <!-- Desktop -->
  <template v-if="!isMobile">
    <Select
      :key="String(modelValue)"
      :model-value="modelValue ?? undefined"
      :disabled="disabled"
      @update:model-value="handleDesktopChange"
    >
      <SelectTrigger :class="hasError ? 'w-full border-red-300' : 'w-full'">
        <SelectValue :placeholder="placeholder" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem v-if="clearable" value="__none__" class="text-gray-400">
          {{ clearLabel }}
        </SelectItem>
        <SelectItem v-for="option in options" :key="option.value" :value="option.value">
          {{ option.label }}
        </SelectItem>
      </SelectContent>
    </Select>
  </template>

  <!-- Mobile -->
  <template v-else>
    <button
      type="button"
      :disabled="disabled"
      :class="[
        'appearance-none block w-full px-3 py-2 border rounded-md text-sm text-left',
        hasError ? 'border-red-300' : 'border-gray-300',
        disabled ? 'bg-gray-100 text-gray-400' : 'bg-white',
      ]"
      @click="sheetOpen = true"
    >
      <span v-if="modelValue" class="text-gray-900">
        {{ options.find(o => o.value === modelValue)?.label }}
      </span>
      <span v-else class="text-gray-400">{{ placeholder }}</span>
    </button>

    <BottomSheet v-model="sheetOpen" :label="sheetLabel">
      <div class="overflow-y-auto flex-1">
        <button
          v-if="clearable"
          type="button"
          class="w-full px-4 py-3 text-left text-sm border-b border-gray-50 text-gray-400 active:bg-gray-100"
          @click="handleSheetSelect(undefined)"
        >
          {{ clearLabel }}
        </button>
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          :class="[
            'w-full px-4 py-3 text-left text-sm border-b border-gray-50 active:bg-gray-100',
            option.value === modelValue ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-900',
          ]"
          @click="handleSheetSelect(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
    </BottomSheet>
  </template>
</template>
