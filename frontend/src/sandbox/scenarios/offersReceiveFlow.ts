/**
 * Offers Receive Flow Scenario (Updated)
 * Сценарий обучения: выбор предложений на свою заявку через уведомления
 */

import type { TutorialStep } from './types'
import { mockFreightRequests } from '@/sandbox/mockData/freightRequests'
import { mockOffers } from '@/sandbox/mockData/offers'
import { mockNotifications } from '@/sandbox/mockData/notifications'
import { useNotificationsStore } from '@/stores/notifications'
import { useAuthStore } from '@/stores/auth'

// ID заявки для этого сценария
const FR_ID = 'sandbox-fr-offers'

/**
 * Инициализация сценария — создаёт mock данные при любом старте/возобновлении
 * Эта функция вызывается ВСЕГДА, даже если сценарий возобновляется с середины
 */
async function initialize() {
  const auth = useAuthStore()

  // Очищаем предыдущие данные
  mockNotifications.clear()

  // Создаём заявку с офферами и уведомлениями
  // Используем текущую организацию пользователя как владельца заявки
  await mockFreightRequests.seedWithOffers(FR_ID, 4, {
    customer_org_id: auth.organizationId!,
    customer_org_name: auth.organizationName || 'Моя организация',
    customer_member_id: auth.memberId!,
  })

  // Принудительно обновить Pinia store чтобы badge появился сразу
  const notificationsStore = useNotificationsStore()
  await Promise.all([
    notificationsStore.fetchUnreadCount(),
    notificationsStore.fetchRecentNotifications()
  ])
}

export const steps: TutorialStep[] = [
  // === Уведомления ===
  {
    id: 'notifications_intro',
    title: 'Новые предложения!',
    description:
      'Перевозчики откликнулись на вашу заявку. Нажмите на колокольчик чтобы посмотреть уведомления.',
    target: 'notification-bell',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'notification:bellOpened',
    hint: 'Красный индикатор показывает количество непрочитанных уведомлений',
    // beforeStep убран — логика перенесена в initialize()
  },
  {
    id: 'notifications_click',
    title: 'Перейдите к заявке',
    description: 'Нажмите на любое уведомление о предложении чтобы перейти к заявке.',
    // Без target — tooltip не перекрывает уведомления
    completionType: 'navigate',
    completionAction: `/freight-requests/${FR_ID}`,
  },

  // === Вкладка предложений ===
  {
    id: 'offers_view_tab',
    title: 'Вкладка Предложения',
    description: 'Откройте выпадающий список и выберите "Предложения" чтобы увидеть все офферы.',
    target: 'tabs-dropdown',
    tooltipPosition: 'right', // Справа, чтобы не перекрывать dropdown
    completionType: 'action',
    completionAction: 'tab:offers',
  },

  // === Работа с офферами ===
  {
    id: 'offers_reject_click',
    title: 'Отклонение',
    description: 'Отклоните неподходящее предложение. Нажмите "Отклонить" на любом оффере.',
    target: 'reject-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'rejectModal:opened',
  },
  {
    id: 'offers_reject_confirm',
    title: 'Подтверждение отклонения',
    description: 'Можно указать причину отклонения (опционально). Нажмите "Отклонить" для подтверждения или "Отменить" чтобы вернуться.',
    target: 'reject-offer-modal',
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
    description:
      'Теперь выберите другое предложение. Перевозчик автоматически подтвердит.',
    target: 'select-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'offer:confirmed',
    async beforeStep() {
      // Настраиваем автоподтверждение для всех оставшихся pending офферов
      const offers = mockOffers.listByFreightRequest(FR_ID)
      offers
        .filter(o => o.status === 'pending')
        .forEach(offer => mockOffers.setAutoConfirm(offer.id, true))
    },
  },

  // === Завершение ===
  {
    id: 'offers_complete',
    title: 'Готово!',
    description:
      'Перевозчик подтверждён. Перевозка готова к выполнению.',
    completionType: 'manual',
  },
]

export default { steps, initialize }
