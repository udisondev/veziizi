<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { Component } from 'vue'
import { ChevronDown, Check } from 'lucide-vue-next'

export interface TabSliderItem {
  value: string
  label: string
  icon?: Component
  badge?: number | string
  highlight?: boolean
  separator?: boolean
}

const props = defineProps<{
  items: TabSliderItem[]
  modelValue: string
  stretch?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const wrapperRef = ref<HTMLElement | null>(null)
const moreButtonRef = ref<HTMLElement | null>(null)
const tabRefs = ref<(HTMLElement | null)[]>([])

const visibleCount = ref(props.items.length)
const moreOpen = ref(false)

const hasOverflow = computed(() => visibleCount.value < props.items.length)
const overflowItems = computed(() => props.items.slice(visibleCount.value))
const activeInOverflow = computed(() => overflowItems.value.some(i => i.value === props.modelValue))

// Slider
const sliderStyle = ref({ left: '0px', width: '0px' })

function updateSlider() {
  if (activeInOverflow.value) {
    const el = moreButtonRef.value
    if (el) {
      sliderStyle.value = { left: `${el.offsetLeft}px`, width: `${el.offsetWidth}px` }
    }
    return
  }
  const idx = props.items.findIndex(i => i.value === props.modelValue)
  const el = tabRefs.value[idx]
  if (el) {
    sliderStyle.value = { left: `${el.offsetLeft}px`, width: `${el.offsetWidth}px` }
  }
}

watch(
  () => [props.modelValue, visibleCount.value] as const,
  async () => { await nextTick(); updateSlider() },
  { flush: 'post' },
)

function recalculate() {
  if (!wrapperRef.value) return

  // Временно показываем все табы для замера (через opacity, не display)
  // tabRefs всегда содержат все элементы — скрытые через opacity/pointer-events
  const available = wrapperRef.value.clientWidth - 8

  // Собираем ширины через scrollWidth каждого элемента
  // (работает даже если элемент opacity-0, но НЕ работает если display:none)
  const widths = tabRefs.value.map(el => el?.scrollWidth ?? 0)
  if (widths.length === 0 || available <= 0) return

  const moreW = moreButtonRef.value?.scrollWidth ?? 72

  const totalAll = widths.reduce((s, w) => s + w, 0)
  if (totalAll <= available) {
    visibleCount.value = props.items.length
    return
  }

  let acc = 0
  let count = 0
  for (let i = 0; i < widths.length; i++) {
    const w = widths[i] ?? 0
    const needMore = i < widths.length - 1
    if (acc + w + (needMore ? moreW : 0) <= available) {
      acc += w
      count++
    } else {
      break
    }
  }
  visibleCount.value = Math.max(1, count)
}

function rAF(): Promise<void> {
  return new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
}

let ro: ResizeObserver | null = null

onMounted(async () => {
  // Двойной rAF — гарантирует завершение layout и paint
  await rAF()
  recalculate()
  await nextTick()
  updateSlider()

  if (wrapperRef.value) {
    ro = new ResizeObserver(() => {
      recalculate()
      nextTick(() => updateSlider())
    })
    ro.observe(wrapperRef.value)
  }
})

onUnmounted(() => {
  ro?.disconnect()
  ro = null
  document.removeEventListener('click', handleClickOutside)
})

function select(value: string) {
  emit('update:modelValue', value)
  moreOpen.value = false
}

function handleClickOutside(e: MouseEvent) {
  const target = e.target as Node
  if (moreButtonRef.value && !moreButtonRef.value.contains(target)) {
    moreOpen.value = false
  }
}

watch(moreOpen, (open) => {
  document.removeEventListener('click', handleClickOutside)
  if (open) document.addEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="wrapperRef">
    <div class="relative flex bg-muted rounded-lg p-1" :class="(hasOverflow || stretch) ? 'w-full' : 'w-fit'">
      <!-- Sliding indicator -->
      <div
        class="absolute top-1 bottom-1 rounded-md bg-background shadow-sm transition-all duration-200 ease-in-out pointer-events-none"
        :style="sliderStyle"
      />

      <!--
        Все табы ВСЕГДА в DOM.
        Скрытые — opacity-0 + pointer-events-none + w-0 overflow-hidden px-0 gap-0
        Это позволяет scrollWidth работать для замера, не занимая места в layout.
      -->
      <button
        v-for="(item, index) in items"
        :key="item.value"
        :ref="(el) => { tabRefs[index] = el as HTMLElement | null }"
        role="tab"
        :aria-selected="item.value === modelValue"
        type="button"
        class="relative z-10 inline-flex items-center gap-1.5 text-sm font-medium rounded-md transition-all duration-200 whitespace-nowrap cursor-pointer"
        :class="index < visibleCount
          ? ['px-3 py-1.5', stretch && 'flex-1 justify-center', item.value === modelValue ? 'text-foreground' : 'text-muted-foreground hover:text-foreground']
          : 'opacity-0 pointer-events-none w-0 overflow-hidden px-0 py-1.5'"
        @click="select(item.value)"
      >
        <component v-if="item.icon" :is="item.icon" class="h-4 w-4 shrink-0" />
        {{ item.label }}
        <span
          v-if="item.badge"
          class="ml-0.5 rounded-full bg-primary/15 px-1.5 py-0.5 text-xs font-medium text-primary leading-none"
        >
          {{ item.badge }}
        </span>
        <span v-if="item.highlight" class="w-1.5 h-1.5 rounded-full bg-destructive shrink-0" />
      </button>

      <!-- Кнопка Ещё -->
      <div
        ref="moreButtonRef"
        :class="['flex', hasOverflow ? 'flex-1' : 'opacity-0 pointer-events-none w-0 overflow-hidden']"
      >
        <button
          type="button"
          class="relative z-10 flex-1 inline-flex items-center justify-center gap-1 px-3 py-1.5 text-sm font-medium rounded-md whitespace-nowrap cursor-pointer transition-colors duration-200"
          :class="activeInOverflow ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'"
          @click.stop="moreOpen = !moreOpen"
        >
          Ещё
          <span v-if="activeInOverflow" class="w-1.5 h-1.5 rounded-full bg-primary shrink-0" />
          <ChevronDown
            class="h-3.5 w-3.5 shrink-0 transition-transform duration-200"
            :class="moreOpen ? 'rotate-180' : ''"
          />
        </button>
      </div>
    </div>

    <!-- Dropdown -->
    <div v-if="moreOpen && hasOverflow" class="relative">
      <div class="absolute right-0 top-1 z-50 min-w-44 bg-white border border-border rounded-lg shadow-lg py-1">
        <button
          v-for="item in overflowItems"
          :key="item.value"
          type="button"
          class="w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted"
          :class="item.value === modelValue ? 'text-primary font-medium' : 'text-foreground'"
          @click="select(item.value)"
        >
          <component v-if="item.icon" :is="item.icon" class="h-4 w-4 shrink-0" />
          <span class="flex-1 text-left">{{ item.label }}</span>
          <span
            v-if="item.badge"
            class="rounded-full bg-primary/15 px-1.5 py-0.5 text-xs font-medium text-primary leading-none"
          >
            {{ item.badge }}
          </span>
          <span v-if="item.highlight" class="w-1.5 h-1.5 rounded-full bg-destructive shrink-0" />
          <Check v-if="item.value === modelValue" class="h-4 w-4 shrink-0 text-primary ml-auto" />
        </button>
      </div>
    </div>
  </div>
</template>
