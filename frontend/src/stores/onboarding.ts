/**
 * Onboarding Store
 * Управление режимом обучения (sandbox mode)
 */

import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { tutorialBus } from '@/sandbox/events'
import type {
  ScenarioType,
  TutorialStep,
  TutorialProgress,
  STORAGE_KEYS,
  TutorialEventKey,
} from '@/types/tutorial'
import type { PopupDirection } from '@/composables/useTutorialPopupTracker'

// localStorage ключи
const STORAGE_KEYS = {
  PROGRESS: 'veziizi_tutorial_progress',
  HAS_SEEN_WELCOME: 'veziizi_has_seen_welcome',
  HAS_SEEN_HELP_HINT: 'veziizi_has_seen_help_hint',
  SANDBOX_STATE: 'veziizi_sandbox_state',
} as const

// Ленивая загрузка сценариев (чтобы не грузить если не нужно)
const scenarioLoaders: Record<ScenarioType, () => Promise<{ default: { steps: TutorialStep[] } }>> = {
  customer_flow: () => import('@/sandbox/scenarios/customerFlow'),
  carrier_flow: () => import('@/sandbox/scenarios/carrierFlow'),
  offers_receive_flow: () => import('@/sandbox/scenarios/offersReceiveFlow'),
  admin_flow: () => import('@/sandbox/scenarios/adminFlow'),
  orders_flow: () => import('@/sandbox/scenarios/ordersFlow'),
  subscriptions_flow: () => import('@/sandbox/scenarios/subscriptionsFlow'),
  telegram_flow: () => import('@/sandbox/scenarios/telegramFlow'),
}

export const useOnboardingStore = defineStore('onboarding', () => {
  const router = useRouter()

  // === State ===

  // Режим песочницы
  const isSandboxMode = ref(false)

  // Текущий сценарий и шаги
  const activeScenario = ref<ScenarioType | null>(null)
  const currentSteps = ref<TutorialStep[]>([])
  const currentStepIndex = ref(0)

  // UI состояние
  const isTooltipVisible = ref(false)
  const highlightedElement = ref<string | null>(null)

  // Направление открытого popup (для умного позиционирования tooltip)
  const popupDirection = ref<PopupDirection | null>(null)

  // Прогресс
  const progress = ref<TutorialProgress>({
    completedScenarios: [],
    currentScenarioProgress: {},
    lastActiveStep: null,
  })

  // Флаги первого входа
  const hasSeenWelcome = ref(false)
  const hasSeenHelpHint = ref(false)
  const hasCompletedOnboarding = ref(false)

  // Обработчик текущего события
  let currentEventUnsubscribe: (() => void) | null = null

  // Фейковая заявка созданная в sandbox режиме
  const sandboxCreatedRequest = ref<{
    id: string
    origin_address: string
    destination_address: string
    cargo_weight: number
    price_amount?: number
    price_currency?: string
    vehicle_type: string
    vehicle_subtype: string
    status: string
    created_at: string
  } | null>(null)

  // === Computed ===

  const currentStep = computed<TutorialStep | null>(() => {
    if (!currentSteps.value.length) return null
    return currentSteps.value[currentStepIndex.value] || null
  })

  const totalSteps = computed(() => currentSteps.value.length)

  // Алиас для UI компонентов
  const scenarioSteps = currentSteps

  const isLastStep = computed(() => currentStepIndex.value >= totalSteps.value - 1)

  const progressPercent = computed(() => {
    if (!totalSteps.value) return 0
    return Math.round(((currentStepIndex.value + 1) / totalSteps.value) * 100)
  })

  const canResume = computed(() => progress.value.lastActiveStep !== null)

  // === Actions ===

  /**
   * Загрузить прогресс из localStorage
   */
  function loadProgress() {
    try {
      const savedProgress = localStorage.getItem(STORAGE_KEYS.PROGRESS)
      if (savedProgress) {
        progress.value = JSON.parse(savedProgress)
      }

      const savedWelcome = localStorage.getItem(STORAGE_KEYS.HAS_SEEN_WELCOME)
      hasSeenWelcome.value = savedWelcome === 'true'

      const savedHelpHint = localStorage.getItem(STORAGE_KEYS.HAS_SEEN_HELP_HINT)
      hasSeenHelpHint.value = savedHelpHint === 'true'

      const savedState = localStorage.getItem(STORAGE_KEYS.SANDBOX_STATE)
      if (savedState) {
        const state = JSON.parse(savedState)
        if (state.isSandboxMode && state.activeScenario) {
          // Восстанавливаем состояние sandbox
          enterSandbox(state.activeScenario, state.currentStepIndex)
        }
      }
    } catch (e) {
      console.error('Failed to load onboarding progress:', e)
    }
  }

  /**
   * Сохранить прогресс в localStorage
   */
  function saveProgress() {
    try {
      localStorage.setItem(STORAGE_KEYS.PROGRESS, JSON.stringify(progress.value))

      if (isSandboxMode.value) {
        localStorage.setItem(STORAGE_KEYS.SANDBOX_STATE, JSON.stringify({
          isSandboxMode: true,
          activeScenario: activeScenario.value,
          currentStepIndex: currentStepIndex.value,
        }))
      } else {
        localStorage.removeItem(STORAGE_KEYS.SANDBOX_STATE)
      }
    } catch (e) {
      console.error('Failed to save onboarding progress:', e)
    }
  }

  /**
   * Отметить welcome модалку как просмотренную
   */
  function markWelcomeSeen() {
    hasSeenWelcome.value = true
    localStorage.setItem(STORAGE_KEYS.HAS_SEEN_WELCOME, 'true')
  }

  /**
   * Отметить подсказку кнопки помощи как просмотренную
   */
  function markHelpHintSeen() {
    hasSeenHelpHint.value = true
    localStorage.setItem(STORAGE_KEYS.HAS_SEEN_HELP_HINT, 'true')
  }

  /**
   * Войти в режим песочницы
   */
  async function enterSandbox(scenario: ScenarioType, resumeFromStep = 0) {
    try {
      // Загружаем сценарий
      const module = await scenarioLoaders[scenario]()
      currentSteps.value = module.default.steps

      activeScenario.value = scenario
      currentStepIndex.value = resumeFromStep
      isSandboxMode.value = true

      // Показываем первый шаг
      await showCurrentStep()

      saveProgress()
    } catch (e) {
      console.error('Failed to enter sandbox:', e)
    }
  }

  /**
   * Выйти из режима песочницы
   */
  function exitSandbox() {
    // Сохраняем позицию для возможного возврата
    if (activeScenario.value !== null) {
      progress.value.lastActiveStep = {
        scenario: activeScenario.value,
        stepIndex: currentStepIndex.value,
      }
    }

    cleanup()
    saveProgress()
  }

  /**
   * Очистить состояние sandbox
   */
  function cleanup() {
    isSandboxMode.value = false
    activeScenario.value = null
    currentSteps.value = []
    currentStepIndex.value = 0
    isTooltipVisible.value = false
    highlightedElement.value = null
    popupDirection.value = null
    sandboxCreatedRequest.value = null

    if (currentEventUnsubscribe) {
      currentEventUnsubscribe()
      currentEventUnsubscribe = null
    }

    localStorage.removeItem(STORAGE_KEYS.SANDBOX_STATE)
  }

  /**
   * Сбросить весь прогресс
   */
  function resetProgress() {
    progress.value = {
      completedScenarios: [],
      currentScenarioProgress: {},
      lastActiveStep: null,
    }
    hasCompletedOnboarding.value = false
    cleanup()
    localStorage.removeItem(STORAGE_KEYS.PROGRESS)
    localStorage.removeItem(STORAGE_KEYS.SANDBOX_STATE)
  }

  /**
   * Показать текущий шаг
   */
  async function showCurrentStep() {
    const step = currentStep.value
    if (!step) return

    // Отписываемся от предыдущего события
    if (currentEventUnsubscribe) {
      currentEventUnsubscribe()
      currentEventUnsubscribe = null
    }

    // Выполняем beforeStep если есть
    if (step.beforeStep) {
      await step.beforeStep()
    }

    // Навигация если нужна
    if (step.route) {
      const currentPath = router.currentRoute.value.path
      // Заменяем :id на sandbox-id для динамических маршрутов
      const targetPath = step.route.replace(/:(\w+)/g, 'sandbox-$1')

      // Исправленная логика: для '/' точное сравнение
      const shouldNavigate = targetPath === '/'
        ? currentPath !== '/'
        : !currentPath.startsWith(targetPath)

      if (shouldNavigate) {
        await router.push(targetPath)
        // Даём время на рендеринг страницы
        await new Promise(resolve => setTimeout(resolve, 100))
      }
    }

    // Подсветка элемента — поддерживаем и target и highlightSelector
    const selector = step.target
      ? `[data-tutorial="${step.target}"]`
      : step.highlightSelector

    if (selector) {
      highlightedElement.value = selector
      isTooltipVisible.value = true
    } else {
      highlightedElement.value = null
      isTooltipVisible.value = false
    }

    // Подписываемся на событие завершения
    if (step.completionType === 'action' && step.completionAction) {
      const eventName = step.completionAction as TutorialEventKey
      const handler = () => {
        nextStep()
      }
      tutorialBus.on(eventName, handler)
      currentEventUnsubscribe = () => tutorialBus.off(eventName, handler)
    }

    // Для navigate типа — слушаем router
    if (step.completionType === 'navigate' && step.completionAction) {
      const targetPath = step.completionAction
      const unwatch = watch(
        () => router.currentRoute.value.path,
        (newPath) => {
          // Для корневого пути '/' — точное сравнение
          // Для остальных — проверяем начало пути (с учётом динамических параметров)
          const normalizedTarget = targetPath.replace(/:(\w+)/g, '')
          const isMatch = targetPath === '/'
            ? newPath === '/'
            : newPath.startsWith(normalizedTarget) || newPath.includes(normalizedTarget)

          if (isMatch) {
            unwatch()
            nextStep()
          }
        },
        { immediate: true }
      )
      currentEventUnsubscribe = unwatch
    }

    // Для manual — шаг завершается вручную кнопкой "Далее"
    // Для симуляций с delay
    if (step.completionType === 'manual' && step.simulationDelay) {
      setTimeout(() => {
        if (currentStep.value?.id === step.id) {
          nextStep()
        }
      }, step.simulationDelay)
    }
  }

  /**
   * Перейти к следующему шагу
   */
  async function nextStep() {
    const step = currentStep.value
    if (!step) return

    // Отмечаем шаг как завершённый
    progress.value.currentScenarioProgress[step.id] = true

    // Выполняем afterStep если есть
    if (step.afterStep) {
      await step.afterStep()
    }

    // Переходим к следующему шагу или завершаем
    if (isLastStep.value) {
      completeScenario()
    } else {
      currentStepIndex.value++
      await showCurrentStep()
    }

    saveProgress()
  }

  /**
   * Пропустить текущий шаг
   */
  async function skipStep() {
    if (currentEventUnsubscribe) {
      currentEventUnsubscribe()
      currentEventUnsubscribe = null
    }

    if (isLastStep.value) {
      completeScenario()
    } else {
      currentStepIndex.value++
      await showCurrentStep()
    }

    saveProgress()
  }

  /**
   * Перейти к предыдущему шагу
   */
  async function prevStep() {
    if (currentStepIndex.value > 0) {
      if (currentEventUnsubscribe) {
        currentEventUnsubscribe()
        currentEventUnsubscribe = null
      }

      currentStepIndex.value--
      await showCurrentStep()
      saveProgress()
    }
  }

  /**
   * Перейти к конкретному шагу
   */
  async function goToStep(index: number) {
    if (index >= 0 && index < totalSteps.value) {
      if (currentEventUnsubscribe) {
        currentEventUnsubscribe()
        currentEventUnsubscribe = null
      }

      currentStepIndex.value = index
      await showCurrentStep()
      saveProgress()
    }
  }

  /**
   * Завершить сценарий
   */
  function completeScenario() {
    if (activeScenario.value) {
      if (!progress.value.completedScenarios.includes(activeScenario.value)) {
        progress.value.completedScenarios.push(activeScenario.value)
      }
      progress.value.lastActiveStep = null
    }

    cleanup()
    saveProgress()

    // Проверяем, все ли сценарии пройдены
    const allScenarios: ScenarioType[] = ['customer_flow', 'carrier_flow']
    const allCompleted = allScenarios.every(s =>
      progress.value.completedScenarios.includes(s)
    )
    if (allCompleted) {
      hasCompletedOnboarding.value = true
    }
  }

  /**
   * Пропустить обучение
   */
  function skipOnboarding() {
    markWelcomeSeen()
    cleanup()
  }

  /**
   * Возобновить прерванное обучение
   */
  async function resumeOnboarding() {
    const last = progress.value.lastActiveStep
    if (last) {
      await enterSandbox(last.scenario, last.stepIndex)
    }
  }

  /**
   * Установить фейковую заявку созданную в sandbox
   */
  function setSandboxCreatedRequest(request: {
    id: string
    origin_address: string
    destination_address: string
    cargo_weight: number
    price_amount?: number
    price_currency?: string
    vehicle_type: string
    vehicle_subtype: string
    created_at: string
  }) {
    sandboxCreatedRequest.value = {
      ...request,
      status: 'published',
    }
  }

  /**
   * Очистить фейковую заявку
   */
  function clearSandboxCreatedRequest() {
    sandboxCreatedRequest.value = null
  }

  /**
   * Показать tooltip для элемента
   */
  function showTooltip(selector: string) {
    highlightedElement.value = selector
    isTooltipVisible.value = true
  }

  /**
   * Скрыть tooltip
   */
  function hideTooltip() {
    isTooltipVisible.value = false
    highlightedElement.value = null
  }

  /**
   * Установить направление открытого popup
   */
  function setPopupDirection(direction: PopupDirection | null) {
    popupDirection.value = direction
  }

  return {
    // State
    isSandboxMode,
    activeScenario,
    currentSteps,
    currentStepIndex,
    isTooltipVisible,
    highlightedElement,
    popupDirection,
    progress,
    hasSeenWelcome,
    hasSeenHelpHint,
    hasCompletedOnboarding,
    sandboxCreatedRequest,

    // Computed
    currentStep,
    totalSteps,
    scenarioSteps,
    isLastStep,
    progressPercent,
    canResume,

    // Actions
    loadProgress,
    saveProgress,
    markWelcomeSeen,
    markHelpHintSeen,
    enterSandbox,
    exitSandbox,
    cleanup,
    resetProgress,
    showCurrentStep,
    nextStep,
    skipStep,
    prevStep,
    goToStep,
    completeScenario,
    skipOnboarding,
    resumeOnboarding,
    showTooltip,
    hideTooltip,
    setPopupDirection,
    setSandboxCreatedRequest,
    clearSandboxCreatedRequest,
  }
})
