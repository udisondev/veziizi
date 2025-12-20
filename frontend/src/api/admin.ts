import { api } from './client'
import type {
  AdminLoginRequest,
  AdminLoginResponse,
  PendingOrganization,
  OrganizationDetail,
  RejectRequest,
  PendingReviewsResponse,
  PendingReview,
  ApproveReviewRequest,
  RejectReviewRequest,
  FraudstersResponse,
  MarkFraudsterRequest,
  UnmarkFraudsterRequest,
} from '@/types/admin'

export const adminApi = {
  login(data: AdminLoginRequest): Promise<AdminLoginResponse> {
    return api.post('/admin/auth/login', data)
  },

  logout(): Promise<void> {
    return api.post('/admin/auth/logout')
  },

  async getOrganizations(): Promise<PendingOrganization[]> {
    const result = await api.get<PendingOrganization[] | null>('/admin/organizations')
    return result ?? []
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

  // Reviews moderation
  async getPendingReviews(limit = 20, offset = 0): Promise<PendingReviewsResponse> {
    return api.get(`/admin/reviews?limit=${limit}&offset=${offset}`)
  },

  getReview(id: string): Promise<PendingReview> {
    return api.get(`/admin/reviews/${id}`)
  },

  approveReview(id: string, data: ApproveReviewRequest): Promise<void> {
    return api.post(`/admin/reviews/${id}/approve`, data)
  },

  rejectReview(id: string, data: RejectReviewRequest): Promise<void> {
    return api.post(`/admin/reviews/${id}/reject`, data)
  },

  // Fraudsters management
  async getFraudsters(limit = 20, offset = 0): Promise<FraudstersResponse> {
    return api.get(`/admin/fraudsters?limit=${limit}&offset=${offset}`)
  },

  markFraudster(orgId: string, data: MarkFraudsterRequest): Promise<void> {
    return api.post(`/admin/organizations/${orgId}/mark-fraudster`, data)
  },

  unmarkFraudster(orgId: string, data: UnmarkFraudsterRequest): Promise<void> {
    return api.post(`/admin/organizations/${orgId}/unmark-fraudster`, data)
  },
}
