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
  status?: string
  limit?: number
  offset?: number
}

export const freightRequestsApi = {
  create(data: CreateFreightRequestRequest): Promise<CreateFreightRequestResponse> {
    return api.post('/freight-requests', data)
  },

  list(params?: FreightRequestListParams): Promise<FreightRequestListItem[]> {
    const searchParams = new URLSearchParams()
    if (params?.customer_org_id) searchParams.set('customer_org_id', params.customer_org_id)
    if (params?.status) searchParams.set('status', params.status)
    if (params?.limit) searchParams.set('limit', params.limit.toString())
    if (params?.offset) searchParams.set('offset', params.offset.toString())

    const query = searchParams.toString()
    return api.get(`/freight-requests${query ? `?${query}` : ''}`)
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
  listOffers(frId: string): Promise<Offer[]> {
    return api.get(`/freight-requests/${frId}/offers`)
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
}
