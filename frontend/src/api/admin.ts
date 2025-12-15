import { api } from './client'
import type {
  AdminLoginRequest,
  AdminLoginResponse,
  PendingOrganization,
  OrganizationDetail,
  RejectRequest,
} from '@/types/admin'

export const adminApi = {
  login(data: AdminLoginRequest): Promise<AdminLoginResponse> {
    return api.post('/admin/auth/login', data)
  },

  logout(): Promise<void> {
    return api.post('/admin/auth/logout')
  },

  getOrganizations(): Promise<PendingOrganization[]> {
    return api.get('/admin/organizations')
  },

  getOrganization(id: string): Promise<OrganizationDetail> {
    return api.get(`/admin/organizations/${id}`)
  },

  approveOrganization(id: string): Promise<void> {
    return api.post(`/admin/organizations/${id}/approve`)
  },

  rejectOrganization(id: string, data: RejectRequest): Promise<void> {
    return api.post(`/admin/organizations/${id}/reject`, data)
  },
}
