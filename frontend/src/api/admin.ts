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

  // Support tickets
  async getSupportTickets(params?: {
    status?: string
    limit?: number
    offset?: number
  }): Promise<{ tickets: AdminSupportTicket[]; total: number }> {
    const queryParams = new URLSearchParams()
    if (params?.status) queryParams.set('status', params.status)
    if (params?.limit) queryParams.set('limit', params.limit.toString())
    if (params?.offset) queryParams.set('offset', params.offset.toString())
    const query = queryParams.toString()
    return api.get(`/admin/support/tickets${query ? '?' + query : ''}`)
  },

  getSupportTicket(id: string): Promise<AdminSupportTicketDetail> {
    return api.get(`/admin/support/tickets/${id}`)
  },

  addSupportMessage(ticketId: string, content: string): Promise<void> {
    return api.post(`/admin/support/tickets/${ticketId}/messages`, { content })
  },

  closeSupportTicket(ticketId: string, resolution?: string): Promise<void> {
    return api.post(`/admin/support/tickets/${ticketId}/close`, { resolution })
  },
}

// Support ticket types
export interface AdminSupportTicket {
  id: string
  ticket_number: number
  member_id: string
  org_id: string
  subject: string
  status: string
  created_at: string
  updated_at: string
}

export interface AdminSupportTicketMessage {
  id: string
  sender_type: 'user' | 'admin'
  sender_id: string
  content: string
  created_at: string
}

export interface AdminSupportTicketDetail {
  id: string
  ticket_number: number
  member_id: string
  org_id: string
  subject: string
  status: string
  messages: AdminSupportTicketMessage[]
  created_at: string
  updated_at: string
  closed_at?: string
}
