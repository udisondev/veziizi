/**
 * Mock Handlers for Offers
 */

import { registerHandler } from './index'
import type { MakeOfferRequest } from '@/types/freightRequest'
import { tutorialBus } from '@/sandbox/events'

// Используем глобальный singleton напрямую для избежания проблем с HMR
function getMockOffers() {
  return window.__mockOffers!
}

export function offersHandlers(): void {
  // List offers for freight request
  registerHandler('GET', '/freight-requests/:frId/offers', (params) => {
    const offers = getMockOffers().listByFreightRequest(params.frId!)
    return { data: offers }
  })

  // Make offer (как перевозчик)
  registerHandler('POST', '/freight-requests/:frId/offers', (params, body) => {
    const data = body as MakeOfferRequest
    const result = getMockOffers().create(params.frId!, data)

    // Эмитим событие
    tutorialBus.emit('offer:created', {
      frId: params.frId!,
      offerId: result.offer_id
    })

    return { data: result }
  })

  // Select offer
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/select', (params) => {
    // Статус FR обновляется внутри mockOffers.select()
    getMockOffers().select(params.frId!, params.offerId!)

    // Эмитим событие
    tutorialBus.emit('offer:selected', {
      frId: params.frId!,
      offerId: params.offerId!
    })

    return { status: 204 }
  })

  // Reject offer
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/reject', (params, _body) => {
    getMockOffers().reject(params.frId!, params.offerId!)

    // Эмитим событие
    tutorialBus.emit('offer:rejected', {
      frId: params.frId!,
      offerId: params.offerId!
    })

    return { status: 204 }
  })

  // Unselect offer
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/unselect', (params) => {
    // Статус FR обновляется внутри mockOffers.unselect()
    getMockOffers().unselect(params.frId!, params.offerId!)

    // Эмитим событие
    tutorialBus.emit('offer:unselected', {
      frId: params.frId!,
      offerId: params.offerId!
    })

    return { status: 204 }
  })

  // Confirm offer (создаёт заказ)
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/confirm', (params) => {
    // Статус FR обновляется и order:created эмитится внутри mockOffers.confirm()
    getMockOffers().confirm(params.frId!, params.offerId!)

    return { status: 204 }
  })

  // Decline offer (перевозчик отказывается от выбранного оффера)
  registerHandler('POST', '/freight-requests/:frId/offers/:offerId/decline', (params) => {
    // Статус FR обновляется внутри mockOffers.decline()
    getMockOffers().decline(params.frId!, params.offerId!)

    // Эмитим событие
    tutorialBus.emit('offer:declined', {
      frId: params.frId!,
      offerId: params.offerId!
    })

    return { status: 204 }
  })

  // Withdraw offer (отзыв своего предложения)
  registerHandler('DELETE', '/freight-requests/:frId/offers/:offerId', (params, _body) => {
    getMockOffers().withdraw(params.frId!, params.offerId!)

    // Эмитим событие
    tutorialBus.emit('offer:withdrawn', {
      frId: params.frId!,
      offerId: params.offerId!
    })

    return { status: 204 }
  })

  // List my offers (для перевозчика)
  registerHandler('GET', '/offers', () => {
    const offers = getMockOffers().listMyOffers()
    return { data: offers }
  })
}
