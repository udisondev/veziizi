import { api } from './client'
import type {
  CreateFreightRequestRequest,
  CreateFreightRequestResponse,
  FreightRequest,
  FreightRequestListItem,
  Offer,
  MakeOfferRequest,
  MakeOfferResponse,
} from '@/types/freightRequest'

export interface FreightRequestListParams {
  customer_org_id?: string
  member_id?: string
  status?: string
  org_name?: string
  org_inn?: string
  org_country?: string
  limit?: number
  offset?: number
}

export const freightRequestsApi = {
  create(data: CreateFreightRequestRequest): Promise<CreateFreightRequestResponse> {
    return api.post('/freight-requests', data)
  },

  async list(params?: FreightRequestListParams): Promise<FreightRequestListItem[]> {
    const searchParams = new URLSearchParams()
    if (params?.customer_org_id) searchParams.set('customer_org_id', params.customer_org_id)
    if (params?.member_id) searchParams.set('member_id', params.member_id)
    if (params?.status) searchParams.set('status', params.status)
    if (params?.org_name) searchParams.set('org_name', params.org_name)
    if (params?.org_inn) searchParams.set('org_inn', params.org_inn)
    if (params?.org_country) searchParams.set('org_country', params.org_country)
    if (params?.limit) searchParams.set('limit', params.limit.toString())
    if (params?.offset) searchParams.set('offset', params.offset.toString())

    const query = searchParams.toString()
    const result = await api.get<FreightRequestListItem[] | null>(`/freight-requests${query ? `?${query}` : ''}`)
    return result ?? []
  },

  get(id: string): Promise<FreightRequest> {
    return api.get(`/freight-requests/${id}`)
  },

  update(id: string, data: Partial<CreateFreightRequestRequest>): Promise<void> {
    return api.patch(`/freight-requests/${id}`, data)
  },

  cancel(id: string, reason?: string): Promise<void> {
    return api.delete(`/freight-requests/${id}`, reason ? { reason } : undefined)
  },

  // Offers
  async listOffers(frId: string): Promise<Offer[]> {
    const result = await api.get<Offer[] | null>(`/freight-requests/${frId}/offers`)
    return result ?? []
  },

  makeOffer(frId: string, data: MakeOfferRequest): Promise<MakeOfferResponse> {
    return api.post(`/freight-requests/${frId}/offers`, data)
  },

  withdrawOffer(frId: string, offerId: string, reason?: string): Promise<void> {
    return api.delete(`/freight-requests/${frId}/offers/${offerId}`, reason ? { reason } : undefined)
  },

  selectOffer(frId: string, offerId: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/offers/${offerId}/select`)
  },

  rejectOffer(frId: string, offerId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/offers/${offerId}/reject`, reason ? { reason } : undefined)
  },

  confirmOffer(frId: string, offerId: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/offers/${offerId}/confirm`)
  },

  declineOffer(frId: string, offerId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/offers/${offerId}/decline`, reason ? { reason } : undefined)
  },

  reassign(frId: string, newMemberId: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/reassign`, { new_member_id: newMemberId })
  },
}
