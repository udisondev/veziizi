/**
 * Carrier Flow Scenario
 * Сценарий обучения для перевозчика (поиск заявок, предложения)
 */

import type { TutorialStep } from './types'
import { mockFreightRequests } from '@/sandbox/mockData/freightRequests'
import { mockOffers } from '@/sandbox/mockData/offers'

export const steps: TutorialStep[] = [
  // === Поиск заявок ===
  {
    id: 'carrier_start',
    title: 'Добро пожаловать!',
    description: 'Научимся находить заявки и делать предложения. Откроем фильтры для поиска.',
    route: '/',
    highlightSelector: '[data-tutorial="filters-btn"]',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'filters:applied',
    async beforeStep() {
      // Заполняем mock заявками
      await mockFreightRequests.seed(10)
    },
    skippable: true,
  },
  {
    id: 'carrier_view_request',
    title: 'Просмотр заявки',
    description: 'Нажмите на заявку чтобы посмотреть детали.',
    highlightSelector: '[data-tutorial="freight-request-card"]',
    tooltipPosition: 'right',
    completionType: 'navigate',
    completionAction: '/freight-requests/',
  },

  // === Создание предложения ===
  {
    id: 'carrier_make_offer',
    title: 'Предложение',
    description: 'Сделайте предложение заказчику. Укажите вашу цену.',
    highlightSelector: '[data-tutorial="make-offer-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'offer:created',
  },
  {
    id: 'carrier_view_my_offers',
    title: 'Мои предложения',
    description: 'Перейдите в раздел "Предложения" чтобы отслеживать статусы.',
    highlightSelector: '[data-tutorial="my-offers-link"]',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/my-offers',
  },

  // === Симуляция отказа ===
  {
    id: 'carrier_receive_rejection',
    title: 'Отказ заказчика',
    description: 'Заказчик отклонил ваше предложение. Это нормально, попробуйте другие заявки.',
    completionType: 'manual',
    simulationDelay: 2000,
    async beforeStep() {
      await mockOffers.simulateRejection('sandbox-my-offer-1')
    },
  },

  // === Отзыв предложения ===
  {
    id: 'carrier_withdraw_offer',
    title: 'Отзыв предложения',
    description: 'Вы можете отозвать своё предложение, пока оно не выбрано. Нажмите "Отозвать".',
    highlightSelector: '[data-tutorial="withdraw-offer-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'offer:withdrawn',
  },

  // === Симуляция выбора ===
  {
    id: 'carrier_receive_selection',
    title: 'Вас выбрали!',
    description: 'Заказчик выбрал ваше предложение. Подтвердите готовность.',
    completionType: 'manual',
    simulationDelay: 2000,
    async beforeStep() {
      await mockOffers.simulateSelection('sandbox-my-offer-2')
    },
  },
  {
    id: 'carrier_confirm_offer',
    title: 'Подтверждение',
    description: 'Нажмите "Подтвердить" чтобы начать выполнение перевозки.',
    highlightSelector: '[data-tutorial="confirm-offer-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'offer:confirmed',
  },
]

export default { steps }
