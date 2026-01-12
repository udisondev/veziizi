/**
 * Completion Flow Scenario
 * Сценарий обучения: завершение заявки и оставление отзыва
 * Универсальный для обеих сторон (заказчик и перевозчик) — процесс одинаковый
 */

import type { TutorialStep } from './types'
import router from '@/router'
import { mockFreightRequests } from '@/sandbox/mockData/freightRequests'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'

// ID заявки для этого сценария (fallback для автономного режима)
const FR_ID = 'sandbox-fr-completion'

/**
 * Получить ID заявки — из chain context или fallback
 */
function getFreightRequestId(): string {
  const onboarding = useOnboardingStore()
  return onboarding.getChainedFreightRequestId() ?? FR_ID
}

/**
 * Инициализация сценария — создаёт mock данные при любом старте/возобновлении
 */
async function initialize() {
  const auth = useAuthStore()

  // Определяем ID заявки (из chain context или fallback)
  const frId = getFreightRequestId()

  // Всегда создаём заявку в mockFreightRequests
  // (даже в режиме цепочки, на случай если данные потеряны при восстановлении)
  await mockFreightRequests.seedConfirmedRequest(frId, {
    customer_org_id: auth.organizationId!,
    customer_org_name: auth.organization?.name || 'Моя организация',
    customer_member_id: auth.memberId!,
    customer_member_name: auth.name || 'Ответственный',
  })
}

export const steps: TutorialStep[] = [
  // === Введение ===
  {
    id: 'completion_intro',
    title: 'Завершение перевозки',
    description:
      'Когда перевозка выполнена, обе стороны (заказчик и перевозчик) должны подтвердить завершение. Процесс одинаковый для всех участников.',
    completionType: 'manual',
    hint: 'После завершения обеими сторонами заявка переходит в статус "Завершена"',
  },

  // === Завершение ===
  {
    id: 'completion_button',
    title: 'Завершить заявку',
    description: 'Нажмите кнопку "Завершить заявку" в правом верхнем углу.',
    target: 'complete-request-btn',
    tooltipPosition: 'bottom',
    completionType: 'action',
    completionAction: 'completion:confirmOpened',
    // Навигация на страницу заявки (динамический ID)
    async beforeStep() {
      const frId = getFreightRequestId()
      const currentPath = router.currentRoute.value.path
      if (!currentPath.includes(frId)) {
        await router.push(`/freight-requests/${frId}`)
        // Даём время на рендеринг страницы
        await new Promise(resolve => setTimeout(resolve, 200))
      }
    },
  },
  {
    id: 'completion_confirm',
    title: 'Подтверждение',
    description: 'Подтвердите завершение перевозки. После этого откроется окно для отзыва.',
    target: 'complete-confirm-modal',
    hideTooltip: true,
    completionType: 'action',
    completionAction: 'completion:completed',
  },

  // === Завершение курса (ждём закрытия модального окна отзыва) ===
  {
    id: 'completion_complete',
    title: 'Оставьте отзыв',
    description: 'Оставьте отзыв или пропустите — отзыв можно изменить в течение 24 часов.',
    hideTooltip: true,
    completionType: 'action',
    completionAction: 'review:closed',
  },
]

export default { steps, initialize }
