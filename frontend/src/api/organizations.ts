import { api } from './client'
import type { RegisterRequest, RegisterResponse } from '@/types/registration'
import type { OrganizationDetail, OrganizationRating, OrganizationReviewsResponse } from '@/types/admin'

export const organizationsApi = {
  register(data: RegisterRequest): Promise<RegisterResponse> {
    return api.post('/organizations', data)
  },

  get(id: string): Promise<OrganizationDetail> {
    return api.get(`/organizations/${id}`)
  },

  getRating(id: string): Promise<OrganizationRating> {
    return api.get(`/organizations/${id}/rating`)
  },

  async getReviews(id: string, params?: { limit?: number; offset?: number }): Promise<OrganizationReviewsResponse> {
    const query = new URLSearchParams()
    if (params?.limit) query.set('limit', params.limit.toString())
    if (params?.offset) query.set('offset', params.offset.toString())
    const queryStr = query.toString()
    const result = await api.get<OrganizationReviewsResponse | null>(`/organizations/${id}/reviews${queryStr ? `?${queryStr}` : ''}`)
    return result ?? { items: [], total: 0 }
  },
}
