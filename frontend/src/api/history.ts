import { api } from './client'

export interface Actor {
  id: string
  name: string
  email: string
}

export interface DisplayField {
  label: string
  value: string
  type?: 'text' | 'money' | 'date' | 'status'
}

export interface DiffValue {
  label: string
  old_value: string
  new_value: string
}

export interface DisplayView {
  title: string
  description: string
  fields?: DisplayField[]
  diffs?: DiffValue[]
  icon?: string
  severity?: 'info' | 'success' | 'warning' | 'error'
}

export interface DisplayableHistoryItem {
  id: string
  event_type: string
  version: number
  occurred_at: string
  actor?: Actor
  display: DisplayView
}

export interface DisplayableHistoryPage {
  items: DisplayableHistoryItem[]
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
  async getOrganizationHistory(orgId: string, params?: HistoryParams): Promise<DisplayableHistoryPage> {
    return api.get<DisplayableHistoryPage>(`/organizations/${orgId}/history${buildQuery(params)}`)
  },

  async getFreightRequestHistory(frId: string, params?: HistoryParams): Promise<DisplayableHistoryPage> {
    return api.get<DisplayableHistoryPage>(`/freight-requests/${frId}/history${buildQuery(params)}`)
  },
}
