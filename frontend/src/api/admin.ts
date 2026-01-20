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
  EmailTemplate,
  EmailTemplatesListResponse,
  EmailTemplateListFilter,
  CreateEmailTemplateRequest,
  UpdateEmailTemplateRequest,
  PreviewEmailTemplateRequest,
  PreviewEmailTemplateResponse,
} from '@/types/admin'

export const adminApi = {
  login(data: AdminLoginRequest): Promise<AdminLoginResponse> {
    return api.post('/admin/auth/login', data)
  },

  logout(): Promise<void> {
    return api.post('/admin/auth/logout')
  },

  me(): Promise<AdminLoginResponse> {
    return api.get('/admin/auth/me')
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

  // Email templates
  async getEmailTemplates(filter?: EmailTemplateListFilter): Promise<EmailTemplatesListResponse> {
    const params = new URLSearchParams()
    if (filter?.category) params.set('category', filter.category)
    if (filter?.is_active !== undefined) params.set('is_active', String(filter.is_active))
    if (filter?.is_system !== undefined) params.set('is_system', String(filter.is_system))
    if (filter?.search) params.set('search', filter.search)
    if (filter?.limit) params.set('limit', String(filter.limit))
    if (filter?.offset) params.set('offset', String(filter.offset))
    const query = params.toString()
    return api.get(`/admin/email-templates${query ? '?' + query : ''}`)
  },

  getEmailTemplate(id: string): Promise<EmailTemplate> {
    return api.get(`/admin/email-templates/${id}`)
  },

  createEmailTemplate(data: CreateEmailTemplateRequest): Promise<EmailTemplate> {
    return api.post('/admin/email-templates', data)
  },

  updateEmailTemplate(id: string, data: UpdateEmailTemplateRequest): Promise<EmailTemplate> {
    return api.patch(`/admin/email-templates/${id}`, data)
  },

  deleteEmailTemplate(id: string): Promise<void> {
    return api.delete(`/admin/email-templates/${id}`)
  },

  previewEmailTemplate(data: PreviewEmailTemplateRequest): Promise<PreviewEmailTemplateResponse> {
    return api.post('/admin/email-templates/preview', data)
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
