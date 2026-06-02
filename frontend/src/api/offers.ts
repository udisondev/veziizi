import { api } from './client'
import type { OfferStatus, PaymentMethod, VatType } from '@/types/freightRequest'
import type { CursorPaginatedResponse } from './freightRequests'

export interface OutgoingOfferListItem {
  id: string
  freight_request_id: string
  freight_request_number?: number
  carrier_org_id: string
  customer_org_name?: string
  status: OfferStatus
  price_amount?: number
  price_currency?: string
  payment_method?: PaymentMethod
  vat_type?: VatType
  comment?: string
  created_at: string
  origin_address?: string
  destination_address?: string
  cargo_weight?: number
}

export interface IncomingOfferListItem {
  id: string
  freight_request_id: string
  freight_request_number?: number
  carrier_org_id: string
  carrier_org_name?: string
  carrier_member_id?: string
  carrier_member_name?: string
  // Поля цены и условий оплаты добавятся когда бэкенд обогатит offers_lookup
  price_amount?: number
  price_currency?: string
  vat_type?: VatType
  payment_method?: PaymentMethod
  comment?: string
  status: OfferStatus
  created_at: string
  origin_address?: string
  destination_address?: string
}

// backward compat alias
export type MyOfferListItem = OutgoingOfferListItem

export interface OutgoingOfferListParams {
  statuses?: string
  member_id?: string
  min_price?: number
  max_price?: number
  payment_methods?: string
  vat_types?: string
  sort_by?: 'created_at_desc' | 'price_desc' | 'price_asc'
  limit?: number
  cursor?: string
}

export interface IncomingOfferListParams {
  statuses?: string
  member_id?: string
  carrier_org_name?: string
  min_price?: number
  max_price?: number
  payment_methods?: string
  vat_types?: string
  freight_request_number?: number
  sort_by?: 'created_at_desc' | 'price_desc' | 'price_asc'
  limit?: number
  cursor?: string
}

// legacy type kept for compat
export interface OfferListParams {
  status?: OfferStatus
  member_id?: string
  limit?: number
  offset?: number
}

export const offersApi = {
  async listOutgoing(params?: OutgoingOfferListParams): Promise<CursorPaginatedResponse<OutgoingOfferListItem>> {
    const p = new URLSearchParams()
    if (params?.statuses) p.set('statuses', params.statuses)
    if (params?.member_id) p.set('member_id', params.member_id)
    if (params?.min_price !== undefined) p.set('min_price', params.min_price.toString())
    if (params?.max_price !== undefined) p.set('max_price', params.max_price.toString())
    if (params?.payment_methods) p.set('payment_methods', params.payment_methods)
    if (params?.vat_types) p.set('vat_types', params.vat_types)
    if (params?.sort_by) p.set('sort_by', params.sort_by)
    if (params?.limit) p.set('limit', params.limit.toString())
    if (params?.cursor) p.set('cursor', params.cursor)
    const q = p.toString()
    const raw = await api.get<CursorPaginatedResponse<OutgoingOfferListItem> | OutgoingOfferListItem[] | null>(`/offers${q ? `?${q}` : ''}`)
    if (!raw) return { items: [], has_more: false }
    if (Array.isArray(raw)) return { items: raw, has_more: false }
    return raw
  },

  async listIncoming(params?: IncomingOfferListParams): Promise<CursorPaginatedResponse<IncomingOfferListItem>> {
    const p = new URLSearchParams()
    if (params?.statuses) p.set('statuses', params.statuses)
    if (params?.member_id) p.set('member_id', params.member_id)
    if (params?.carrier_org_name) p.set('carrier_org_name', params.carrier_org_name)
    if (params?.min_price !== undefined) p.set('min_price', params.min_price.toString())
    if (params?.max_price !== undefined) p.set('max_price', params.max_price.toString())
    if (params?.payment_methods) p.set('payment_methods', params.payment_methods)
    if (params?.vat_types) p.set('vat_types', params.vat_types)
    if (params?.freight_request_number) p.set('freight_request_number', params.freight_request_number.toString())
    if (params?.sort_by) p.set('sort_by', params.sort_by)
    if (params?.limit) p.set('limit', params.limit.toString())
    if (params?.cursor) p.set('cursor', params.cursor)
    const q = p.toString()
    const result = await api.get<CursorPaginatedResponse<IncomingOfferListItem> | null>(`/offers/incoming${q ? `?${q}` : ''}`)
    return result ?? { items: [], has_more: false }
  },

  async listMy(params?: OfferListParams): Promise<OutgoingOfferListItem[]> {
    const res = await offersApi.listOutgoing({
      statuses: params?.status,
      member_id: params?.member_id,
      limit: params?.limit ?? 100,
    })
    return res.items
  },

  withdraw(freightRequestId: string, offerId: string): Promise<void> {
    return api.delete(`/freight-requests/${freightRequestId}/offers/${offerId}`)
  },

  confirm(freightRequestId: string, offerId: string): Promise<void> {
    return api.post(`/freight-requests/${freightRequestId}/offers/${offerId}/confirm`)
  },

  decline(freightRequestId: string, offerId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${freightRequestId}/offers/${offerId}/decline`, reason ? { reason } : undefined)
  },

  select(freightRequestId: string, offerId: string): Promise<void> {
    return api.post(`/freight-requests/${freightRequestId}/offers/${offerId}/select`)
  },

  reject(freightRequestId: string, offerId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${freightRequestId}/offers/${offerId}/reject`, reason ? { reason } : undefined)
  },

  unselect(freightRequestId: string, offerId: string, reason?: string): Promise<void> {
    return api.post(`/freight-requests/${freightRequestId}/offers/${offerId}/unselect`, reason ? { reason } : undefined)
  },
}
