import { api } from './client'
import type {
  CreateFreightRequestRequest,
  CreateFreightRequestResponse,
  FreightRequest,
  FreightRequestListItem,
  CarrierInviteItem,
  Offer,
  MakeOfferRequest,
  MakeOfferResponse,
} from '@/types/freightRequest'

/**
 * Ответ с cursor-based пагинацией
 */
export interface CursorPaginatedResponse<T> {
  items: T[]
  next_cursor?: string
  has_more: boolean
}

export interface FreightRequestListParams {
  customer_org_id?: string
  carrier_org_id?: string
  member_id?: string
  statuses?: string  // comma-separated statuses
  org_name?: string
  org_inn?: string
  org_country?: string
  request_number?: number
  // Extended filters (subscription-like)
  min_weight?: number
  max_weight?: number
  min_price?: number
  max_price?: number
  min_volume?: number
  max_volume?: number
  vehicle_types?: string  // comma-separated
  vehicle_subtypes?: string   // comma-separated
  payment_methods?: string // comma-separated
  payment_terms?: string // comma-separated
  vat_types?: string // comma-separated
  route_city_ids?: string // comma-separated city IDs (любая точка маршрута)
  route_country_ids?: string // comma-separated country IDs (любая точка, без города)
  origin_city_ids?: string // города начала маршрута
  origin_country_ids?: string // страны начала маршрута (без города)
  destination_city_ids?: string // города конца маршрута
  destination_country_ids?: string // страны конца маршрута (без города)
  has_pending_offers?: boolean
  not_expired?: boolean
  sort_by?: 'created_at_desc' | 'expires_at_asc' | 'price_desc' | 'weight_desc' | 'loading_date_asc' | 'offers_count_asc'
  limit?: number
  cursor?: string // cursor для keyset pagination
}

export const freightRequestsApi = {
  create(data: CreateFreightRequestRequest): Promise<CreateFreightRequestResponse> {
    return api.post('/freight-requests', data)
  },

  async list(params?: FreightRequestListParams): Promise<CursorPaginatedResponse<FreightRequestListItem>> {
    const searchParams = new URLSearchParams()
    if (params?.customer_org_id) searchParams.set('customer_org_id', params.customer_org_id)
    if (params?.carrier_org_id) searchParams.set('carrier_org_id', params.carrier_org_id)
    if (params?.member_id) searchParams.set('member_id', params.member_id)
    if (params?.statuses) searchParams.set('statuses', params.statuses)
    if (params?.org_name) searchParams.set('org_name', params.org_name)
    if (params?.org_inn) searchParams.set('org_inn', params.org_inn)
    if (params?.org_country) searchParams.set('org_country', params.org_country)
    if (params?.request_number) searchParams.set('request_number', params.request_number.toString())
    // Extended filters
    if (params?.min_weight !== undefined) searchParams.set('min_weight', params.min_weight.toString())
    if (params?.max_weight !== undefined) searchParams.set('max_weight', params.max_weight.toString())
    if (params?.min_price !== undefined) searchParams.set('min_price', params.min_price.toString())
    if (params?.max_price !== undefined) searchParams.set('max_price', params.max_price.toString())
    if (params?.min_volume !== undefined) searchParams.set('min_volume', params.min_volume.toString())
    if (params?.max_volume !== undefined) searchParams.set('max_volume', params.max_volume.toString())
    if (params?.vehicle_types) searchParams.set('vehicle_types', params.vehicle_types)
    if (params?.vehicle_subtypes) searchParams.set('vehicle_subtypes', params.vehicle_subtypes)
    if (params?.payment_methods) searchParams.set('payment_methods', params.payment_methods)
    if (params?.payment_terms) searchParams.set('payment_terms', params.payment_terms)
    if (params?.vat_types) searchParams.set('vat_types', params.vat_types)
    if (params?.route_city_ids) searchParams.set('route_city_ids', params.route_city_ids)
    if (params?.route_country_ids) searchParams.set('route_country_ids', params.route_country_ids)
    if (params?.origin_city_ids) searchParams.set('origin_city_ids', params.origin_city_ids)
    if (params?.origin_country_ids) searchParams.set('origin_country_ids', params.origin_country_ids)
    if (params?.destination_city_ids) searchParams.set('destination_city_ids', params.destination_city_ids)
    if (params?.destination_country_ids) searchParams.set('destination_country_ids', params.destination_country_ids)
    if (params?.has_pending_offers) searchParams.set('has_pending_offers', 'true')
    if (params?.not_expired) searchParams.set('not_expired', 'true')
    if (params?.sort_by) searchParams.set('sort_by', params.sort_by)
    // Pagination
    if (params?.limit) searchParams.set('limit', params.limit.toString())
    if (params?.cursor) searchParams.set('cursor', params.cursor)

    const query = searchParams.toString()
    const result = await api.get<CursorPaginatedResponse<FreightRequestListItem> | null>(`/freight-requests${query ? `?${query}` : ''}`)
    if (!result) return { items: [], has_more: false }
    return { ...result, items: result.items ?? [] }
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

  inviteCarrier(frId: string, vehicleId: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/invite-carrier`, { vehicle_id: vehicleId })
  },

  async listCarrierInvitations(params?: { limit?: number; cursor?: string }): Promise<CursorPaginatedResponse<CarrierInviteItem>> {
    const p = new URLSearchParams()
    if (params?.limit) p.set('limit', params.limit.toString())
    if (params?.cursor) p.set('cursor', params.cursor)
    const query = p.toString()
    const result = await api.get<CursorPaginatedResponse<CarrierInviteItem> | null>(`/carrier-invitations${query ? `?${query}` : ''}`)
    if (!result) return { items: [], has_more: false }
    return { ...result, items: result.items ?? [] }
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

  unselectOffer(frId: string, offerId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/offers/${offerId}/unselect`, reason ? { reason } : undefined)
  },

  reassign(frId: string, newMemberId: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/reassign`, { new_member_id: newMemberId })
  },

  reassignCarrier(frId: string, newMemberId: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/reassign-carrier`, { new_member_id: newMemberId })
  },

  // Completion & Reviews
  complete(frId: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/complete`)
  },

  leaveReview(frId: string, data: { rating: number; comment?: string }): Promise<{ review_id: string }> {
    return api.post(`/freight-requests/${frId}/review`, data)
  },

  editReview(frId: string, data: { rating: number; comment?: string }): Promise<void> {
    return api.patch(`/freight-requests/${frId}/review`, data)
  },

  cancelAfterConfirmed(frId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${frId}/cancel-confirmed`, reason ? { reason } : undefined)
  },
}
