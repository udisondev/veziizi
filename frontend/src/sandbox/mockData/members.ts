/**
 * Mock Members Store
 * Mock данные для сотрудников организации
 */

import { generateId, randomItem, CONTACT_NAMES, PHONE_NUMBERS } from './generators'

interface MockMember {
  id: string
  email: string
  name: string
  phone?: string
  role: 'owner' | 'administrator' | 'employee'
  status: 'active' | 'blocked'
  created_at: string
}

interface MockInvitation {
  id: string
  email: string
  role: 'administrator' | 'employee'
  status: 'pending' | 'accepted' | 'expired'
  created_at: string
  expires_at: string
}

const EMAILS = [
  'ivan@example.com',
  'alexey@example.com',
  'dmitry@example.com',
  'maria@example.com',
  'elena@example.com',
]

class MockMembersStore {
  private members: Map<string, MockMember> = new Map()
  private invitations: Map<string, MockInvitation> = new Map()
  private seeded = false

  /**
   * Заполнить mock данными
   */
  async seed(count: number = 5): Promise<void> {
    if (this.seeded) return

    // Добавляем владельца
    const owner: MockMember = {
      id: 'sandbox-member-owner',
      email: 'owner@sandbox.local',
      name: 'Владелец (Sandbox)',
      phone: '+7 (999) 000-00-00',
      role: 'owner',
      status: 'active',
      created_at: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
    }
    this.members.set(owner.id, owner)

    // Добавляем сотрудников
    for (let i = 0; i < count - 1; i++) {
      const member: MockMember = {
        id: generateId('member'),
        email: EMAILS[i % EMAILS.length],
        name: CONTACT_NAMES[i % CONTACT_NAMES.length],
        phone: PHONE_NUMBERS[i % PHONE_NUMBERS.length],
        role: i === 0 ? 'administrator' : 'employee',
        status: 'active',
        created_at: new Date(Date.now() - (i + 1) * 30 * 24 * 60 * 60 * 1000).toISOString(),
      }
      this.members.set(member.id, member)
    }

    this.seeded = true
  }

  /**
   * Получить список сотрудников
   */
  list(): MockMember[] {
    return Array.from(this.members.values())
  }

  /**
   * Получить список приглашений
   */
  listInvitations(): MockInvitation[] {
    return Array.from(this.invitations.values())
  }

  /**
   * Получить сотрудника по ID
   */
  get(id: string): MockMember | null {
    return this.members.get(id) || null
  }

  /**
   * Создать приглашение
   */
  createInvitation(email: string, role: 'administrator' | 'employee'): { id: string; token: string } {
    const id = generateId('invitation')
    const token = generateId('token')

    const invitation: MockInvitation = {
      id,
      email,
      role,
      status: 'pending',
      created_at: new Date().toISOString(),
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    }

    this.invitations.set(id, invitation)
    return { id, token }
  }

  /**
   * Изменить роль
   */
  changeRole(memberId: string, newRole: 'administrator' | 'employee'): void {
    const member = this.members.get(memberId)
    if (member && member.role !== 'owner') {
      member.role = newRole
    }
  }

  /**
   * Заблокировать
   */
  block(memberId: string): void {
    const member = this.members.get(memberId)
    if (member && member.role !== 'owner') {
      member.status = 'blocked'
    }
  }

  /**
   * Разблокировать
   */
  unblock(memberId: string): void {
    const member = this.members.get(memberId)
    if (member) {
      member.status = 'active'
    }
  }

  /**
   * Очистить store
   */
  clear(): void {
    this.members.clear()
    this.invitations.clear()
    this.seeded = false
  }
}

export const mockMembers = new MockMembersStore()
