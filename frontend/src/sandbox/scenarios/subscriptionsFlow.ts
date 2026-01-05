/**
 * Subscriptions Flow Scenario
 * Сценарий обучения для подписок на заявки
 */

import type { TutorialStep } from './types'

export const steps: TutorialStep[] = [
  {
    id: 'subscriptions_start',
    title: 'Подписки на заявки',
    description: 'Настройте автоматические уведомления о новых заявках по вашим критериям.',
    route: '/',
    highlightSelector: '[data-tutorial="subscriptions-link"]',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/subscriptions',
  },
  {
    id: 'subscriptions_create',
    title: 'Создание подписки',
    description: 'Настройте фильтры и создайте подписку. Вы будете получать уведомления о подходящих заявках.',
    highlightSelector: '[data-tutorial="create-subscription-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'subscription:created',
  },
  {
    id: 'subscriptions_complete',
    title: 'Готово!',
    description: 'Теперь вы будете получать уведомления о новых заявках, соответствующих вашим критериям.',
    completionType: 'manual',
  },
]

export default { steps }
