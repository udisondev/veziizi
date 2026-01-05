/**
 * Orders Flow Scenario
 * Сценарий обучения работе с заказами
 */

import type { TutorialStep } from './types'
import { mockOrders } from '@/sandbox/mockData/orders'
import { mockBot } from '@/sandbox/mockData/bot'

export const steps: TutorialStep[] = [
  // === Начало: переход к заказам ===
  {
    id: 'orders_start',
    title: 'Управление заказами',
    description: 'Научимся работать с активными заказами. Перейдите в раздел "Заказы".',
    route: '/',
    highlightSelector: '[data-tutorial="orders-link"]',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/orders',
    async beforeStep() {
      // Создаём mock заказ для tutorial
      mockOrders.seed(1)
    },
  },

  // === Выбор заказа ===
  {
    id: 'orders_select',
    title: 'Выберите заказ',
    description: 'Нажмите на карточку заказа, чтобы открыть детали.',
    highlightSelector: '[data-tutorial="order-card"]',
    tooltipPosition: 'right',
    completionType: 'navigate',
    completionAction: '/orders/sandbox-order-1',
  },

  // === Вкладка сообщений ===
  {
    id: 'orders_messages_tab',
    title: 'Переписка с контрагентом',
    description: 'Перейдите на вкладку "Сообщения" для связи с перевозчиком.',
    highlightSelector: '[data-tutorial="messages-tab"]',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'tab:messages',
  },

  // === Отправка сообщения ===
  {
    id: 'orders_send_message',
    title: 'Отправьте сообщение',
    description: 'Напишите сообщение перевозчику и отправьте его.',
    highlightSelector: '[data-tutorial="message-input"]',
    tooltipPosition: 'top',
    completionType: 'action',
    completionAction: 'message:sent',
    async afterStep() {
      // Бот ответит через 1.5 секунды
      await mockBot.scheduleReply('sandbox-order-1', 1500)
    },
  },

  // === Вкладка документов ===
  {
    id: 'orders_documents_tab',
    title: 'Документы заказа',
    description: 'Перейдите на вкладку "Документы" для работы с файлами.',
    highlightSelector: '[data-tutorial="documents-tab"]',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'tab:documents',
  },

  // === Загрузка документа ===
  {
    id: 'orders_upload_doc',
    title: 'Загрузка документа',
    description: 'Здесь можно загружать документы: накладные, акты, фото груза. Можете пропустить этот шаг.',
    highlightSelector: '[data-tutorial="upload-doc-btn"]',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'document:uploaded',
    skippable: true,
  },

  // === Завершение ===
  {
    id: 'orders_complete',
    title: 'Готово!',
    description: 'Вы освоили основы работы с заказами: переписку и документы. Когда перевозка завершена, вы сможете завершить заказ и оставить отзыв.',
    completionType: 'manual',
  },
]

export default { steps }
