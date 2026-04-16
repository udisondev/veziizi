<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue'
import type { Component } from 'vue'

export interface TabSliderItem {
  value: string
  label: string
  icon?: Component
  badge?: number | string
}

const props = defineProps<{
  items: TabSliderItem[]
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const tabRefs = ref<HTMLElement[]>([])
const sliderStyle = ref({ left: '0px', width: '0px' })

function updateSlider() {
  const activeIndex = props.items.findIndex((item) => item.value === props.modelValue)
  const el = tabRefs.value[activeIndex]
  if (el) {
    sliderStyle.value = {
      left: `${el.offsetLeft}px`,
      width: `${el.offsetWidth}px`,
    }
  }
}

watch(
  () => props.modelValue,
  async () => {
    await nextTick()
    updateSlider()
  },
)

onMounted(async () => {
  await nextTick()
  updateSlider()
})
</script>

<template>
  <div class="overflow-x-auto">
    <div class="relative inline-flex bg-muted rounded-lg p-1 min-w-max" role="tablist">
      <!-- Sliding indicator -->
      <div
        class="absolute top-1 bottom-1 rounded-md bg-background shadow-sm transition-all duration-200 ease-in-out pointer-events-none"
        :style="sliderStyle"
      />

      <!-- Tab buttons -->
      <button
        v-for="(item, index) in items"
        :key="item.value"
        :ref="(el) => { if (el) tabRefs[index] = el as HTMLElement }"
        role="tab"
        :aria-selected="item.value === modelValue"
        class="relative z-10 inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors duration-200 whitespace-nowrap cursor-pointer"
        :class="
          item.value === modelValue
            ? 'text-foreground'
            : 'text-muted-foreground hover:text-foreground'
        "
        type="button"
        @click="emit('update:modelValue', item.value)"
      >
        <component v-if="item.icon" :is="item.icon" class="h-4 w-4 shrink-0" />
        {{ item.label }}
        <span
          v-if="item.badge"
          class="ml-0.5 rounded-full bg-primary/15 px-1.5 py-0.5 text-xs font-medium text-primary leading-none"
        >
          {{ item.badge }}
        </span>
      </button>
    </div>
  </div>
</template>
