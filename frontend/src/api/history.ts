import { api } from './client'

export interface Actor {
  id: string
  name: string
  email: string
}

export interface EventHistoryItem {
  id: string
  event_type: string
  version: number
  occurred_at: string
  actor?: Actor
  data: Record<string, unknown>
}

export interface EventHistoryPage {
  items: EventHistoryItem[]
  total: number
}

export interface HistoryParams {
  limit?: number
  offset?: number
}

function buildQuery(params?: HistoryParams): string {
  if (!params) return ''
  const searchParams = new URLSearchParams()
  if (params.limit) searchParams.set('limit', params.limit.toString())
  if (params.offset) searchParams.set('offset', params.offset.toString())
  const query = searchParams.toString()
  return query ? `?${query}` : ''
}

export const historyApi = {
  async getOrganizationHistory(orgId: string, params?: HistoryParams): Promise<EventHistoryPage> {
    return api.get<EventHistoryPage>(`/organizations/${orgId}/history${buildQuery(params)}`)
  },

  async getFreightRequestHistory(frId: string, params?: HistoryParams): Promise<EventHistoryPage> {
    return api.get<EventHistoryPage>(`/freight-requests/${frId}/history${buildQuery(params)}`)
  },

  async getOrderHistory(orderId: string, params?: HistoryParams): Promise<EventHistoryPage> {
    return api.get<EventHistoryPage>(`/orders/${orderId}/history${buildQuery(params)}`)
  },
}
