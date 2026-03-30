/**
 * Mock Handlers for FreightRequests
 */

import { registerHandler } from './index'
import type { CreateFreightRequestRequest } from '@/types/freightRequest'
import { tutorialBus } from '@/sandbox/events'

// Используем глобальный singleton напрямую для избежания проблем с HMR
function getMockFreightRequests() {
  return window.__mockFreightRequests!
}

export function freightRequestsHandlers(): void {
  // List freight requests
  registerHandler('GET', '/freight-requests', (_params, _body, query) => {
    const items = getMockFreightRequests().list()

    // Применяем фильтры из query
    let filtered = items

    const status = query?.get('status')
    if (status && status !== 'all') {
      filtered = filtered.filter((fr) => fr.status === status)
    }

    const customerOrgId = query?.get('customer_org_id')
    if (customerOrgId) {
      filtered = filtered.filter((fr) => fr.customer_org_id === customerOrgId)
    }

    // Лимит и offset
    const limit = parseInt(query?.get('limit') || '50', 10)
    const offset = parseInt(query?.get('offset') || '0', 10)
    filtered = filtered.slice(offset, offset + limit)

    return {
      data: {
        items: filtered,
        next_cursor: undefined,
        has_more: false,
      },
    }
  })

  // Get freight request by ID
  registerHandler('GET', '/freight-requests/:id', (params) => {
    const fr = getMockFreightRequests().get(params.id!)
    if (!fr) {
      return {
        status: 404,
        data: { error: 'Заявка не найдена', error_code: 'NOT_FOUND' },
      }
    }
    return { data: fr }
  })

  // Create freight request
  registerHandler('POST', '/freight-requests', async (_params, body) => {
    const data = body as CreateFreightRequestRequest
    const result = getMockFreightRequests().create(data)

    // Эмитим событие для tutorial
    tutorialBus.emit('freightRequest:created', { id: result.id })

    return { data: { id: result.id } }
  })

  // Update freight request
  registerHandler('PATCH', '/freight-requests/:id', (params, _body) => {
    const fr = getMockFreightRequests().get(params.id!)
    if (!fr) {
      return {
        status: 404,
        data: { error: 'Заявка не найдена', error_code: 'NOT_FOUND' },
      }
    }
    // В sandbox не обновляем, просто возвращаем успех
    return { status: 204 }
  })

  // Cancel freight request
  registerHandler('DELETE', '/freight-requests/:id', (params, body) => {
    const reason = (body as { reason?: string })?.reason
    getMockFreightRequests().cancel(params.id!, reason)

    // Эмитим событие
    tutorialBus.emit('freightRequest:cancelled', { id: params.id! })

    return { status: 204 }
  })

  // Reassign freight request
  registerHandler('POST', '/freight-requests/:id/reassign', (_params, _body) => {
    // В sandbox просто принимаем
    return { status: 204 }
  })

  // Complete freight request
  registerHandler('POST', '/freight-requests/:id/complete', (params) => {
    const fr = getMockFreightRequests().get(params.id!)
    if (!fr) {
      return {
        status: 404,
        data: { error: 'Заявка не найдена', error_code: 'NOT_FOUND' },
      }
    }

    // Определяем сторону (в sandbox всегда customer)
    getMockFreightRequests().complete(params.id!, 'customer')

    return { status: 204 }
  })

  // Leave review
  registerHandler('POST', '/freight-requests/:id/review', (params, body) => {
    const fr = getMockFreightRequests().get(params.id!)
    if (!fr) {
      return {
        status: 404,
        data: { error: 'Заявка не найдена', error_code: 'NOT_FOUND' },
      }
    }

    const { rating, comment } = body as { rating: number; comment?: string }

    if (rating < 1 || rating > 5) {
      return {
        status: 400,
        data: { error: 'Рейтинг должен быть от 1 до 5', error_code: 'INVALID_RATING' },
      }
    }

    // В sandbox всегда оставляем отзыв как customer
    const reviewId = getMockFreightRequests().leaveReview(params.id!, 'customer', rating, comment)

    return { status: 201, data: { review_id: reviewId } }
  })

  // Edit review
  registerHandler('PATCH', '/freight-requests/:id/review', (params, body) => {
    const fr = getMockFreightRequests().get(params.id!)
    if (!fr) {
      return {
        status: 404,
        data: { error: 'Заявка не найдена', error_code: 'NOT_FOUND' },
      }
    }

    const { rating, comment } = body as { rating: number; comment?: string }

    if (rating < 1 || rating > 5) {
      return {
        status: 400,
        data: { error: 'Рейтинг должен быть от 1 до 5', error_code: 'INVALID_RATING' },
      }
    }

    // В sandbox всегда редактируем отзыв как customer
    getMockFreightRequests().editReview(params.id!, 'customer', rating, comment)

    return { status: 204 }
  })
}
