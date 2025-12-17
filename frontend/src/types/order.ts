// View mode for orders list
export type ViewMode = 'all' | 'as_customer' | 'as_carrier'

export const viewModeOptions: { value: ViewMode; label: string }[] = [
  { value: 'all', label: 'Все заказы' },
  { value: 'as_customer', label: 'Как заказчик' },
  { value: 'as_carrier', label: 'Как перевозчик' },
]

export const viewModeLabels: Record<ViewMode, string> = {
  all: 'Все заказы',
  as_customer: 'Как заказчик',
  as_carrier: 'Как перевозчик',
}

// Order statuses (synchronized with backend values/status.go)
export type OrderStatus =
  | 'active'
  | 'customer_completed'
  | 'carrier_completed'
  | 'completed'
  | 'cancelled_by_customer'
  | 'cancelled_by_carrier'

// Order list item (from projection)
export interface OrderListItem {
  id: string
  freight_request_id: string
  customer_org_id: string
  carrier_org_id: string
  status: OrderStatus
  created_at: string
}

// Message in order chat
export interface OrderMessage {
  id: string
  sender_org_id: string
  sender_member_id: string
  content: string
  created_at: string
}

// Document attached to order
export interface OrderDocument {
  id: string
  name: string
  mime_type: string
  size: number
  uploaded_by: string
  created_at: string
}

// Review left after order completion
export interface OrderReview {
  id: string
  reviewer_org_id: string
  rating: number // 1-5
  comment: string
  created_at: string
}

// Full order data (from GET /orders/{id})
export interface Order {
  id: string
  freight_request_id: string
  offer_id: string
  customer_org_id: string
  customer_org_name: string
  customer_member_id: string
  customer_member_name: string
  carrier_org_id: string
  carrier_org_name: string
  carrier_member_id: string
  carrier_member_name: string
  status: OrderStatus
  messages: OrderMessage[]
  documents: OrderDocument[]
  reviews: OrderReview[]
  created_at: string
  completed_at?: string
  cancelled_at?: string
}

// Request types
export interface SendMessageRequest {
  content: string
}

export interface CancelOrderRequest {
  reason?: string
}

export interface LeaveReviewRequest {
  rating: number // 1-5
  comment?: string
}

// Labels for UI
export const orderStatusLabels: Record<OrderStatus, string> = {
  active: 'Активный',
  customer_completed: 'Завершён заказчиком',
  carrier_completed: 'Завершён перевозчиком',
  completed: 'Завершён',
  cancelled_by_customer: 'Отменён заказчиком',
  cancelled_by_carrier: 'Отменён перевозчиком',
}

// Colors for status badges
export const orderStatusColors: Record<OrderStatus, string> = {
  active: 'bg-green-100 text-green-800',
  customer_completed: 'bg-yellow-100 text-yellow-800',
  carrier_completed: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-blue-100 text-blue-800',
  cancelled_by_customer: 'bg-red-100 text-red-800',
  cancelled_by_carrier: 'bg-red-100 text-red-800',
}

// Options for status filter select
export const orderStatusOptions: { value: OrderStatus | ''; label: string }[] = [
  { value: '', label: 'Все статусы' },
  { value: 'active', label: 'Активные' },
  { value: 'customer_completed', label: 'Ожидают перевозчика' },
  { value: 'carrier_completed', label: 'Ожидают заказчика' },
  { value: 'completed', label: 'Завершённые' },
  { value: 'cancelled_by_customer', label: 'Отменены заказчиком' },
  { value: 'cancelled_by_carrier', label: 'Отменены перевозчиком' },
]

// Helper functions
export function isOrderFinished(status: OrderStatus): boolean {
  return ['completed', 'cancelled_by_customer', 'cancelled_by_carrier'].includes(status)
}

export function isOrderCancelled(status: OrderStatus): boolean {
  return ['cancelled_by_customer', 'cancelled_by_carrier'].includes(status)
}

export function isOrderActive(status: OrderStatus): boolean {
  return ['active', 'customer_completed', 'carrier_completed'].includes(status)
}
