<script setup lang="ts">
/**
 * TutorialTooltip
 * Подсказка с инструкцией для текущего шага
 */

import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useOnboardingStore } from '@/stores/onboarding'
import { storeToRefs } from 'pinia'
import { Button } from '@/components/ui/button'
import { ChevronRight, SkipForward } from 'lucide-vue-next'
import { useTutorialPopupTracker } from '@/composables/useTutorialPopupTracker'

const onboarding = useOnboardingStore()
const { currentStep, currentStepIndex, scenarioSteps, popupDirection } = storeToRefs(onboarding)
const popupTracker = useTutorialPopupTracker()

const tooltipRef = ref<HTMLDivElement | null>(null)
const position = ref({ top: 0, left: 0 })
const placement = ref<'top' | 'bottom' | 'left' | 'right'>('bottom')

// Прокручиваем к элементу если он не виден
async function scrollToTargetIfNeeded(target: Element): Promise<void> {
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

  if (!targetSelector) return

  const target = document.querySelector(targetSelector)
  if (!target) {
    console.warn(`[Tutorial] Target not found: ${targetSelector}`)
    return
  }

  // Прокручиваем к элементу перед позиционированием tooltip
  await scrollToTargetIfNeeded(target)

  const targetRect = target.getBoundingClientRect()
  const tooltip = tooltipRef.value
  if (!tooltip) return

  // Анализируем popup рядом с target
  const analysis = popupTracker.analyzeTarget(targetRect)

  const tooltipRect = tooltip.getBoundingClientRect()
  const padding = 12
  const arrowSize = 8
  const bottomPadding = 100 // Отступ снизу для кнопок tooltip

  // Выбираем rect для позиционирования:
  // - Если есть popup, используем TARGET rect но с учётом направления popup
  // - Это держит tooltip близко к полю ввода, а не к огромному combined rect
  const rect = targetRect

  // Определяем лучшую позицию
  let spaceTop = rect.top
  let spaceBottom = window.innerHeight - rect.bottom
  const spaceLeft = rect.left
  const spaceRight = window.innerWidth - rect.right

  let preferredPlacement = currentStep.value.tooltipPosition || 'bottom'

  // Если есть открытый popup — позиционируем tooltip на противоположной стороне от popup
  if (analysis.hasPopups && analysis.primaryDirection) {
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
  const needsHorizontalSpace = tooltipRect.width + padding
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
      left = rect.left + rect.width / 2 - tooltipRect.width / 2
      break
    case 'bottom':
      top = rect.bottom + arrowSize + padding
      left = rect.left + rect.width / 2 - tooltipRect.width / 2
      break
    case 'left':
      top = rect.top + rect.height / 2 - tooltipRect.height / 2
      left = rect.left - tooltipRect.width - arrowSize - padding
      break
    case 'right':
      top = rect.top + rect.height / 2 - tooltipRect.height / 2
      left = rect.right + arrowSize + padding
      break
  }

  // Ограничиваем границами экрана с учётом отступа снизу
  left = Math.max(padding, Math.min(left, window.innerWidth - tooltipRect.width - padding))
  top = Math.max(padding, Math.min(top, window.innerHeight - tooltipRect.height - bottomPadding))

  // Дополнительная проверка: если tooltip перекрывает целевой элемент, сдвигаем
  const tooltipBottom = top + tooltipRect.height
  const tooltipRight = left + tooltipRect.width

  // Если tooltip перекрывает цель по вертикали
  if (top < rect.bottom && tooltipBottom > rect.top &&
      left < rect.right && tooltipRight > rect.left) {
    // Сдвигаем tooltip ниже целевого элемента
    top = rect.bottom + arrowSize + padding
    newPlacement = 'bottom'
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

// MutationObserver для отслеживания появления/исчезновения popup
let popupObserver: MutationObserver | null = null

function setupPopupObserver() {
  if (popupObserver) return

  popupObserver = new MutationObserver((mutations) => {
    // Проверяем, появился или исчез popup
    for (const mutation of mutations) {
      if (mutation.type === 'childList') {
        // Проверяем добавленные узлы на наличие popup
        for (const node of mutation.addedNodes) {
          if (node instanceof Element) {
            const isPopup =
              node.matches('[data-reka-popper-content-wrapper]') ||
              node.matches('[data-state="open"][role="listbox"]') ||
              node.querySelector('[data-reka-popper-content-wrapper]') ||
              node.querySelector('[data-state="open"][role="listbox"]')
            if (isPopup) {
              // Даём время на позиционирование popup
              setTimeout(updatePosition, 50)
              return
            }
          }
        }
        // Проверяем удалённые узлы
        for (const node of mutation.removedNodes) {
          if (node instanceof Element) {
            const wasPopup =
              node.matches?.('[data-reka-popper-content-wrapper]') ||
              node.matches?.('[data-state][role="listbox"]')
            if (wasPopup) {
              setTimeout(updatePosition, 50)
              return
            }
          }
        }
      }
      // Проверяем изменение атрибута data-state
      if (mutation.type === 'attributes' && mutation.attributeName === 'data-state') {
        setTimeout(updatePosition, 50)
        return
      }
    }
  })

  popupObserver.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['data-state'],
  })
}

function cleanupPopupObserver() {
  if (popupObserver) {
    popupObserver.disconnect()
    popupObserver = null
  }
}

// Обновляем позицию при скролле и ресайзе
onMounted(() => {
  window.addEventListener('scroll', updatePosition, true)
  window.addEventListener('resize', updatePosition)
  setupPopupObserver()
})

onUnmounted(() => {
  window.removeEventListener('scroll', updatePosition, true)
  window.removeEventListener('resize', updatePosition)
  cleanupPopupObserver()
})

function handleContinue() {
  onboarding.nextStep()
}

function handleSkip() {
  onboarding.skipStep()
}

function startOffersTraining() {
  // Завершаем текущий сценарий и переключаемся на offers_receive_flow
  onboarding.completeScenario()
  onboarding.enterSandbox('offers_receive_flow')
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

const isManualStep = computed(() => currentStep.value?.completionType === 'manual')
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
        v-if="currentStep"
        ref="tooltipRef"
        class="fixed z-[70] max-w-sm max-h-[calc(100vh-200px)] overflow-y-auto rounded-lg border bg-white p-4 shadow-xl pointer-events-none"
        :style="{
          top: `${position.top}px`,
          left: `${position.left}px`,
        }"
      >
        <!-- Arrow -->
        <div :class="arrowClasses" />

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
          <div class="flex items-center justify-between pointer-events-auto">
            <Button variant="ghost" size="sm" @click="handleSkip">
              <SkipForward class="mr-1 h-4 w-4" />
              Пропустить
            </Button>

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
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
