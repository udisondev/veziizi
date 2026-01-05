/**
 * Tutorial System Types
 * Типы для системы интерактивного обучения (sandbox mode)
 */

// Типы сценариев обучения
export type ScenarioType =
  | 'customer_flow'
  | 'carrier_flow'
  | 'offers_receive_flow'
  | 'admin_flow'
  | 'orders_flow'
  | 'subscriptions_flow'
  | 'telegram_flow'

// Тип завершения шага
export type StepCompletionType =
  | 'action'    // Ждём определённое действие (событие)
  | 'navigate'  // Ждём навигацию на определённый route
  | 'manual'    // Пользователь нажимает "Далее"

// Позиция tooltip относительно элемента
export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right'

// Шаг обучения
export interface TutorialStep {
  id: string
  title: string
  description: string

  // Подсказка для шага
  hint?: string

  // Навигация (опционально)
  route?: string

  // UI подсказка
  target?: string           // data-tutorial="target" — для подсветки элемента
  highlightSelector?: string // CSS селектор (устаревший, использовать target)
  tooltipPosition?: TooltipPosition

  // Условия завершения
  completionType: StepCompletionType
  completionAction?: string // Событие которое завершает шаг

  // Симуляции (callbacks)
  beforeStep?: () => Promise<void> | void
  afterStep?: () => Promise<void> | void

  // Задержка для автосимуляций (ms)
  simulationDelay?: number

  // Можно ли пропустить шаг
  skippable?: boolean

  // Показать кнопку перехода к обучению по предложениям
  showOffersTrainingButton?: boolean
}

// Сценарий обучения
export interface Scenario {
  id: ScenarioType
  name: string
  description: string
  icon?: string
  steps: TutorialStep[]
  requiredRole?: 'owner' | 'administrator' | 'employee'
}

// Прогресс обучения
export interface TutorialProgress {
  completedScenarios: ScenarioType[]
  currentScenarioProgress: Record<string, boolean> // stepId -> completed
  lastActiveStep: {
    scenario: ScenarioType
    stepIndex: number
  } | null
}

// Состояние onboarding store
export interface OnboardingState {
  // Режим песочницы
  isSandboxMode: boolean

  // Прогресс обучения
  progress: TutorialProgress

  // Текущий сценарий
  activeScenario: ScenarioType | null

  // Текущий шаг
  currentStepIndex: number

  // Состояние UI подсказок
  isTooltipVisible: boolean
  highlightedElement: string | null

  // Первый вход
  hasSeenWelcome: boolean
  hasCompletedOnboarding: boolean
}

// Конфигурация tooltip
export interface TooltipConfig {
  title: string
  description: string
  position: TooltipPosition
  showSkip?: boolean
  showNext?: boolean
}

// События tutorial системы
export interface TutorialEvents {
  // Wizard events
  'wizard:next': void
  'wizard:prev': void
  'wizard:submit': void

  // FreightRequest events
  'freightRequest:created': { id: string }
  'freightRequest:cancelled': { id: string }

  // Offer events
  'offer:created': { frId: string; offerId: string }
  'offer:selected': { frId: string; offerId: string }
  'offer:rejected': { frId: string; offerId: string }
  'offer:unselected': { frId: string; offerId: string }
  'offer:confirmed': { frId: string; offerId: string }
  'offer:declined': { frId: string; offerId: string }
  'offer:withdrawn': { frId: string; offerId: string }

  // Order events
  'order:created': { id: string }
  'order:completed': { orderId: string }
  'order:cancelled': { orderId: string }

  // Message events
  'message:sent': { orderId: string }

  // Document events
  'document:uploaded': { orderId: string; docId: string }
  'document:deleted': { orderId: string; docId: string }

  // Review events
  'review:left': { orderId: string }

  // Tab navigation
  'tab:offers': void
  'tab:messages': void
  'tab:documents': void
  'tab:reviews': void

  // Filters
  'filters:applied': void
  'filters:cleared': void

  // Subscriptions
  'subscription:created': { id: string }
  'subscription:deleted': { id: string }

  // Member management
  'invitation:created': { invitationId: string; email: string }
  'member:roleChanged': { memberId: string; newRole: string }
  'member:blocked': { memberId: string }
  'member:unblocked': { memberId: string }

  // Telegram
  'telegram:linkRequested': void
  'telegram:connected': void

  // Navigation
  'navigate': { path: string }

  // Menu events
  'menu:opened': void
  'menu:closed': void

  // Route step events (детальный туториал маршрута)
  'route:citySelected': { pointIndex: number }
  'route:dateSet': { pointIndex: number }
  'route:timeToggled': { pointIndex: number; shown: boolean }
  'route:contactToggled': { pointIndex: number; shown: boolean }
  'route:commentToggled': { pointIndex: number; shown: boolean }
  'route:pointAdded': { newIndex: number }
  'route:pointsReordered': void
}

// Ключи событий
export type TutorialEventKey = keyof TutorialEvents

// localStorage ключи
export const STORAGE_KEYS = {
  PROGRESS: 'veziizi_tutorial_progress',
  HAS_SEEN_WELCOME: 'veziizi_has_seen_welcome',
  HAS_SEEN_HELP_HINT: 'veziizi_has_seen_help_hint',
  SANDBOX_STATE: 'veziizi_sandbox_state',
} as const

// Mock организации для sandbox
export interface MockOrganization {
  id: string
  name: string
  inn: string
}

// Mock пользователь для sandbox
export interface MockMember {
  id: string
  name: string
  email: string
  phone?: string
  organizationId: string
}

// Mock контрагенты (перевозчики/заказчики)
export const MOCK_COUNTERPARTIES = [
  { id: 'carrier-1', name: 'ТрансЛогистик', memberName: 'Иван Петров' },
  { id: 'carrier-2', name: 'СпецГруз', memberName: 'Алексей Смирнов' },
  { id: 'carrier-3', name: 'МегаФура', memberName: 'Дмитрий Козлов' },
  { id: 'customer-1', name: 'ООО Ромашка', memberName: 'Мария Иванова' },
  { id: 'customer-2', name: 'ИП Сидоров', memberName: 'Пётр Сидоров' },
] as const
