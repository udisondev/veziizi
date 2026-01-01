import { api } from './client'

// Types
export interface FAQItem {
  question: string
  answer: string
  category: string
}

export interface CreateTicketRequest {
  subject: string
  message: string
}

export interface CreateTicketResponse {
  id: string
}

export interface TicketListItem {
  id: string
  ticket_number: number
  subject: string
  status: string
  created_at: string
  updated_at: string
}

export interface TicketMessage {
  id: string
  sender_type: 'user' | 'admin'
  sender_id: string
  content: string
  created_at: string
}

export interface TicketDetail {
  id: string
  ticket_number: number
  subject: string
  status: string
  messages: TicketMessage[]
  created_at: string
  updated_at: string
  closed_at?: string
}

export interface AddMessageRequest {
  content: string
}

// API methods
export async function getFAQ(): Promise<FAQItem[]> {
  return api.get<FAQItem[]>('/support/faq')
}

export async function createTicket(data: CreateTicketRequest): Promise<CreateTicketResponse> {
  return api.post<CreateTicketResponse>('/support/tickets', data)
}

export async function getMyTickets(params?: {
  status?: string
  limit?: number
  offset?: number
}): Promise<TicketListItem[]> {
  const query = new URLSearchParams()
  if (params?.status) query.set('status', params.status)
  if (params?.limit) query.set('limit', params.limit.toString())
  if (params?.offset) query.set('offset', params.offset.toString())
  const queryStr = query.toString()
  return api.get<TicketListItem[]>(`/support/tickets${queryStr ? `?${queryStr}` : ''}`)
}

export async function getTicket(id: string): Promise<TicketDetail> {
  return api.get<TicketDetail>(`/support/tickets/${id}`)
}

export async function addMessage(ticketId: string, data: AddMessageRequest): Promise<void> {
  await api.post(`/support/tickets/${ticketId}/messages`, data)
}

export async function reopenTicket(ticketId: string): Promise<void> {
  await api.post(`/support/tickets/${ticketId}/reopen`)
}
