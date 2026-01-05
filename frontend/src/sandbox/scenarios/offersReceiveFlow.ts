/**
 * Offers Receive Flow Scenario
 * Сценарий обучения: выбор предложений на свою заявку
 */

import type { TutorialStep } from './types'
import { mockFreightRequests } from '@/sandbox/mockData/freightRequests'
import { mockOffers } from '@/sandbox/mockData/offers'

export const steps: TutorialStep[] = [
  // === Навигация к заявке ===
  {
    id: 'offers_select_request',
    title: 'Выберите заявку',
    description: 'Нажмите на заявку с предложениями чтобы увидеть детали.',
    route: '/',  // Автоматически перенаправляем на страницу заявок
    target: 'freight-request-card',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/freight-requests/',
    async beforeStep() {
      // Создаём заявку с офферами
      await mockFreightRequests.seedWithOffers('sandbox-fr-offers', 4)
    },
  },
  {
    id: 'offers_view_tab',
    title: 'Вкладка Предложения',
    description: 'Перейдите на вкладку "Предложения" чтобы увидеть поступившие офферы от перевозчиков.',
    target: 'offers-tab',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'tab:offers',
  },
  // === Работа с офферами ===
  {
    id: 'offers_reject',
    title: 'Отклонение',
    description: 'Отклоните неподходящее предложение. Нажмите "Отклонить" на любом оффере.',
    target: 'reject-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'offer:rejected',
  },
  {
    id: 'offers_select',
    title: 'Выбор предложения',
    description: 'Выберите подходящее предложение. Нажмите "Выбрать".',
    target: 'select-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'offer:selected',
  },
  {
    id: 'offers_unselect',
    title: 'Отмена выбора',
    description: 'Пока перевозчик не подтвердил, можно отменить выбор. Нажмите "Отменить выбор".',
    target: 'unselect-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'offer:unselected',
  },
  {
    id: 'offers_select_confirm',
    title: 'Выбор с подтверждением',
    description: 'Теперь выберите другое предложение. Перевозчик автоматически подтвердит — и создастся заказ.',
    target: 'select-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'order:created',
    async beforeStep() {
      // Настраиваем автоподтверждение для следующего оффера
      mockOffers.setAutoConfirm('sandbox-offer-2', true)
      mockOffers.setAutoConfirm('sandbox-offer-3', true)
      mockOffers.setAutoConfirm('sandbox-offer-4', true)
    },
  },
  // === Завершение ===
  {
    id: 'offers_complete',
    title: 'Готово!',
    description: 'Заказ создан. Теперь можно общаться с перевозчиком и отслеживать выполнение заказа.',
    completionType: 'manual',
  },
]

export default { steps }
