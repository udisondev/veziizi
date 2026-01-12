<script setup lang="ts">
/**
 * TutorialTooltip
 * Подсказка с инструкцией для текущего шага
 */

import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useOnboardingStore } from '@/stores/onboarding'
import { storeToRefs } from 'pinia'
import { Button } from '@/components/ui/button'
import { ChevronRight, ChevronLeft, SkipForward } from 'lucide-vue-next'
import { useTutorialPopupTracker, onPopupChange } from '@/composables/useTutorialPopupTracker'
import { tutorialBus } from '@/sandbox/events'
import { throttle, SCROLL_THROTTLE_DELAY } from '@/utils/debounce'

const router = useRouter()
const onboarding = useOnboardingStore()
const { currentStep, currentStepIndex, scenarioSteps, popupDirection, scrollPaused, validationErrorMode } = storeToRefs(onboarding)
const popupTracker = useTutorialPopupTracker()

const tooltipRef = ref<HTMLDivElement | null>(null)
const position = ref({ top: 0, left: 0 })
const placement = ref<'top' | 'bottom' | 'left' | 'right'>('bottom')

// Прокручиваем к элементу если он не виден
// НЕ прокручиваем если scrollPaused (идёт scroll к ошибке валидации)
async function scrollToTargetIfNeeded(target: Element): Promise<void> {
  // Не прокручиваем во время паузы (scroll к ошибке валидации)
  if (scrollPaused.value) return

  // Не скроллим для элементов внутри fixed/modal контейнеров (Sheet, Dialog)
  // scrollIntoView на таких элементах скроллит основной документ неправильно
  const isInsideFixedContainer = target.closest('[role="dialog"], [data-state="open"], .fixed')
  if (isInsideFixedContainer) return

  const rect = target.getBoundingClientRect()
  const viewportHeight = window.innerHeight

  // Для больших элементов (больше половины viewport) не скроллим
  // Пользователь сам прокрутит к нужной части
  const isSmallElement = rect.height < viewportHeight * 0.5
  if (!isSmallElement) return

  // Если элемент не виден или слишком близко к краям — прокручиваем
  if (rect.top < 100 || rect.bottom > viewportHeight - 200) {
    target.scrollIntoView({ behavior: 'smooth', block: 'center' })
    // Ждём завершения прокрутки
    await new Promise(resolve => setTimeout(resolve, 350))
  }
}

// Получаем позицию целевого элемента
async function updatePosition() {
  // Поддерживаем target (data-tutorial) и highlightSelector (CSS селектор)
  const targetSelector = currentStep.value?.target
    ? `[data-tutorial="${currentStep.value.target}"]`
    : currentStep.value?.highlightSelector

  const tooltip = tooltipRef.value
  if (!tooltip) return

  // Если нет target — центрируем tooltip на экране
  if (!targetSelector) {
    const tooltipRect = tooltip.getBoundingClientRect()
    position.value = {
      top: Math.max(100, (window.innerHeight - tooltipRect.height) / 2),
      left: Math.max(20, (window.innerWidth - tooltipRect.width) / 2),
    }
    placement.value = 'bottom' // скрываем стрелку через стили
    return
  }

  const target = document.querySelector(targetSelector)
  if (!target) {
    // Target ещё не в DOM (например, внутри модалки с анимацией)
    // Центрируем tooltip временно, retry вызовется автоматически через 600ms
    const tooltipRect = tooltip.getBoundingClientRect()
    position.value = {
      top: Math.max(100, (window.innerHeight - tooltipRect.height) / 2),
      left: Math.max(20, (window.innerWidth - tooltipRect.width) / 2),
    }
    return
  }

  // Прокручиваем к элементу перед позиционированием tooltip
  await scrollToTargetIfNeeded(target)

  const targetRect = target.getBoundingClientRect()

  // Анализируем popup рядом с target
  const analysis = popupTracker.analyzeTarget(targetRect)

  const tooltipRect = tooltip.getBoundingClientRect()
  const padding = 12
  const arrowSize = 8
  const bottomPadding = 100 // Отступ снизу для кнопок tooltip

  // Максимальная ширина tooltip (sm = 384px или 100vw - 24px на мобильных)
  // Используем maxTooltipWidth для расчёта позиции, т.к. CSS maxWidth гарантирует эту ширину
  const maxTooltipWidth = Math.min(384, window.innerWidth - padding * 2)
  const effectiveTooltipWidth = maxTooltipWidth

  // Выбираем rect для позиционирования:
  // - Если popup открыт вниз, используем combined rect чтобы tooltip был под popup
  // - Иначе используем target rect
  const rect = (analysis.hasPopups && analysis.primaryDirection === 'down')
    ? analysis.combinedRect as DOMRect
    : targetRect

  // Определяем лучшую позицию
  let spaceTop = rect.top
  let spaceBottom = window.innerHeight - rect.bottom
  const spaceLeft = rect.left
  const spaceRight = window.innerWidth - rect.right

  let preferredPlacement = currentStep.value.tooltipPosition || 'bottom'

  // Если popup открыт вниз — tooltip тоже идёт вниз (под popup)
  // Мы уже используем combined rect, так что tooltip будет под выпадающим списком
  if (analysis.hasPopups && analysis.primaryDirection === 'down') {
    preferredPlacement = 'bottom'
  } else if (analysis.hasPopups && analysis.primaryDirection) {
    // Для других направлений используем стандартную логику
    preferredPlacement = popupTracker.getOptimalTooltipPosition(
      analysis.primaryDirection,
      preferredPlacement as 'top' | 'bottom' | 'left' | 'right'
    )

    // Корректируем доступное пространство с учётом popup
    // Если popup внизу — уменьшаем spaceBottom
    // Если popup вверху — уменьшаем spaceTop
    if (analysis.primaryDirection === 'down') {
      // Popup занимает место снизу от target
      spaceBottom = Math.max(0, window.innerHeight - analysis.combinedRect.bottom)
    } else if (analysis.primaryDirection === 'up') {
      // Popup занимает место сверху от target
      spaceTop = Math.max(0, analysis.combinedRect.top)
    }
  }

  // Вычисляем позицию в зависимости от размещения
  let newPlacement = preferredPlacement
  let top = 0
  let left = 0

  // Определяем fallback позицию если недостаточно места
  const needsVerticalSpace = tooltipRect.height + padding
  const needsHorizontalSpace = effectiveTooltipWidth + padding
  const hasSpaceTop = spaceTop >= needsVerticalSpace
  const hasSpaceBottom = spaceBottom >= needsVerticalSpace + bottomPadding
  const hasSpaceLeft = spaceLeft >= needsHorizontalSpace
  const hasSpaceRight = spaceRight >= needsHorizontalSpace

  if (preferredPlacement === 'bottom' && !hasSpaceBottom) {
    // Нет места снизу — пробуем top, потом left/right
    if (hasSpaceTop) {
      newPlacement = 'top'
    } else if (hasSpaceLeft) {
      newPlacement = 'left'
    } else if (hasSpaceRight) {
      newPlacement = 'right'
    }
    // Иначе оставляем bottom и clamp'им позицию
  } else if (preferredPlacement === 'top' && !hasSpaceTop) {
    // Нет места сверху — пробуем bottom, потом left/right
    if (hasSpaceBottom) {
      newPlacement = 'bottom'
    } else if (hasSpaceLeft) {
      newPlacement = 'left'
    } else if (hasSpaceRight) {
      newPlacement = 'right'
    }
    // Иначе оставляем top и clamp'им позицию
  } else if (preferredPlacement === 'right' && !hasSpaceRight) {
    newPlacement = hasSpaceBottom ? 'bottom' : hasSpaceLeft ? 'left' : 'top'
  } else if (preferredPlacement === 'left' && !hasSpaceLeft) {
    newPlacement = hasSpaceBottom ? 'bottom' : hasSpaceRight ? 'right' : 'top'
  }

  switch (newPlacement) {
    case 'top':
      top = rect.top - tooltipRect.height - arrowSize - padding
      left = rect.left + rect.width / 2 - effectiveTooltipWidth / 2
      break
    case 'bottom':
      top = rect.bottom + arrowSize + padding
      left = rect.left + rect.width / 2 - effectiveTooltipWidth / 2
      break
    case 'left':
      top = rect.top + rect.height / 2 - tooltipRect.height / 2
      left = rect.left - effectiveTooltipWidth - arrowSize - padding
      break
    case 'right':
      top = rect.top + rect.height / 2 - tooltipRect.height / 2
      left = rect.right + arrowSize + padding
      break
  }

  // Ограничиваем границами экрана с учётом отступа
  const maxLeft = window.innerWidth - effectiveTooltipWidth - padding
  // Если экран уже чем tooltip + отступы, прижимаем к левому краю
  if (maxLeft < padding) {
    left = padding
  } else {
    left = Math.max(padding, Math.min(left, maxLeft))
  }
  top = Math.max(padding, Math.min(top, window.innerHeight - tooltipRect.height - bottomPadding))

  // Дополнительная проверка: если tooltip перекрывает целевой элемент, сдвигаем
  const tooltipBottom = top + tooltipRect.height
  const tooltipRight = left + effectiveTooltipWidth

  // Если tooltip перекрывает цель по вертикали
  if (top < rect.bottom && tooltipBottom > rect.top &&
      left < rect.right && tooltipRight > rect.left) {
    // Сдвигаем tooltip ниже целевого элемента
    top = rect.bottom + arrowSize + padding
    newPlacement = 'bottom'
  }

  // Финальное ограничение top после всех корректировок
  const maxTop = window.innerHeight - tooltipRect.height - padding
  if (top > maxTop) {
    top = Math.max(padding, maxTop)
  }

  position.value = { top, left }
  placement.value = newPlacement
}

// Следим за изменениями шага
watch(
  currentStep,
  async () => {
    if (currentStep.value) {
      await nextTick()
      // Небольшая задержка для отрисовки целевого элемента
      setTimeout(updatePosition, 100)
      // Повторная попытка для элементов в модалках/меню с анимацией (500ms)
      setTimeout(updatePosition, 600)
    }
  },
  { immediate: true }
)

// Следим за изменением направления popup для перепозиционирования tooltip
watch(
  popupDirection,
  async () => {
    if (currentStep.value) {
      await nextTick()
      updatePosition()
    }
  }
)

// Используем shared MutationObserver из popupTracker
let popupUnsubscribe: (() => void) | null = null

function setupPopupObserver() {
  if (popupUnsubscribe) return

  // Подписываемся на shared observer
  popupUnsubscribe = onPopupChange(() => {
    updatePosition()
  })
}

function cleanupPopupObserver() {
  if (popupUnsubscribe) {
    popupUnsubscribe()
    popupUnsubscribe = null
  }
}

// Throttled версия для scroll (много событий при прокрутке)
const throttledUpdatePosition = throttle(updatePosition, SCROLL_THROTTLE_DELAY)

// Обновляем позицию при скролле и ресайзе
onMounted(() => {
  window.addEventListener('scroll', throttledUpdatePosition, true)
  window.addEventListener('resize', updatePosition)
  setupPopupObserver()
})

onUnmounted(() => {
  window.removeEventListener('scroll', throttledUpdatePosition, true)
  window.removeEventListener('resize', updatePosition)
  cleanupPopupObserver()
})

function handleContinue() {
  onboarding.nextStep()
}

function handleSkip() {
  onboarding.skipStep()
}

function handleBack() {
  onboarding.prevStep()
}

async function startOffersTraining() {
  // СНАЧАЛА сохраняем ID созданной заявки (до completeScenario, который очищает его)
  const createdRequest = onboarding.sandboxCreatedRequest
  const frId = createdRequest?.id ?? 'sandbox-fr-offers'

  // ПОТОМ завершаем текущий сценарий (customer_flow)
  onboarding.completeScenario()

  // Офферы, уведомления и заявка создаются в initialize() сценария offers_receive_flow

  // НЕ переходим сразу на страницу заявки
  // Пользователь должен кликнуть на уведомление чтобы попасть туда
  // Возвращаемся на главную если не там
  if (router.currentRoute.value.path !== '/') {
    await router.push('/')
  }

  // Ждём загрузки страницы
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 100))

  // Входим в сценарий В РЕЖИМЕ ЦЕПОЧКИ
  await onboarding.enterSandbox('offers_receive_flow', 0, {
    freightRequestId: frId,
  })
}

async function startCompletionTraining() {
  // СНАЧАЛА сохраняем ID заявки (до completeScenario, который очищает chainContext)
  const frId = onboarding.getChainedFreightRequestId() ?? 'sandbox-fr-completion'

  // ПОТОМ завершаем текущий сценарий (offers_receive_flow)
  onboarding.completeScenario()

  // Переходим на страницу заявки (она уже confirmed после offersReceiveFlow)
  await router.push(`/freight-requests/${frId}`)

  // Ждём загрузки страницы
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 200))

  // Входим в completionFlow В РЕЖИМЕ ЦЕПОЧКИ с пропуском intro шага
  await onboarding.enterSandbox('completion_flow', 0, {
    freightRequestId: frId,
    skipIntro: true,
  })
}

const arrowClasses = computed(() => {
  const base = 'absolute w-3 h-3 bg-white border transform rotate-45'
  switch (placement.value) {
    case 'top':
      return `${base} bottom-[-6px] left-1/2 -translate-x-1/2 border-t-0 border-l-0`
    case 'bottom':
      return `${base} top-[-6px] left-1/2 -translate-x-1/2 border-b-0 border-r-0`
    case 'left':
      return `${base} right-[-6px] top-1/2 -translate-y-1/2 border-l-0 border-b-0`
    case 'right':
      return `${base} left-[-6px] top-1/2 -translate-y-1/2 border-r-0 border-t-0`
    default:
      return base
  }
})

// Есть ли target у текущего шага (для скрытия стрелки)
const hasTarget = computed(() => {
  return !!(currentStep.value?.target || currentStep.value?.highlightSelector)
})

// Показываем кнопку "Далее" для manual шагов, но НЕ если есть кнопка обучения по предложениям
const isManualStep = computed(() => {
  if (!currentStep.value) return false
  if (currentStep.value.showOffersTrainingButton) return false
  return currentStep.value.completionType === 'manual'
})

// Показываем кнопку "Пропустить" ТОЛЬКО если шаг явно skippable
const showSkipButton = computed(() => {
  if (!currentStep.value) return false
  if (currentStep.value.showOffersTrainingButton) return false
  return currentStep.value.skippable === true
})

// Показываем кнопку "Назад" если не на первом шаге и НЕ action шаг
const showBackButton = computed(() => {
  if (!currentStep.value) return false
  // Для action шагов не показываем кнопки навигации — ждём действия пользователя
  if (currentStep.value.completionType === 'action') return false
  // Явно скрыть "Назад" для шагов внутри модалок
  if (currentStep.value.hideBackButton) return false
  return currentStepIndex.value > 0
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-all duration-200 ease-out"
      leave-active-class="transition-all duration-150 ease-in"
      enter-from-class="opacity-0 scale-95"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="currentStep && !validationErrorMode && !currentStep.hideTooltip"
        ref="tooltipRef"
        class="fixed z-[70] max-h-[calc(100vh-200px)] overflow-y-auto rounded-lg border bg-white p-4 shadow-xl"
        :style="{
          top: `${Math.max(12, position.top)}px`,
          left: `${Math.max(12, position.left)}px`,
          maxWidth: `min(384px, calc(100vw - 24px))`,
        }"
      >
        <!-- Arrow (скрываем если нет target) -->
        <div v-if="hasTarget" :class="arrowClasses" />

        <!-- Content -->
        <div class="relative">
          <!-- Progress -->
          <div class="mb-2 text-xs text-muted-foreground">
            Шаг {{ currentStepIndex + 1 }} из {{ scenarioSteps.length }}
          </div>

          <!-- Title -->
          <h4 class="mb-2 font-medium text-foreground">
            {{ currentStep.title }}
          </h4>

          <!-- Description -->
          <p class="mb-4 text-sm text-muted-foreground">
            {{ currentStep.description }}
          </p>

          <!-- Hint -->
          <p v-if="currentStep.hint" class="mb-4 text-xs text-blue-600">
            {{ currentStep.hint }}
          </p>

          <!-- Actions -->
          <div v-if="showBackButton || showSkipButton || isManualStep" class="flex items-center justify-between pointer-events-auto">
            <div class="flex gap-1">
              <Button v-if="showBackButton" variant="ghost" size="sm" @click="handleBack">
                <ChevronLeft class="mr-1 h-4 w-4" />
                Назад
              </Button>
              <Button v-if="showSkipButton" variant="ghost" size="sm" @click="handleSkip">
                <SkipForward class="mr-1 h-4 w-4" />
                Пропустить
              </Button>
            </div>

            <Button
              v-if="isManualStep"
              variant="default"
              size="sm"
              @click="handleContinue"
            >
              Далее
              <ChevronRight class="ml-1 h-4 w-4" />
            </Button>
          </div>

          <!-- Кнопка перехода к обучению по предложениям -->
          <Button
            v-if="currentStep?.showOffersTrainingButton"
            variant="default"
            size="sm"
            class="w-full mt-3 pointer-events-auto"
            @click="startOffersTraining"
          >
            Пройти обучение по предложениям
          </Button>

          <!-- Кнопка перехода к обучению по завершению -->
          <Button
            v-if="currentStep?.showCompletionTrainingButton"
            variant="default"
            size="sm"
            class="w-full mt-3 pointer-events-auto"
            @click="startCompletionTraining"
          >
            Пройти обучение по завершению
          </Button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
