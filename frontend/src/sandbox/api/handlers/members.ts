/**
 * Mock Handlers for Members & Organizations
 */

import { registerHandler } from './index'
import { mockMembers } from '@/sandbox/mockData'
import { tutorialBus } from '@/sandbox/events'
import type { MemberRole } from '@/types/member'

export function membersHandlers(): void {
  // Get member profile
  registerHandler('GET', '/members/:id', (params) => {
    const member = mockMembers.get(params.id)
    if (!member) {
      // Для sandbox возвращаем mock профиль текущего пользователя
      if (params.id === 'sandbox-member-1') {
        return {
          data: {
            id: 'sandbox-member-1',
            name: 'Я (Sandbox)',
            email: 'me@sandbox.local',
            phone: '+7 (999) 123-45-67',
            role: 'owner',
            status: 'active',
            organization_id: 'sandbox-org-1',
            organization_name: 'Моя организация (Sandbox)',
            created_at: new Date().toISOString(),
          },
        }
      }
      return {
        status: 404,
        data: { error: 'Сотрудник не найден', error_code: 'NOT_FOUND' },
      }
    }
    return {
      data: {
        id: member.id,
        name: member.name,
        email: member.email,
        phone: member.phone,
        role: member.role,
        status: member.status,
        organization_id: 'sandbox-org-1',
        organization_name: 'Моя организация (Sandbox)',
        created_at: member.created_at,
      },
    }
  })

  // Get organization with members (для listByOrganization)
  registerHandler('GET', '/organizations/:orgId/full', async (params) => {
    // Сидим members если ещё не засеяны
    await mockMembers.seed(5)

    const members = mockMembers.list()

    return {
      data: {
        id: params.orgId,
        name: 'Моя организация (Sandbox)',
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

  // Get current user session (auth/me)
  registerHandler('GET', '/auth/me', () => {
    return {
      data: {
        member_id: 'sandbox-member-1',
        name: 'Я (Sandbox)',
        email: 'me@sandbox.local',
        organization_id: 'sandbox-org-1',
        role: 'owner',
        organization: {
          name: 'Моя организация (Sandbox)',
          status: 'active',
        },
      },
    }
  })

  // Logout (just return success in sandbox)
  registerHandler('POST', '/auth/logout', () => {
    return { status: 204 }
  })
}
