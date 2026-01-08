/**
 * Mock Handlers for Members & Organizations
 */

import { registerHandler } from './index'
import { mockMembers } from '@/sandbox/mockData'
import { tutorialBus } from '@/sandbox/events'
import { useAuthStore } from '@/stores/auth'
import type { MemberRole } from '@/types/member'

export function membersHandlers(): void {
  // Get member profile
  registerHandler('GET', '/members/:id', (params) => {
    const auth = useAuthStore()
    const member = mockMembers.get(params.id)

    if (!member) {
      // Если member не найден в mock store — пропускаем к реальному API
      return null
    }

    // Используем реальные данные организации из auth store
    return {
      data: {
        id: member.id,
        name: member.name,
        email: member.email,
        phone: member.phone,
        role: member.role,
        status: member.status,
        organization_id: auth.organizationId,
        organization_name: auth.organization?.name || 'Моя организация',
        created_at: member.created_at,
      },
    }
  })

  // Get organization with members (для listByOrganization)
  registerHandler('GET', '/organizations/:orgId/full', async (params) => {
    const auth = useAuthStore()

    // Сидим members если ещё не засеяны
    await mockMembers.seed(5)

    const members = mockMembers.list()

    return {
      data: {
        id: params.orgId,
        // Используем реальное название организации из auth store
        name: auth.organization?.name || 'Моя организация',
        members: members.map((m) => ({
          id: m.id,
          name: m.name,
          email: m.email,
          phone: m.phone,
          role: m.role,
          status: m.status,
          created_at: m.created_at,
        })),
      },
    }
  })

  // Change member role
  registerHandler('PATCH', '/organizations/:orgId/members/:memberId/role', (params, body) => {
    const { role } = body as { role: MemberRole }

    if (role !== 'administrator' && role !== 'employee') {
      return {
        status: 400,
        data: { error: 'Недопустимая роль', error_code: 'INVALID_ROLE' },
      }
    }

    mockMembers.changeRole(params.memberId, role)

    // Эмитим событие
    tutorialBus.emit('member:roleChanged', {
      memberId: params.memberId,
      newRole: role,
    })

    return { status: 204 }
  })

  // Block member
  registerHandler('POST', '/organizations/:orgId/members/:memberId/block', (params, body) => {
    const { reason } = body as { reason: string }

    mockMembers.block(params.memberId)

    // Эмитим событие
    tutorialBus.emit('member:blocked', { memberId: params.memberId })

    return { status: 204 }
  })

  // Unblock member
  registerHandler('POST', '/organizations/:orgId/members/:memberId/unblock', (params) => {
    mockMembers.unblock(params.memberId)

    // Эмитим событие
    tutorialBus.emit('member:unblocked', { memberId: params.memberId })

    return { status: 204 }
  })

  // Create invitation
  registerHandler('POST', '/organizations/:orgId/invitations', (params, body) => {
    const { email, role } = body as { email: string; role: 'administrator' | 'employee' }

    const result = mockMembers.createInvitation(email, role)

    // Эмитим событие
    tutorialBus.emit('invitation:created', {
      invitationId: result.id,
      email,
    })

    return { data: result }
  })

  // List invitations
  registerHandler('GET', '/organizations/:orgId/invitations', () => {
    const invitations = mockMembers.listInvitations()
    return { data: invitations }
  })

  // ВАЖНО: НЕ перехватываем /auth/me и /auth/logout
  // Эти запросы должны идти к реальному API, чтобы сохранить
  // данные реального пользователя в sandbox режиме
}
