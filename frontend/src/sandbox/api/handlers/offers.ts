/**
 * Mock Handlers for Offers
 */

import { registerHandler } from './index'
import { mockOffers, mockFreightRequests } from '@/sandbox/mockData'
import type { MakeOfferRequest } from '@/types/freightRequest'
import { tutorialBus } from '@/sandbox/events'

export function offersHandlers(): void {
  // List offers for freight request
  registerHandler('GET', '/freight-requests/:frId/offers', (params) => {
    const offers = mockOffers.listByFreightRequest(params.frId)
    return { data: offers }
  })

  // Make offer (как перевозчик)
  registerHandler('POST', '/freight-requests/:frId/offers', (params, body) => {
    const data = body as MakeOfferRequest
    const result = mockOffers.create(params.frId, data)

    // Эмитим событие
    tutorialBus.emit('offer:created', {
      frId: params.frId,
      offerId: result.offer_id
    })

    return { data: result }
  })

  // Select offer
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/select', (params) => {
    mockOffers.select(params.frId, params.offerId)

    // Обновляем статус заявки
    const fr = mockFreightRequests.get(params.frId)
    if (fr) {
      fr.status = 'selected'
    }

    // Эмитим событие
    tutorialBus.emit('offer:selected', {
      frId: params.frId,
      offerId: params.offerId
    })

    return { status: 204 }
  })

  // Reject offer
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/reject', (params, body) => {
    mockOffers.reject(params.frId, params.offerId)

    // Эмитим событие
    tutorialBus.emit('offer:rejected', {
      frId: params.frId,
      offerId: params.offerId
    })

    return { status: 204 }
  })

  // Unselect offer
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/unselect', (params, body) => {
    mockOffers.unselect(params.frId, params.offerId)

    // Возвращаем статус заявки на published
    const fr = mockFreightRequests.get(params.frId)
    if (fr) {
      fr.status = 'published'
    }

    // Эмитим событие
    tutorialBus.emit('offer:unselected', {
      frId: params.frId,
      offerId: params.offerId
    })

    return { status: 204 }
  })

  // Confirm offer (создаёт заказ)
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/confirm', (params) => {
    mockOffers.confirm(params.frId, params.offerId)

    // Обновляем статус заявки
    const fr = mockFreightRequests.get(params.frId)
    if (fr) {
      fr.status = 'confirmed'
    }

    // Событие order:created эмитится внутри mockOffers.confirm()
    // через mockOrders.createFromOffer()

    return { status: 204 }
  })

  // Decline offer (перевозчик отказывается от выбранного оффера)
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/decline', (params, body) => {
    const offers = mockOffers.listByFreightRequest(params.frId)
    const offer = offers.find((o) => o.id === params.offerId)
    if (offer) {
      offer.status = 'declined'
    }

    // Возвращаем статус заявки на published
    const fr = mockFreightRequests.get(params.frId)
    if (fr) {
      fr.status = 'published'
    }

    return { status: 204 }
  })

  // Withdraw offer (отзыв своего предложения)
  registerHandler('DELETE', '/freight-requests/:frId/offers/:offerId', (params, body) => {
    mockOffers.withdraw(params.frId, params.offerId)

    // Эмитим событие
    tutorialBus.emit('offer:withdrawn', {
      frId: params.frId,
      offerId: params.offerId
    })

    return { status: 204 }
  })

  // List my offers (для перевозчика)
  registerHandler('GET', '/offers', () => {
    const offers = mockOffers.listMyOffers()
    return { data: offers }
  })
}
