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
import { useOnboardingStore } from '@/stores/onboarding'

// ID заявки для этого сценария (fallback для автономного режима)
const FR_ID = 'sandbox-fr-offers'

/**
 * Получить ID заявки — из chain context или fallback
 */
function getFreightRequestId(): string {
  const onboarding = useOnboardingStore()
  return onboarding.getChainedFreightRequestId() ?? FR_ID
}

/**
 * Инициализация сценария — создаёт mock данные при любом старте/возобновлении
 * Эта функция вызывается ВСЕГДА, даже если сценарий возобновляется с середины
 */
async function initialize() {
  const auth = useAuthStore()
  const onboarding = useOnboardingStore()

  // Очищаем предыдущие уведомления в любом случае
  mockNotifications.clear()

  // Определяем ID заявки (из chain context или fallback)
  const frId = getFreightRequestId()

  // В обоих режимах создаём заявку в mockFreightRequests с офферами
  // (в режиме цепочки заявка из customerFlow хранится только в sandboxCreatedRequest,
  // но не в mockFreightRequests — поэтому нужно её создать)
  await mockFreightRequests.seedWithOffers(frId, 4, {
    customer_org_id: auth.organizationId!,
    customer_org_name: auth.organization?.name || 'Моя организация',
    customer_member_id: auth.memberId!,
    customer_member_name: auth.name || 'Ответственный',
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
    // Partial match — любой путь начинающийся с /freight-requests/
    completionAction: '/freight-requests/',
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
    completionAction: 'selectModal:opened',
  },
  {
    id: 'offers_select_confirm_modal',
    title: 'Подтверждение выбора',
    description: 'Подтвердите выбор предложения в открывшемся окне.',
    completionType: 'action',
    completionAction: 'offer:selected',
    hideTooltip: true,
  },
  {
    id: 'offers_unselect',
    title: 'Отмена выбора',
    description: 'Пока перевозчик не подтвердил, можно отменить выбор. Нажмите "Отменить выбор".',
    target: 'unselect-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'unselectModal:opened',
  },
  {
    id: 'offers_unselect_confirm_modal',
    title: 'Подтверждение отмены',
    description: 'Подтвердите отмену выбора в открывшемся окне.',
    completionType: 'action',
    completionAction: 'offer:unselected',
    hideTooltip: true,
  },
  {
    id: 'offers_select_confirm',
    title: 'Выбор с подтверждением',
    description:
      'Теперь выберите другое предложение. Перевозчик автоматически подтвердит.',
    target: 'select-offer-btn',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'selectModal:opened',
    async beforeStep() {
      // Настраиваем автоподтверждение для всех оставшихся pending офферов
      const frId = getFreightRequestId()
      const offers = mockOffers.listByFreightRequest(frId)
      offers
        .filter(o => o.status === 'pending')
        .forEach(offer => mockOffers.setAutoConfirm(offer.id, true))
    },
  },
  {
    id: 'offers_select_confirm_modal_final',
    title: 'Подтверждение выбора',
    description: 'Подтвердите выбор. После этого перевозчик автоматически подтвердит участие.',
    completionType: 'action',
    completionAction: 'offer:confirmed',
    hideTooltip: true,
  },
  {
    id: 'offers_switch_to_details',
    title: 'Детали заявки',
    description: 'Теперь откройте вкладку "Детали заявки" чтобы увидеть информацию о перевозчике.',
    target: 'tabs-dropdown',
    tooltipPosition: 'right',
    completionType: 'action',
    completionAction: 'tab:details',
  },
  {
    id: 'offers_carrier_info',
    title: 'Информация о перевозчике',
    description: 'После подтверждения здесь отображается информация о перевозчике: организация и ответственный сотрудник.',
    target: 'carrier-info',
    tooltipPosition: 'bottom',
    completionType: 'manual',
  },
  {
    id: 'offers_visit_carrier',
    title: 'Профиль перевозчика',
    description: 'Нажмите на название организации перевозчика, чтобы посмотреть его профиль и рейтинг.',
    target: 'carrier-org-link',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/organizations/carrier-',
  },
  {
    id: 'offers_back_to_request',
    title: 'Вернитесь к заявке',
    description: 'Нажмите кнопку "Назад" чтобы вернуться к заявке.',
    target: 'org-back-button',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    // Partial match — любой путь начинающийся с /freight-requests/
    completionAction: '/freight-requests/',
    hint: 'Кнопка "Назад" вернёт вас на предыдущую страницу',
  },
  {
    id: 'offers_visit_member',
    title: 'Профиль ответственного',
    description: 'Нажмите на ФИО ответственного перевозчика, чтобы посмотреть его профиль.',
    target: 'carrier-member-link',
    tooltipPosition: 'bottom',
    completionType: 'navigate',
    completionAction: '/members/carrier-',
  },

  // === Завершение ===
  {
    id: 'offers_complete',
    title: 'Обучение завершено!',
    description:
      'Вы научились работать с предложениями. Хотите пройти обучение по завершению заявки и отзывам?',
    completionType: 'manual',
    showCompletionTrainingButton: true,
  },
]

export default { steps, initialize }
