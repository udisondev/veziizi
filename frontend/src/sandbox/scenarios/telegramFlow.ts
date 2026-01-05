/**
 * Telegram Flow Scenario
 * Сценарий обучения для подключения Telegram
 */

import type { TutorialStep } from './types'

export const steps: TutorialStep[] = [
  {
    id: 'telegram_start',
    title: 'Подключение Telegram',
    description: 'Подключите Telegram для получения уведомлений. Перейдём в настройки уведомлений.',
    route: '/settings/notifications',
    highlightSelector: '[data-tutorial="notifications-settings-link"]',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/settings/notifications',
  },
  {
    id: 'telegram_connect',
    title: 'Получение кода',
    description: 'Нажмите "Подключить Telegram" чтобы получить код для бота.',
    highlightSelector: '[data-tutorial="connect-telegram-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'telegram:linkRequested',
  },
  {
    id: 'telegram_complete',
    title: 'Готово!',
    description: 'В реальном приложении вам нужно отправить код боту в Telegram. Теперь вы будете получать уведомления в мессенджер.',
    completionType: 'manual',
    async beforeStep() {
      // Симулируем успешное подключение
    },
  },
]

export default { steps }
