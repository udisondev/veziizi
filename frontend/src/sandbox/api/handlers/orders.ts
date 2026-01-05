/**
 * Mock Handlers for Orders
 */

import { registerHandler } from './index'
import { mockOrders, mockBot } from '@/sandbox/mockData'
import type { SendMessageRequest, LeaveReviewRequest, Order } from '@/types/order'
import { tutorialBus } from '@/sandbox/events'
import { generateId } from '@/sandbox/mockData/generators'

export function ordersHandlers(): void {
  // List orders
  registerHandler('GET', '/orders', (params, body, query) => {
    const items = mockOrders.list()

    // Применяем фильтры
    let filtered = items

    const status = query?.get('status')
    if (status && status !== 'all') {
      filtered = filtered.filter((o) => o.status === status)
    }

    const customerOrgId = query?.get('customer_org_id')
    if (customerOrgId) {
      filtered = filtered.filter((o) => o.customer_org_id === customerOrgId)
    }

    const carrierOrgId = query?.get('carrier_org_id')
    if (carrierOrgId) {
      filtered = filtered.filter((o) => o.carrier_org_id === carrierOrgId)
    }

    const limit = parseInt(query?.get('limit') || '50', 10)
    const offset = parseInt(query?.get('offset') || '0', 10)
    filtered = filtered.slice(offset, offset + limit)

    return { data: filtered }
  })

  // Get order by ID
  registerHandler('GET', '/orders/:id', (params) => {
    const order = mockOrders.get(params.id)
    if (!order) {
      return {
        status: 404,
        data: { error: 'Заказ не найден', error_code: 'NOT_FOUND' },
      }
    }
    return { data: order }
  })

  // Send message
  registerHandler('POST', '/orders/:id/messages', async (params, body) => {
    const data = body as SendMessageRequest

    const message = mockOrders.addMessage(params.id, {
      sender_org_id: 'sandbox-org-1',
      sender_member_id: 'sandbox-member-1',
      content: data.content,
    })

    // Эмитим событие
    tutorialBus.emit('message:sent', { orderId: params.id })

    // Планируем ответ бота
    await mockBot.scheduleReply(params.id, 1500)

    return { status: 204 }
  })

  // Upload document (симуляция)
  registerHandler('POST', '/orders/:id/documents', (params, body) => {
    // В реальности это multipart, но для sandbox просто симулируем
    const doc = mockOrders.addDocument(params.id, {
      name: `document-${Date.now()}.pdf`,
      mime_type: 'application/pdf',
      size: 1024 * 100, // 100KB
      uploaded_by: 'sandbox-member-1',
    })

    // Эмитим событие
    tutorialBus.emit('document:uploaded', { orderId: params.id, docId: doc.id })

    return { data: doc }
  })

  // Download document (симуляция)
  registerHandler('GET', '/orders/:id/documents/:docId', (params) => {
    // Возвращаем mock blob URL
    return {
      data: {
        url: 'data:application/pdf;base64,JVBERi0x', // Minimal PDF
        mimeType: 'application/pdf',
      },
    }
  })

  // Remove document
  registerHandler('DELETE', '/orders/:id/documents/:docId', (params) => {
    mockOrders.removeDocument(params.id, params.docId)
    return { status: 204 }
  })

  // Complete order
  registerHandler('POST', '/orders/:id/complete', (params) => {
    mockOrders.complete(params.id, 'customer')

    // Эмитим событие
    tutorialBus.emit('order:completed', { orderId: params.id })

    return { status: 204 }
  })

  // Cancel order
  registerHandler('POST', '/orders/:id/cancel', (params, body) => {
    const reason = (body as { reason?: string })?.reason
    mockOrders.cancel(params.id, 'customer', reason || '')

    // Эмитим событие
    tutorialBus.emit('order:cancelled', { orderId: params.id })

    return { status: 204 }
  })

  // Leave review
  registerHandler('POST', '/orders/:id/review', (params, body) => {
    const data = body as LeaveReviewRequest

    const review = mockOrders.leaveReview(params.id, {
      rating: data.rating,
      comment: data.comment || '',
    })

    // Эмитим событие
    tutorialBus.emit('review:left', { orderId: params.id })

    return { data: review }
  })

  // Reassign order
  registerHandler('POST', '/orders/:id/reassign', (params, body) => {
    // В sandbox просто принимаем
    return { status: 204 }
  })
}
