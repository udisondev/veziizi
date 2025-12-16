import { api } from './client'

export interface MemberProfile {
  id: string
  name: string
  email: string
  phone?: string
  organization_id: string
  organization_name: string
  created_at: string
}

export const membersApi = {
  getProfile(id: string): Promise<MemberProfile> {
    return api.get(`/members/${id}`)
  },
}
