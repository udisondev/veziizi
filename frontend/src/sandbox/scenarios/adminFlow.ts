/**
 * Admin Flow Scenario
 * Сценарий обучения для администратора/владельца (управление командой)
 */

import type { TutorialStep } from './types'
import { mockMembers } from '@/sandbox/mockData/members'

export const steps: TutorialStep[] = [
  {
    id: 'admin_start',
    title: 'Управление командой',
    description: 'Научимся управлять сотрудниками организации. Перейдём в раздел "Сотрудники".',
    route: '/members',
    highlightSelector: '[data-tutorial="members-link"]',
    tooltipPosition: 'right',
    completionType: 'navigate',
    completionAction: '/members',
    async beforeStep() {
      // Добавляем mock сотрудников
      await mockMembers.seed(5)
    },
  },
  {
    id: 'admin_invite_member',
    title: 'Приглашение сотрудника',
    description: 'Пригласите нового члена команды. Нажмите "Пригласить".',
    highlightSelector: '[data-tutorial="invite-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'invitation:created',
  },
  {
    id: 'admin_change_role',
    title: 'Изменение роли',
    description: 'Измените роль сотрудника. Нажмите на меню действий.',
    highlightSelector: '[data-tutorial="member-actions"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'member:roleChanged',
  },
  {
    id: 'admin_block_member',
    title: 'Блокировка',
    description: 'Заблокируйте сотрудника для ограничения доступа.',
    highlightSelector: '[data-tutorial="block-member-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'member:blocked',
  },
  {
    id: 'admin_unblock_member',
    title: 'Разблокировка',
    description: 'Разблокируйте сотрудника.',
    highlightSelector: '[data-tutorial="unblock-member-btn"]',
    tooltipPosition: 'left',
    completionType: 'action',
    completionAction: 'member:unblocked',
  },
  {
    id: 'admin_complete',
    title: 'Готово!',
    description: 'Вы научились управлять командой. Теперь можете добавлять, изменять роли и блокировать сотрудников.',
    completionType: 'manual',
  },
]

export default { steps }
