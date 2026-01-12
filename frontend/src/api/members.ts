import { api } from './client'
import type { MemberListItem, MemberRole, MemberStatus } from '@/types/member'

export interface MemberProfile {
  id: string
  name: string
  email: string
  phone?: string
  role: MemberRole
  status: MemberStatus
  organization_id: string
  organization_name: string
  created_at: string
}

export interface OrganizationWithMembers {
  id: string
  name: string
  members: MemberListItem[]
}

export const membersApi = {
  getProfile(id: string): Promise<MemberProfile> {
    return api.get(`/members/${id}`)
  },

  async listByOrganization(orgId: string): Promise<MemberListItem[]> {
    // SEC-019: используем /full endpoint для получения списка членов (только своя организация)
    const org = await api.get<OrganizationWithMembers>(`/organizations/${orgId}/full`)
    return org.members ?? []
  },

  changeRole(orgId: string, memberId: string, role: MemberRole): Promise<void> {
    return api.patch(`/organizations/${orgId}/members/${memberId}/role`, { role })
  },

  block(orgId: string, memberId: string, reason: string): Promise<void> {
    return api.post(`/organizations/${orgId}/members/${memberId}/block`, { reason })
  },

  unblock(orgId: string, memberId: string): Promise<void> {
    return api.post(`/organizations/${orgId}/members/${memberId}/unblock`)
  },

  updateInfo(orgId: string, memberId: string, name: string, email: string, phone: string): Promise<void> {
    return api.patch(`/organizations/${orgId}/members/${memberId}/info`, { name, email, phone })
  },
}
