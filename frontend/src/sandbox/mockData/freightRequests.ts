/**
 * Mock Freight Requests Store
 * Mock данные для заявок на перевозку
 */

import type { FreightRequest, FreightRequestListItem, CreateFreightRequestRequest } from '@/types/freightRequest'
import {
  generateId,
  generateRequestNumber,
  randomInt,
  randomItem,
  randomFutureDate,
  randomTime,
  CITIES,
  CARGO_DESCRIPTIONS,
  ADDRESSES,
  CONTACT_NAMES,
  PHONE_NUMBERS,
} from './generators'
import { mockOffers } from './offers'
import { mockNotifications } from './notifications'

class MockFreightRequestStore {
  private items: Map<string, FreightRequest> = new Map()
  private seeded = false
  private frCounter = 0

  /**
   * Заполнить mock данными
   */
  async seed(count: number = 10): Promise<void> {
    if (this.seeded) return

    for (let i = 0; i < count; i++) {
      const fr = this.generateMockRequest()
      this.items.set(fr.id, fr)
    }
    this.seeded = true
  }

  /**
   * Создать заявку с офферами для tutorial
   */
  async seedWithOffers(
    frId: string,
    offersCount: number = 4,
    owner?: {
      customer_org_id: string
      customer_org_name: string
      customer_member_id: string
      customer_member_name?: string
    }
  ): Promise<void> {
    // Создаём заявку с указанным ID
    const fr = this.generateMockRequest()
    fr.id = frId
    fr.customer_org_id = owner?.customer_org_id || 'sandbox-org-1'
    fr.customer_org_name = owner?.customer_org_name || 'Моя организация (Sandbox)'
    fr.customer_member_id = owner?.customer_member_id || 'sandbox-member-1'
    fr.customer_member_name = owner?.customer_member_name
    fr.status = 'published'

    this.items.set(frId, fr)

    // Генерируем офферы для этой заявки (без задержки для tutorial)
    await mockOffers.simulateIncomingOffers(frId, offersCount, 0)

    // Генерируем уведомления о новых предложениях
    mockNotifications.seedOffersNotifications(frId, offersCount)
  }

  /**
   * Получить список заявок
   */
  list(): FreightRequestListItem[] {
    return Array.from(this.items.values()).map((fr) => ({
      id: fr.id,
      request_number: fr.request_number,
      customer_org_id: fr.customer_org_id,
      customer_org_name: fr.customer_org_name,
      customer_member_id: fr.customer_member_id,
      status: fr.status,
      expires_at: fr.expires_at,
      created_at: fr.created_at,
      route: fr.route,
      origin_address: fr.route.points.find((p) => p.is_loading)?.address,
      destination_address: fr.route.points.find((p) => p.is_unloading)?.address,
      cargo_weight: fr.cargo.weight,
      price_amount: fr.payment.price?.amount,
      price_currency: fr.payment.price?.currency,
      vehicle_type: fr.vehicle_requirements.vehicle_type,
      vehicle_subtype: fr.vehicle_requirements.vehicle_subtype,
    }))
  }

  /**
   * Получить заявку по ID
   */
  get(id: string): FreightRequest | null {
    return this.items.get(id) || null
  }

  /**
   * Создать заявку
   */
  create(data: CreateFreightRequestRequest): { id: string; request_number: number } {
    this.frCounter++
    const id = `sandbox-fr-${this.frCounter}`
    const requestNumber = generateRequestNumber()

    const fr: FreightRequest = {
      id,
      request_number: requestNumber,
      customer_org_id: 'sandbox-org-1',
      customer_org_name: 'Моя организация (Sandbox)',
      customer_member_id: 'sandbox-member-1',
      route: data.route,
      cargo: data.cargo,
      vehicle_requirements: data.vehicle_requirements,
      payment: data.payment,
      comment: data.comment,
      status: 'published',
      freight_version: 1,
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      created_at: new Date().toISOString(),
    }

    this.items.set(id, fr)
    return { id, request_number: requestNumber }
  }

  /**
   * Отменить заявку
   */
  cancel(id: string, reason?: string): void {
    const fr = this.items.get(id)
    if (fr) {
      fr.status = 'cancelled'
    }
  }

  /**
   * Завершить заявку (для sandbox)
   */
  complete(id: string, party: 'customer' | 'carrier'): void {
    const fr = this.items.get(id)
    if (!fr) return

    const now = new Date().toISOString()

    if (party === 'customer') {
      fr.customer_completed = true
      fr.customer_completed_at = now
    } else {
      fr.carrier_completed = true
      fr.carrier_completed_at = now
    }

    // Если обе стороны завершили - статус completed
    if (fr.customer_completed && fr.carrier_completed) {
      fr.status = 'completed'
      fr.completed_at = now
    } else {
      fr.status = 'partially_completed'
    }
  }

  /**
   * Оставить отзыв (для sandbox)
   */
  leaveReview(id: string, party: 'customer' | 'carrier', rating: number, comment?: string): string {
    const fr = this.items.get(id)
    if (!fr) return ''

    const reviewId = generateId('review')
    const now = new Date()
    const expiresAt = new Date(now.getTime() + 24 * 60 * 60 * 1000) // +24 часа

    const review = {
      id: reviewId,
      rating,
      comment,
      created_at: now.toISOString(),
      can_edit: true,
      edit_expires_at: expiresAt.toISOString(),
    }

    if (party === 'customer') {
      fr.customer_review = review
    } else {
      fr.carrier_review = review
    }

    return reviewId
  }

  /**
   * Редактировать отзыв (для sandbox)
   */
  editReview(id: string, party: 'customer' | 'carrier', rating: number, comment?: string): void {
    const fr = this.items.get(id)
    if (!fr) return

    const review = party === 'customer' ? fr.customer_review : fr.carrier_review
    if (!review) return

    review.rating = rating
    review.comment = comment
  }

  /**
   * Создать подтверждённую заявку для сценария завершения
   */
  async seedConfirmedRequest(
    frId: string,
    owner: {
      customer_org_id: string
      customer_org_name: string
      customer_member_id: string
      customer_member_name?: string
    }
  ): Promise<void> {
    // Создаём базовую заявку
    const fr = this.generateMockRequest()
    fr.id = frId
    fr.customer_org_id = owner.customer_org_id
    fr.customer_org_name = owner.customer_org_name
    fr.customer_member_id = owner.customer_member_id
    fr.customer_member_name = owner.customer_member_name
    fr.status = 'confirmed'

    // Данные перевозчика (mock)
    fr.carrier_org_id = 'carrier-sandbox-1'
    fr.carrier_org_name = 'ТрансЛогистик'
    fr.carrier_member_id = 'carrier-member-1'
    fr.carrier_member_name = 'Иван Петров'

    // Статусы завершения
    fr.customer_completed = false
    fr.carrier_completed = false

    this.items.set(frId, fr)
  }

  /**
   * Генерация реалистичной заявки
   */
  private generateMockRequest(): FreightRequest {
    const id = generateId('fr')
    const fromCity = randomItem(CITIES)
    let toCity = randomItem(CITIES)
    while (toCity.id === fromCity.id) {
      toCity = randomItem(CITIES)
    }

    const weight = randomInt(1, 20)
    const volume = randomInt(5, 80)
    const pricePerKm = randomInt(30, 60)
    const distance = randomInt(200, 1500)
    const price = pricePerKm * distance * 100 // В копейках

    return {
      id,
      request_number: generateRequestNumber(),
      customer_org_id: randomItem(['customer-1', 'customer-2', 'customer-3']),
      customer_org_name: randomItem(['ООО Ромашка', 'ИП Сидоров', 'АО ТрансКомплект']),
      customer_member_id: `member-${randomInt(1, 10)}`,
      route: {
        points: [
          {
            is_loading: true,
            is_unloading: false,
            country_id: 1,
            city_id: fromCity.id,
            address: `${fromCity.name}, ${randomItem(ADDRESSES)}`,
            coordinates: {
              latitude: 55 + Math.random() * 5,
              longitude: 37 + Math.random() * 20,
            },
            date_from: randomFutureDate(7),
            time_from: randomTime(),
            contact_name: randomItem(CONTACT_NAMES),
            contact_phone: randomItem(PHONE_NUMBERS),
          },
          {
            is_loading: false,
            is_unloading: true,
            country_id: 1,
            city_id: toCity.id,
            address: `${toCity.name}, ${randomItem(ADDRESSES)}`,
            coordinates: {
              latitude: 55 + Math.random() * 5,
              longitude: 37 + Math.random() * 20,
            },
            date_from: randomFutureDate(14),
            time_from: randomTime(),
            contact_name: randomItem(CONTACT_NAMES),
            contact_phone: randomItem(PHONE_NUMBERS),
          },
        ],
      },
      cargo: {
        description: randomItem(CARGO_DESCRIPTIONS),
        weight,
        volume,
        quantity: randomInt(1, 50),
        adr_class: 'none',
      },
      vehicle_requirements: {
        vehicle_type: randomItem(['van', 'flatbed', 'light_truck', 'medium_truck']),
        vehicle_subtype: randomItem(['dry_van', 'refrigerator', 'curtain_side']),
        capacity: weight * 1000,
        volume: volume,
        loading_types: ['rear'],
      },
      payment: {
        price: { amount: price, currency: 'RUB' },
        vat_type: randomItem(['included', 'excluded', 'none']),
        method: randomItem(['bank_transfer', 'cash', 'card']),
        terms: randomItem(['prepaid', 'on_loading', 'on_unloading', 'deferred']),
      },
      status: 'published',
      freight_version: 1,
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      created_at: new Date(Date.now() - randomInt(1, 48) * 60 * 60 * 1000).toISOString(),
    }
  }

  /**
   * Очистить store
   */
  clear(): void {
    this.items.clear()
    this.seeded = false
    this.frCounter = 0
  }
}

// Глобальный singleton для корректной работы при HMR
declare global {
  interface Window {
    __mockFreightRequests?: MockFreightRequestStore
  }
}

if (!window.__mockFreightRequests) {
  window.__mockFreightRequests = new MockFreightRequestStore()
}

export const mockFreightRequests = window.__mockFreightRequests
