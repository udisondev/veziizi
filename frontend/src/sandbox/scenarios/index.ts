/**
 * Scenarios Index
 * Экспорт всех сценариев обучения
 */

import type { Scenario, ScenarioType } from '@/types/tutorial'

// Метаданные сценариев для UI (без шагов - они загружаются лениво)
export const scenarioMeta: Record<ScenarioType, Omit<Scenario, 'steps'>> = {
  customer_flow: {
    id: 'customer_flow',
    name: 'Путь заказчика',
    description: 'Научитесь создавать заявки и работать с перевозчиками',
    icon: 'Package',
  },
  carrier_flow: {
    id: 'carrier_flow',
    name: 'Путь перевозчика',
    description: 'Научитесь находить заявки и делать предложения',
    icon: 'Truck',
  },
  admin_flow: {
    id: 'admin_flow',
    name: 'Управление командой',
    description: 'Научитесь управлять сотрудниками организации',
    icon: 'Users',
    requiredRole: 'administrator',
  },
  subscriptions_flow: {
    id: 'subscriptions_flow',
    name: 'Подписки на заявки',
    description: 'Настройте автоматические уведомления о новых заявках',
    icon: 'Bell',
  },
  telegram_flow: {
    id: 'telegram_flow',
    name: 'Подключение Telegram',
    description: 'Получайте уведомления в Telegram',
    icon: 'MessageCircle',
  },
  offers_receive_flow: {
    id: 'offers_receive_flow',
    name: 'Получение офферов',
    description: 'Научитесь работать с предложениями перевозчиков',
    icon: 'Handshake',
  },
  completion_flow: {
    id: 'completion_flow',
    name: 'Завершение заявки',
    description: 'Научитесь завершать заявки и оставлять отзывы',
    icon: 'CheckCircle',
  },
}

// Основные сценарии для welcome модалки
export const mainScenarios: ScenarioType[] = ['customer_flow', 'carrier_flow', 'admin_flow']

// Дополнительные сценарии
export const additionalScenarios: ScenarioType[] = ['subscriptions_flow', 'telegram_flow']
