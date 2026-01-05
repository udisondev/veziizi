/**
 * Mock Handlers for FreightRequests
 */

import { registerHandler } from './index'
import { mockFreightRequests } from '@/sandbox/mockData'
import type { CreateFreightRequestRequest } from '@/types/freightRequest'
import { tutorialBus } from '@/sandbox/events'

export function freightRequestsHandlers(): void {
  // List freight requests
  registerHandler('GET', '/freight-requests', (params, body, query) => {
    const items = mockFreightRequests.list()

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

    return { data: filtered }
  })

  // Get freight request by ID
  registerHandler('GET', '/freight-requests/:id', (params) => {
    const fr = mockFreightRequests.get(params.id)
    if (!fr) {
      return {
        status: 404,
        data: { error: 'Заявка не найдена', error_code: 'NOT_FOUND' },
      }
    }
    return { data: fr }
  })

  // Create freight request
  registerHandler('POST', '/freight-requests', async (params, body) => {
    const data = body as CreateFreightRequestRequest
    const result = mockFreightRequests.create(data)

    // Эмитим событие для tutorial
    tutorialBus.emit('freightRequest:created', { id: result.id })

    return { data: { id: result.id } }
  })

  // Update freight request
  registerHandler('PATCH', '/freight-requests/:id', (params, body) => {
    const fr = mockFreightRequests.get(params.id)
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
    mockFreightRequests.cancel(params.id, reason)

    // Эмитим событие
    tutorialBus.emit('freightRequest:cancelled', { id: params.id })

    return { status: 204 }
  })

  // Reassign freight request
  registerHandler('POST', '/freight-requests/:id/reassign', (params, body) => {
    // В sandbox просто принимаем
    return { status: 204 }
  })
}
