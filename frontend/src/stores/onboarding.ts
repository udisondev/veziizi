/**
 * Onboarding Store
 * Управление режимом обучения (sandbox mode)
 */

import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { tutorialBus } from '@/sandbox/events'
import { resetAllMockData } from '@/sandbox/mockData'
import { useNotificationsStore } from '@/stores/notifications'
import type {
  ScenarioType,
  TutorialStep,
  TutorialProgress,
  TutorialEventKey,
  ChainContext,
} from '@/types/tutorial'
import type { PopupDirection } from '@/composables/useTutorialPopupTracker'
import type { VehicleType, VehicleSubType, Currency } from '@/types/freightRequest'

// Breakpoint для desktop (md в Tailwind)
const DESKTOP_BREAKPOINT = 768

// localStorage ключи
const STORAGE_KEYS = {
  PROGRESS: 'veziizi_tutorial_progress',
  HAS_SEEN_WELCOME: 'veziizi_has_seen_welcome',
  HAS_SEEN_HELP_HINT: 'veziizi_has_seen_help_hint',
  SANDBOX_STATE: 'veziizi_sandbox_state',
} as const

// Тип модуля сценария
interface ScenarioModule {
  steps: TutorialStep[]
  initialize?: () => Promise<void> | void
}

// Ленивая загрузка сценариев (чтобы не грузить если не нужно)
const scenarioLoaders: Record<ScenarioType, () => Promise<{ default: ScenarioModule }>> = {
  customer_flow: () => import('@/sandbox/scenarios/customerFlow'),
  carrier_flow: () => import('@/sandbox/scenarios/carrierFlow'),
  offers_receive_flow: () => import('@/sandbox/scenarios/offersReceiveFlow'),
  admin_flow: () => import('@/sandbox/scenarios/adminFlow'),
  subscriptions_flow: () => import('@/sandbox/scenarios/subscriptionsFlow'),
  telegram_flow: () => import('@/sandbox/scenarios/telegramFlow'),
  completion_flow: () => import('@/sandbox/scenarios/completionFlow'),
}

export const useOnboardingStore = defineStore('onboarding', () => {
  const router = useRouter()

  // === State ===

  // Режим песочницы
  // ВАЖНО: Синхронно проверяем localStorage при создании store,
  // чтобы API interceptor работал с первого запроса
  let initialSandboxMode = false
  try {
    const savedState = localStorage.getItem(STORAGE_KEYS.SANDBOX_STATE)
    if (savedState) {
      initialSandboxMode = JSON.parse(savedState).isSandboxMode === true
    }
  } catch {
    // Поврежденные данные в localStorage — игнорируем
    localStorage.removeItem(STORAGE_KEYS.SANDBOX_STATE)
  }
  const isSandboxMode = ref(initialSandboxMode)

  // Desktop или mobile (реактивно отслеживается)
  const isDesktop = ref(window.innerWidth >= DESKTOP_BREAKPOINT)

  // Все шаги сценария (без фильтрации)
  const allSteps = ref<TutorialStep[]>([])

  // Текущий сценарий и шаги
  const activeScenario = ref<ScenarioType | null>(null)
  const currentStepIndex = ref(0)

  // Фильтрация шагов по платформе
  function filterStepsByPlatform(steps: TutorialStep[], desktop: boolean): TutorialStep[] {
    return steps.filter(step => {
      const platform = step.platform || 'all'
      if (platform === 'all') return true
      if (platform === 'desktop') return desktop
      if (platform === 'mobile') return !desktop
      return true
    })
  }

  // Отфильтрованные шаги для текущей платформы
  const currentSteps = computed(() => filterStepsByPlatform(allSteps.value, isDesktop.value))

  // UI состояние
  const isTooltipVisible = ref(false)
  const highlightedElement = ref<string | null>(null)

  // Направление открытого popup (для умного позиционирования tooltip)
  const popupDirection = ref<PopupDirection | null>(null)

  // Пауза автоскролла (когда скроллим к ошибке валидации)
  const scrollPaused = ref(false)

  // Режим исправления ошибки валидации - overlay полностью скрыт
  const validationErrorMode = ref(false)

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

  // Готовность sandbox (true если НЕ в sandbox или sandbox полностью инициализирован)
  // Если sandbox восстанавливается из localStorage, нужно подождать initialize()
  const sandboxReady = ref(!initialSandboxMode)

  // Обработчик текущего события
  let currentEventUnsubscribe: (() => void) | null = null

  // Фейковая заявка созданная в sandbox режиме
  const sandboxCreatedRequest = ref<{
    id: string
    origin_address: string
    destination_address: string
    cargo_weight: number
    price_amount?: number
    price_currency?: Currency
    vehicle_type: VehicleType
    vehicle_subtype: VehicleSubType
    status: string
    created_at: string
  } | null>(null)

  // Контекст цепочки курсов (для передачи данных между связанными курсами)
  const chainContext = ref<ChainContext | null>(null)

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
  async function loadProgress() {
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
          // Восстанавливаем состояние sandbox с chainContext если есть
          // ВАЖНО: await чтобы sandbox был готов до рендера компонентов
          const chain = state.chainContext?.freightRequestId
            ? { freightRequestId: state.chainContext.freightRequestId, skipIntro: state.chainContext.skipIntro }
            : undefined
          await enterSandbox(state.activeScenario, state.currentStepIndex, chain)
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
          chainContext: chainContext.value,
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
   * Обработчик resize для обновления isDesktop
   */
  function handleResize() {
    const wasDesktop = isDesktop.value
    isDesktop.value = window.innerWidth >= DESKTOP_BREAKPOINT

    // Если платформа изменилась во время туториала, обновляем индекс
    if (wasDesktop !== isDesktop.value && isSandboxMode.value) {
      // Корректируем индекс чтобы остаться на аналогичном шаге
      const currentId = currentStep.value?.id
      if (currentId) {
        const newIndex = currentSteps.value.findIndex(s => s.id === currentId)
        if (newIndex >= 0) {
          currentStepIndex.value = newIndex
        } else {
          // Если текущий шаг скрылся, переходим к первому доступному
          currentStepIndex.value = 0
        }
      }
    }
  }

  // Подписка на resize при инициализации store
  if (typeof window !== 'undefined') {
    window.addEventListener('resize', handleResize)
  }

  /**
   * Войти в режим песочницы
   * @param scenario - тип сценария
   * @param resumeFromStep - с какого шага начать (по умолчанию 0)
   * @param chain - контекст цепочки курсов (опционально)
   */
  async function enterSandbox(
    scenario: ScenarioType,
    resumeFromStep = 0,
    chain?: { freightRequestId: string; skipIntro?: boolean }
  ) {
    // Помечаем sandbox как не готовый пока не завершится initialize()
    sandboxReady.value = false

    try {
      // Устанавливаем chain context ДО initialize() и загрузки сценария
      if (chain) {
        chainContext.value = {
          isChained: true,
          freightRequestId: chain.freightRequestId,
          skipIntro: chain.skipIntro ?? false,
        }
      } else {
        chainContext.value = null
      }

      // Загружаем сценарий
      const module = await scenarioLoaders[scenario]()
      allSteps.value = module.default.steps

      // ВАЖНО: включаем sandbox mode ДО initialize(),
      // чтобы API interceptor перехватывал запросы
      activeScenario.value = scenario
      currentStepIndex.value = resumeFromStep
      isSandboxMode.value = true

      // Инициализация сценария (создание mock данных и т.д.)
      // Выполняется при ЛЮБОМ старте/возобновлении
      if (module.default.initialize) {
        await module.default.initialize()
      }

      // Если skipIntro — начинаем со следующего шага (пропускаем intro)
      if (chain?.skipIntro && resumeFromStep === 0) {
        currentStepIndex.value = 1
      }

      // Sandbox готов — mock данные созданы
      sandboxReady.value = true

      // Показываем первый шаг
      await showCurrentStep()

      saveProgress()
    } catch (e) {
      // При ошибке помечаем sandbox как готовый, чтобы не блокировать компоненты
      sandboxReady.value = true
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
    // 1. Сначала выключаем sandbox mode — все новые запросы пойдут к реальному API
    isSandboxMode.value = false

    // 2. Очищаем все mock данные
    resetAllMockData()

    // 3. Сбрасываем состояние store
    activeScenario.value = null
    allSteps.value = []
    currentStepIndex.value = 0
    isTooltipVisible.value = false
    highlightedElement.value = null
    popupDirection.value = null
    sandboxCreatedRequest.value = null
    chainContext.value = null

    if (currentEventUnsubscribe) {
      currentEventUnsubscribe()
      currentEventUnsubscribe = null
    }

    // 4. Очищаем Pinia notifications store и перезагружаем с реального сервера
    const notificationsStore = useNotificationsStore()
    notificationsStore.cleanup()
    notificationsStore.initialize()

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

    // Редирект на главную после завершения (sandbox данные невалидны вне sandbox)
    router.push('/')
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
    price_currency?: Currency
    vehicle_type: VehicleType
    vehicle_subtype: VehicleSubType
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

  /**
   * Пауза автоскролла overlay (для скролла к ошибкам валидации)
   */
  function pauseAutoScroll(durationMs: number = 1000) {
    scrollPaused.value = true
    setTimeout(() => {
      scrollPaused.value = false
    }, durationMs)
  }

  /**
   * Войти в режим исправления ошибки валидации
   * Overlay и tooltip полностью скрываются, пользователь может редактировать форму
   */
  function enterValidationErrorMode() {
    validationErrorMode.value = true
    scrollPaused.value = true
  }

  /**
   * Выйти из режима исправления ошибки валидации
   */
  function exitValidationErrorMode() {
    validationErrorMode.value = false
    scrollPaused.value = false
  }

  /**
   * Получить ID заявки из chain context (для цепочки курсов)
   */
  function getChainedFreightRequestId(): string | null {
    return chainContext.value?.freightRequestId ?? null
  }

  /**
   * Проверить, запущен ли сценарий в режиме цепочки
   */
  function isInChainMode(): boolean {
    return chainContext.value?.isChained ?? false
  }

  return {
    // State
    isSandboxMode,
    sandboxReady,
    activeScenario,
    currentSteps,
    currentStepIndex,
    isTooltipVisible,
    highlightedElement,
    popupDirection,
    scrollPaused,
    validationErrorMode,
    progress,
    hasSeenWelcome,
    hasSeenHelpHint,
    hasCompletedOnboarding,
    sandboxCreatedRequest,
    chainContext,
    isDesktop,

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
    pauseAutoScroll,
    enterValidationErrorMode,
    exitValidationErrorMode,
    setSandboxCreatedRequest,
    clearSandboxCreatedRequest,
    getChainedFreightRequestId,
    isInChainMode,
  }
})
