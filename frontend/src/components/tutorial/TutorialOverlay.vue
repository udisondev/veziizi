<script setup lang="ts">
/**
 * TutorialOverlay
 * Затемнение экрана с "дыркой" вокруг целевого элемента
 * Использует 4 div вокруг "дырки" чтобы клики внутри неё проходили к элементу
 *
 * Динамическое расширение области при открытии popup/dropdown:
 * - Автоматически определяет открытые popup рядом с target
 * - Расширяет "дырку" чтобы включить popup
 * - Сужает обратно когда popup закрывается
 */

import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useOnboardingStore } from '@/stores/onboarding'
import { storeToRefs } from 'pinia'
import { useTutorialPopupTracker, onPopupChange } from '@/composables/useTutorialPopupTracker'
import {
  TUTORIAL_OVERLAY_PADDING,
  TUTORIAL_OVERLAY_BORDER_RADIUS,
  TUTORIAL_RAF_THROTTLE_FRAMES,
} from '@/sandbox/constants'
import { throttle, SCROLL_THROTTLE_DELAY } from '@/utils/debounce'

const onboarding = useOnboardingStore()
const { isSandboxMode, currentStep, scrollPaused, validationErrorMode } = storeToRefs(onboarding)
const popupTracker = useTutorialPopupTracker()

// Позиция и размеры "дырки"
const holeRect = ref({ top: 0, left: 0, width: 0, height: 0 })
const isVisible = ref(false)

// Отступ вокруг целевого элемента
const padding = TUTORIAL_OVERLAY_PADDING
const borderRadius = TUTORIAL_OVERLAY_BORDER_RADIUS

// Размеры экрана
const screenWidth = ref(window.innerWidth)
const screenHeight = ref(window.innerHeight)

// Стили для 4 div вокруг "дырки"
const topStyle = computed(() => ({
  top: '0',
  left: '0',
  width: '100%',
  height: `${Math.max(0, holeRect.value.top)}px`,
}))

const bottomStyle = computed(() => ({
  top: `${holeRect.value.top + holeRect.value.height}px`,
  left: '0',
  width: '100%',
  height: `${Math.max(0, screenHeight.value - holeRect.value.top - holeRect.value.height)}px`,
}))

const leftStyle = computed(() => ({
  top: `${holeRect.value.top}px`,
  left: '0',
  width: `${Math.max(0, holeRect.value.left)}px`,
  height: `${holeRect.value.height}px`,
}))

const rightStyle = computed(() => ({
  top: `${holeRect.value.top}px`,
  left: `${holeRect.value.left + holeRect.value.width}px`,
  width: `${Math.max(0, screenWidth.value - holeRect.value.left - holeRect.value.width)}px`,
  height: `${holeRect.value.height}px`,
}))

// Сохраняем rect целевого элемента для popup tracker
const currentTargetRect = ref<DOMRect | null>(null)

function updateHolePosition() {
  // Обновляем размеры экрана
  screenWidth.value = window.innerWidth
  screenHeight.value = window.innerHeight

  // Поддерживаем target (data-tutorial) и highlightSelector (CSS селектор)
  const targetSelector = currentStep.value?.target
    ? `[data-tutorial="${currentStep.value.target}"]`
    : currentStep.value?.highlightSelector

  if (!targetSelector) {
    isVisible.value = false
    onboarding.setPopupDirection(null)
    return
  }

  const target = document.querySelector(targetSelector)
  if (!target) {
    console.warn(`[Tutorial] Target not found: ${targetSelector}`)
    isVisible.value = false
    onboarding.setPopupDirection(null)
    return
  }

  const rect = target.getBoundingClientRect()
  currentTargetRect.value = rect

  // Анализируем popup рядом с target
  const analysis = popupTracker.analyzeTarget(rect)

  if (analysis.hasPopups) {
    // Расширяем "дырку" чтобы включить popup
    const combined = analysis.combinedRect
    holeRect.value = {
      top: combined.top - padding,
      left: combined.left - padding,
      width: combined.width + padding * 2,
      height: combined.height + padding * 2,
    }
    // Сообщаем store о направлении popup для tooltip
    onboarding.setPopupDirection(analysis.primaryDirection)
  } else {
    // Стандартное поведение — только target
    holeRect.value = {
      top: rect.top - padding,
      left: rect.left - padding,
      width: rect.width + padding * 2,
      height: rect.height + padding * 2,
    }
    onboarding.setPopupDirection(null)
  }

  isVisible.value = true

  // Прокручиваем элемент в область видимости если нужно
  // Но только если элемент маленький (меньше половины viewport)
  // Для больших элементов (списки, формы) не скроллим — пользователь сам прокрутит
  // НЕ скроллим если scrollPaused (скроллим к ошибке валидации)
  if (!scrollPaused.value) {
    const elementHeight = rect.height
    const isSmallElement = elementHeight < window.innerHeight * 0.5

    if (isSmallElement && (rect.top < 0 || rect.bottom > window.innerHeight)) {
      target.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // Обновляем позицию после прокрутки
      setTimeout(updateHolePosition, 300)
    }
  }
}

// Отписка от shared observer
let popupUnsubscribe: (() => void) | null = null
let rafId: number | null = null
let rafActive = false

function startPopupTracking() {
  // Используем shared MutationObserver вместо собственного
  popupUnsubscribe = onPopupChange(() => {
    updateHolePosition()
    // Запускаем короткий RAF loop при появлении popup для отслеживания анимации
    startRAFLoop()
  })
}

function stopPopupTracking() {
  if (popupUnsubscribe) {
    popupUnsubscribe()
    popupUnsubscribe = null
  }
  stopRAFLoop()
  popupTracker.reset()
  onboarding.setPopupDirection(null)
}

// RAF loop — запускается только при появлении popup для отслеживания анимации
function startRAFLoop() {
  if (rafActive) return
  rafActive = true

  let frameCount = 0
  function rafLoop() {
    if (!rafActive || !isSandboxMode.value || !currentStep.value) {
      rafActive = false
      return
    }

    frameCount++
    // Обновляем каждые N кадров (~20fps вместо 60fps)
    if (frameCount % TUTORIAL_RAF_THROTTLE_FRAMES === 0 && !scrollPaused.value) {
      updateHolePosition()
    }

    // Останавливаем через ~500ms (30 кадров при ~60fps)
    if (frameCount > 30) {
      rafActive = false
      return
    }

    rafId = requestAnimationFrame(rafLoop)
  }
  rafId = requestAnimationFrame(rafLoop)
}

function stopRAFLoop() {
  rafActive = false
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
}

// Следим за изменениями шага
watch(
  [isSandboxMode, currentStep],
  async () => {
    if (isSandboxMode.value && currentStep.value) {
      await nextTick()
      setTimeout(() => {
        updateHolePosition()
        startPopupTracking()
      }, 100)
    } else {
      isVisible.value = false
      stopPopupTracking()
    }
  },
  { immediate: true }
)

// Throttled версия для scroll (много событий при прокрутке)
const throttledUpdateHolePosition = throttle(updateHolePosition, SCROLL_THROTTLE_DELAY)

// Обновляем при скролле и ресайзе
onMounted(() => {
  window.addEventListener('scroll', throttledUpdateHolePosition, true)
  window.addEventListener('resize', updateHolePosition)
})

onUnmounted(() => {
  window.removeEventListener('scroll', throttledUpdateHolePosition, true)
  window.removeEventListener('resize', updateHolePosition)
  stopPopupTracking()
})

// Блокируем клики на затемнённых областях
function blockClick(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-300"
      leave-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div v-if="isSandboxMode && isVisible && !validationErrorMode" class="fixed inset-0 z-[55] pointer-events-none">
        <!-- 4 div вокруг "дырки" - они блокируют клики, но разрешают скролл -->
        <!-- Верхняя полоса -->
        <div
          class="absolute bg-black/50 pointer-events-auto touch-pan-y"
          :style="topStyle"
          @click.capture="blockClick"
        />
        <!-- Нижняя полоса -->
        <div
          class="absolute bg-black/50 pointer-events-auto touch-pan-y"
          :style="bottomStyle"
          @click.capture="blockClick"
        />
        <!-- Левая полоса -->
        <div
          class="absolute bg-black/50 pointer-events-auto touch-pan-y"
          :style="leftStyle"
          @click.capture="blockClick"
        />
        <!-- Правая полоса -->
        <div
          class="absolute bg-black/50 pointer-events-auto touch-pan-y"
          :style="rightStyle"
          @click.capture="blockClick"
        />

        <!-- Подсветка границ целевого элемента (не блокирует клики) -->
        <div
          class="pointer-events-none absolute ring-2 ring-amber-400 ring-offset-2"
          :style="{
            top: `${holeRect.top}px`,
            left: `${holeRect.left}px`,
            width: `${holeRect.width}px`,
            height: `${holeRect.height}px`,
            borderRadius: `${borderRadius}px`,
          }"
        />
      </div>
    </Transition>
  </Teleport>
</template>
