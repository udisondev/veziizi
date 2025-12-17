import { api } from './client'
import type {
  CreateInvitationRequest,
  CreateInvitationResponse,
  AcceptInvitationRequest,
  AcceptInvitationResponse,
  InvitationDetails,
  InvitationListResponse,
} from '@/types/invitation'

export const invitationsApi = {
  // Создать приглашение (требует авторизации)
  create(orgId: string, data: CreateInvitationRequest): Promise<CreateInvitationResponse> {
    return api.post(`/organizations/${orgId}/invitations`, data)
  },

  // Получить список приглашений организации (требует авторизации)
  async list(orgId: string, status?: string): Promise<InvitationListResponse> {
    const query = status ? `?status=${status}` : ''
    const result = await api.get<InvitationListResponse | null>(`/organizations/${orgId}/invitations${query}`)
    return result ?? { items: [] }
  },

  // Получить данные приглашения по токену (публичный)
  getByToken(token: string): Promise<InvitationDetails> {
    return api.get(`/invitations/${token}`)
  },

  // Принять приглашение (публичный)
  accept(token: string, data: AcceptInvitationRequest): Promise<AcceptInvitationResponse> {
    return api.post(`/invitations/${token}/accept`, data)
  },

  // Отменить приглашение (требует авторизации)
  cancel(orgId: string, invitationId: string): Promise<void> {
    return api.delete(`/organizations/${orgId}/invitations/${invitationId}`)
  },
}
