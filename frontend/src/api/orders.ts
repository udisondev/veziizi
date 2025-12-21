import { api } from './client'
import type {
  Order,
  OrderListItem,
  SendMessageRequest,
  LeaveReviewRequest,
} from '@/types/order'

export interface OrderListParams {
  customer_org_id?: string
  carrier_org_id?: string
  freight_request_id?: string
  status?: string
  limit?: number
  offset?: number
}

export const ordersApi = {
  // List orders with optional filters
  async list(params?: OrderListParams): Promise<OrderListItem[]> {
    const searchParams = new URLSearchParams()
    if (params?.customer_org_id) searchParams.set('customer_org_id', params.customer_org_id)
    if (params?.carrier_org_id) searchParams.set('carrier_org_id', params.carrier_org_id)
    if (params?.freight_request_id) searchParams.set('freight_request_id', params.freight_request_id)
    if (params?.status) searchParams.set('status', params.status)
    if (params?.limit) searchParams.set('limit', params.limit.toString())
    if (params?.offset) searchParams.set('offset', params.offset.toString())

    const query = searchParams.toString()
    const result = await api.get<OrderListItem[] | null>(`/orders${query ? `?${query}` : ''}`)
    return result ?? []
  },

  // Get order details
  get(id: string): Promise<Order> {
    return api.get(`/orders/${id}`)
  },

  // Send message in order chat
  sendMessage(orderId: string, data: SendMessageRequest): Promise<void> {
    return api.post(`/orders/${orderId}/messages`, data)
  },

  // Upload document (multipart form)
  async uploadDocument(orderId: string, file: File): Promise<void> {
    const formData = new FormData()
    formData.append('file', file)

    const response = await fetch(`/api/v1/orders/${orderId}/documents`, {
      method: 'POST',
      body: formData,
      credentials: 'include',
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: `HTTP ${response.status}` }))
      throw new Error(error.error)
    }
  },

  // Download document (returns blob URL)
  async downloadDocument(orderId: string, docId: string): Promise<{ url: string; mimeType: string }> {
    const response = await fetch(`/api/v1/orders/${orderId}/documents/${docId}`, {
      credentials: 'include',
    })

    if (!response.ok) {
      throw new Error('Не удалось скачать документ')
    }

    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const mimeType = response.headers.get('Content-Type') || 'application/octet-stream'

    return { url, mimeType }
  },

  // Remove document
  removeDocument(orderId: string, docId: string): Promise<void> {
    return api.delete(`/orders/${orderId}/documents/${docId}`)
  },

  // Mark order as complete from current org side
  complete(orderId: string): Promise<void> {
    return api.post(`/orders/${orderId}/complete`)
  },

  // Cancel order
  cancel(orderId: string, reason?: string): Promise<void> {
    return api.post(`/orders/${orderId}/cancel`, reason ? { reason } : undefined)
  },

  // Leave review after order completion
  leaveReview(orderId: string, data: LeaveReviewRequest): Promise<void> {
    return api.post(`/orders/${orderId}/review`, data)
  },

  // Reassign responsible member in order
  reassign(orderId: string, newMemberId: string): Promise<void> {
    return api.post(`/orders/${orderId}/reassign`, { new_member_id: newMemberId })
  },
}
