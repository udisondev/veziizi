/**
 * Mock Orders Store
 * Mock данные для заказов
 */

import type { Order, OrderListItem, OrderMessage, OrderDocument, OrderReview } from '@/types/order'
import type { Offer } from '@/types/freightRequest'
import { generateId, generateOrderNumber } from './generators'
import { mockFreightRequests } from './freightRequests'

class MockOrderStore {
  private items: Map<string, Order> = new Map()
  private orderCounter = 0

  /**
   * Получить список заказов
   */
  list(): OrderListItem[] {
    return Array.from(this.items.values()).map((order) => ({
      id: order.id,
      order_number: order.order_number,
      freight_request_id: order.freight_request_id,
      customer_org_id: order.customer_org_id,
      customer_org_name: order.customer_org_name,
      carrier_org_id: order.carrier_org_id,
      carrier_org_name: order.carrier_org_name,
      status: order.status,
      created_at: order.created_at,
    }))
  }

  /**
   * Получить заказ по ID
   */
  get(id: string): Order | null {
    return this.items.get(id) || null
  }

  /**
   * Создать заказ из оффера
   */
  createFromOffer(frId: string, offer: Offer): string {
    const fr = mockFreightRequests.get(frId)
    this.orderCounter++
    const id = `sandbox-order-${this.orderCounter}`

    const order: Order = {
      id,
      order_number: generateOrderNumber(),
      freight_request_id: frId,
      offer_id: offer.id,
      customer_org_id: fr?.customer_org_id || 'sandbox-org-1',
      customer_org_name: fr?.customer_org_name || 'Моя организация (Sandbox)',
      customer_member_id: fr?.customer_member_id || 'sandbox-member-1',
      customer_member_name: 'Я (Sandbox)',
      carrier_org_id: offer.carrier_org_id,
      carrier_org_name: offer.carrier_org_name || 'Перевозчик (Sandbox)',
      carrier_member_id: offer.carrier_member_id,
      carrier_member_name: offer.carrier_member_name || 'Водитель',
      status: 'active',
      messages: [],
      documents: [],
      reviews: [],
      created_at: new Date().toISOString(),
    }

    this.items.set(id, order)
    return id
  }

  /**
   * Добавить сообщение
   */
  addMessage(orderId: string, message: Omit<OrderMessage, 'id' | 'created_at'>): OrderMessage {
    const order = this.items.get(orderId)
    if (!order) throw new Error('Order not found')

    const fullMessage: OrderMessage = {
      id: generateId('msg'),
      ...message,
      created_at: new Date().toISOString(),
    }

    order.messages.push(fullMessage)
    return fullMessage
  }

  /**
   * Добавить документ
   */
  addDocument(orderId: string, doc: Omit<OrderDocument, 'id' | 'created_at'>): OrderDocument {
    const order = this.items.get(orderId)
    if (!order) throw new Error('Order not found')

    const fullDoc: OrderDocument = {
      id: generateId('doc'),
      ...doc,
      created_at: new Date().toISOString(),
    }

    order.documents.push(fullDoc)
    return fullDoc
  }

  /**
   * Удалить документ
   */
  removeDocument(orderId: string, docId: string): void {
    const order = this.items.get(orderId)
    if (order) {
      order.documents = order.documents.filter((d) => d.id !== docId)
    }
  }

  /**
   * Завершить заказ (с одной стороны)
   */
  complete(orderId: string, side: 'customer' | 'carrier'): void {
    const order = this.items.get(orderId)
    if (!order) return

    if (side === 'customer') {
      if (order.status === 'carrier_completed') {
        order.status = 'completed'
        order.completed_at = new Date().toISOString()
      } else if (order.status === 'active') {
        order.status = 'customer_completed'
      }
    } else {
      if (order.status === 'customer_completed') {
        order.status = 'completed'
        order.completed_at = new Date().toISOString()
      } else if (order.status === 'active') {
        order.status = 'carrier_completed'
      }
    }
  }

  /**
   * Симуляция: перевозчик завершил
   */
  simulateCarrierComplete(orderId: string): void {
    const order = this.items.get(orderId)
    if (order && order.status === 'active') {
      order.status = 'carrier_completed'
    }
  }

  /**
   * Симуляция: заказчик завершил
   */
  simulateCustomerComplete(orderId: string): void {
    const order = this.items.get(orderId)
    if (order && order.status === 'active') {
      order.status = 'customer_completed'
    }
  }

  /**
   * Отменить заказ
   */
  cancel(orderId: string, side: 'customer' | 'carrier', reason: string): void {
    const order = this.items.get(orderId)
    if (!order) return

    order.status = side === 'customer' ? 'cancelled_by_customer' : 'cancelled_by_carrier'
    order.cancelled_at = new Date().toISOString()
  }

  /**
   * Оставить отзыв
   */
  leaveReview(orderId: string, review: { rating: number; comment: string }): OrderReview {
    const order = this.items.get(orderId)
    if (!order) throw new Error('Order not found')

    const fullReview: OrderReview = {
      id: generateId('review'),
      reviewer_org_id: 'sandbox-org-1',
      reviewed_org_id: order.carrier_org_id,
      rating: review.rating,
      comment: review.comment,
      created_at: new Date().toISOString(),
    }

    order.reviews.push(fullReview)
    return fullReview
  }

  /**
   * Засеять тестовыми заказами для tutorial
   */
  seed(count: number = 1): void {
    this.clear()

    for (let i = 0; i < count; i++) {
      this.orderCounter++
      const id = `sandbox-order-${this.orderCounter}`
      const order: Order = {
        id,
        order_number: generateOrderNumber(),
        freight_request_id: `sandbox-fr-${i + 1}`,
        offer_id: `sandbox-offer-${i + 1}`,
        customer_org_id: 'sandbox-org-1',
        customer_org_name: 'Моя организация (Sandbox)',
        customer_member_id: 'sandbox-member-1',
        customer_member_name: 'Я (Sandbox)',
        carrier_org_id: `carrier-${i + 1}`,
        carrier_org_name: ['ТрансЛогистик', 'СпецГруз', 'МегаФура'][i % 3],
        carrier_member_id: `carrier-member-${i + 1}`,
        carrier_member_name: ['Иван Петров', 'Алексей Смирнов', 'Дмитрий Козлов'][i % 3],
        status: 'active',
        messages: [],
        documents: [],
        reviews: [],
        created_at: new Date(Date.now() - i * 86400000).toISOString(), // каждый заказ на день раньше
      }
      this.items.set(id, order)
    }
  }

  /**
   * Очистить store
   */
  clear(): void {
    this.items.clear()
    this.orderCounter = 0
  }
}

// Глобальный singleton для корректной работы при HMR
declare global {
  interface Window {
    __mockOrders?: MockOrderStore
  }
}

if (!window.__mockOrders) {
  window.__mockOrders = new MockOrderStore()
}

export const mockOrders = window.__mockOrders
