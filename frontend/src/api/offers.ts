import { api } from './client'
import type { OfferStatus } from '@/types/freightRequest'

export interface MyOfferListItem {
  id: string
  freight_request_id: string
  carrier_org_id: string
  status: OfferStatus
  created_at: string
  origin_address?: string
  destination_address?: string
  cargo_weight?: number
  price_amount?: number
  price_currency?: string
}

export interface OfferListParams {
  status?: OfferStatus
  limit?: number
  offset?: number
}

export const offersApi = {
  async listMy(params?: OfferListParams): Promise<MyOfferListItem[]> {
    const searchParams = new URLSearchParams()
    if (params?.status) searchParams.set('status', params.status)
    if (params?.limit) searchParams.set('limit', params.limit.toString())
    if (params?.offset) searchParams.set('offset', params.offset.toString())

    const query = searchParams.toString()
    const result = await api.get<MyOfferListItem[] | null>(`/offers${query ? `?${query}` : ''}`)
    return result ?? []
  },

  async withdraw(freightRequestId: string, offerId: string): Promise<void> {
    return api.delete(`/freight-requests/${freightRequestId}/offers/${offerId}`)
  },

  async confirm(freightRequestId: string, offerId: string): Promise<void> {
    return api.post(`/freight-requests/${freightRequestId}/offers/${offerId}/confirm`)
  },

  async decline(freightRequestId: string, offerId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${freightRequestId}/offers/${offerId}/decline`, reason ? { reason } : undefined)
  },
}
