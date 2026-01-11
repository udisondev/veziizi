import { api } from './client'
import type { RegisterRequest, RegisterResponse } from '@/types/registration'
import type { OrganizationDetail, OrganizationRating, OrganizationReview } from '@/types/admin'
import { type CursorPaginatedResponse } from './freightRequests'

export const organizationsApi = {
  register(data: RegisterRequest, fingerprint?: string): Promise<RegisterResponse> {
    const headers: Record<string, string> = {}
    if (fingerprint) {
      headers['X-Fingerprint'] = fingerprint
    }
    return api.post('/organizations', data, { headers })
  },

  get(id: string): Promise<OrganizationDetail> {
    return api.get(`/organizations/${id}`)
  },

  getRating(id: string): Promise<OrganizationRating> {
    return api.get(`/organizations/${id}/rating`)
  },

  async getReviews(id: string, params?: { limit?: number; cursor?: string }): Promise<CursorPaginatedResponse<OrganizationReview>> {
    const query = new URLSearchParams()
    if (params?.limit) query.set('limit', params.limit.toString())
    if (params?.cursor) query.set('cursor', params.cursor)
    const queryStr = query.toString()
    const result = await api.get<CursorPaginatedResponse<OrganizationReview> | null>(`/organizations/${id}/reviews${queryStr ? `?${queryStr}` : ''}`)
    return result ?? { items: [], has_more: false }
  },
}
