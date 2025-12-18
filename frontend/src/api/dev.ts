import { api } from './client'

export interface DevUser {
  id: string
  organization_id: string
  email: string
  name: string
  role: string
  status: string
  organization_name: string
  organization_status: string
}

export interface DevStatus {
  enabled: boolean
}

export const devApi = {
  getStatus(): Promise<DevStatus> {
    return api.get('/dev/status')
  },

  listUsers(search?: string): Promise<DevUser[]> {
    const params = new URLSearchParams()
    if (search) params.set('search', search)
    const query = params.toString()
    return api.get(`/dev/users${query ? '?' + query : ''}`)
  },

  switchUser(memberId: string): Promise<DevUser> {
    return api.post('/dev/switch', { member_id: memberId })
  },

  deleteUser(memberId: string): Promise<void> {
    return api.delete(`/dev/users/${memberId}`)
  },
}
