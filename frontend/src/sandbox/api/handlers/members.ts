/**
 * Mock Handlers for Members & Organizations
 */

import { registerHandler } from './index'
import { mockMembers } from '@/sandbox/mockData'
import { tutorialBus } from '@/sandbox/events'
import { useAuthStore } from '@/stores/auth'
import type { MemberRole } from '@/types/member'

// Mock организации-перевозчики (для поиска members)
const CARRIER_ORGANIZATIONS = {
  'carrier-1': {
    id: 'carrier-1',
    name: 'ТрансЛогистик',
    members: [
      {
        id: 'carrier-1-member',
        email: 'petrov@translogistic.ru',
        name: 'Иван Петров',
        phone: '+7 (495) 123-45-68',
        role: 'owner',
        status: 'active',
        created_at: '2024-01-15T10:00:00Z',
      },
    ],
  },
  'carrier-2': {
    id: 'carrier-2',
    name: 'СпецГруз',
    members: [
      {
        id: 'carrier-2-member',
        email: 'smirnov@specgruz.ru',
        name: 'Алексей Смирнов',
        phone: '+7 (495) 234-56-79',
        role: 'owner',
        status: 'active',
        created_at: '2024-02-10T12:00:00Z',
      },
    ],
  },
  'carrier-3': {
    id: 'carrier-3',
    name: 'МегаФура',
    members: [
      {
        id: 'carrier-3-member',
        email: 'kozlov@megafura.ru',
        name: 'Дмитрий Козлов',
        phone: '+7 (495) 345-67-90',
        role: 'owner',
        status: 'active',
        created_at: '2024-03-05T14:00:00Z',
      },
    ],
  },
} as const

// Хелпер для поиска member в mock организациях-перевозчиках
function findMemberInCarrierOrgs(memberId: string) {
  for (const org of Object.values(CARRIER_ORGANIZATIONS)) {
    const member = org.members.find(m => m.id === memberId)
    if (member) {
      return { member, organization: org }
    }
  }
  return null
}

export function membersHandlers(): void {
  // Get member profile
  registerHandler('GET', '/members/:id', (params) => {
    const auth = useAuthStore()

    // Сначала ищем в mock store текущей организации
    const member = mockMembers.get(params.id)
    if (member) {
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
    }

    // Затем ищем в mock организациях-перевозчиках (для tutorial)
    const carrierResult = findMemberInCarrierOrgs(params.id)
    if (carrierResult) {
      const { member: carrierMember, organization } = carrierResult
      return {
        data: {
          id: carrierMember.id,
          name: carrierMember.name,
          email: carrierMember.email,
          phone: carrierMember.phone,
          role: carrierMember.role,
          status: carrierMember.status,
          organization_id: organization.id,
          organization_name: organization.name,
          created_at: carrierMember.created_at,
        },
      }
    }

    // Если member не найден нигде — пропускаем к реальному API
    return null
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

  // Update member info (partial update - nil fields are not changed)
  registerHandler('PATCH', '/organizations/:orgId/members/:memberId/info', (params, body) => {
    const { name, email, phone } = body as { name?: string; email?: string; phone?: string }

    // Валидация: если поле передано, оно не должно быть пустым
    if (name !== undefined && name.trim() === '') {
      return {
        status: 400,
        data: { error: 'name cannot be empty' },
      }
    }
    if (email !== undefined && email.trim() === '') {
      return {
        status: 400,
        data: { error: 'email cannot be empty' },
      }
    }
    if (phone !== undefined && phone.trim() === '') {
      return {
        status: 400,
        data: { error: 'phone cannot be empty' },
      }
    }

    mockMembers.updateInfo(params.memberId, name, email, phone)

    // Эмитим событие
    tutorialBus.emit('member:infoUpdated', { memberId: params.memberId, name, email, phone })

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
