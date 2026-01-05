/**
 * Mock Offers Store
 * Mock данные для предложений перевозчиков
 */

import type { Offer, MakeOfferRequest } from '@/types/freightRequest'
import { generateId, randomInt } from './generators'
import { tutorialBus } from '@/sandbox/events'
import { mockOrders } from './orders'

// Контрагенты для симуляции
const CARRIERS = [
  { id: 'carrier-1', name: 'ТрансЛогистик', memberName: 'Иван Петров' },
  { id: 'carrier-2', name: 'СпецГруз', memberName: 'Алексей Смирнов' },
  { id: 'carrier-3', name: 'МегаФура', memberName: 'Дмитрий Козлов' },
]

class MockOfferStore {
  private offersByFr: Map<string, Offer[]> = new Map()
  private myOffers: Map<string, Offer> = new Map()
  private autoConfirmOffers: Set<string> = new Set()

  /**
   * Получить офферы для заявки
   */
  listByFreightRequest(frId: string): Offer[] {
    return this.offersByFr.get(frId) || []
  }

  /**
   * Получить мои офферы
   */
  listMyOffers(): Offer[] {
    return Array.from(this.myOffers.values())
  }

  /**
   * Получить оффер по ID
   */
  get(offerId: string): Offer | null {
    // Ищем в офферах по заявкам
    for (const offers of this.offersByFr.values()) {
      const offer = offers.find((o) => o.id === offerId)
      if (offer) return offer
    }
    // Ищем в моих офферах
    return this.myOffers.get(offerId) || null
  }

  /**
   * Создать оффер (как перевозчик)
   */
  create(frId: string, data: MakeOfferRequest): { offer_id: string } {
    const id = generateId('my-offer')

    const offer: Offer = {
      id,
      carrier_org_id: 'sandbox-org-1',
      carrier_org_name: 'Моя организация (Sandbox)',
      carrier_member_id: 'sandbox-member-1',
      carrier_member_name: 'Я (Sandbox)',
      price: data.price,
      vat_type: data.vat_type,
      payment_method: data.payment_method,
      comment: data.comment,
      status: 'pending',
      freight_version: 1,
      created_at: new Date().toISOString(),
    }

    this.myOffers.set(id, offer)

    // Добавляем в офферы заявки
    const offers = this.offersByFr.get(frId) || []
    offers.push(offer)
    this.offersByFr.set(frId, offers)

    return { offer_id: id }
  }

  /**
   * Симуляция: входящие офферы от других перевозчиков
   */
  async simulateIncomingOffers(frId: string, count: number, delayMs = 500): Promise<void> {
    for (let i = 0; i < count; i++) {
      await new Promise((resolve) => setTimeout(resolve, delayMs * (i + 1)))

      const carrier = CARRIERS[i % CARRIERS.length]
      const basePrice = randomInt(30000, 80000) * 100 // 30k-80k рублей в копейках

      const offer: Offer = {
        id: `sandbox-offer-${i + 1}`,
        carrier_org_id: carrier.id,
        carrier_org_name: carrier.name,
        carrier_member_id: `${carrier.id}-member`,
        carrier_member_name: carrier.memberName,
        price: { amount: basePrice, currency: 'RUB' },
        vat_type: 'included',
        payment_method: 'bank_transfer',
        status: 'pending',
        freight_version: 1,
        created_at: new Date().toISOString(),
      }

      const offers = this.offersByFr.get(frId) || []
      offers.push(offer)
      this.offersByFr.set(frId, offers)
    }
  }

  /**
   * Установить autoconfirm для оффера
   */
  setAutoConfirm(offerId: string, value: boolean): void {
    if (value) {
      this.autoConfirmOffers.add(offerId)
    } else {
      this.autoConfirmOffers.delete(offerId)
    }
  }

  /**
   * Выбрать оффер
   */
  select(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []

    // Сбрасываем предыдущий selected
    offers.forEach((o) => {
      if (o.status === 'selected') o.status = 'pending'
    })

    const offer = offers.find((o) => o.id === offerId)
    if (!offer) throw new Error('Offer not found')

    offer.status = 'selected'

    // Если autoconfirm — сразу confirm
    if (this.autoConfirmOffers.has(offerId)) {
      setTimeout(() => {
        this.confirm(frId, offerId)
      }, 1000)
    }
  }

  /**
   * Отклонить оффер
   */
  reject(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)
    if (offer) {
      offer.status = 'rejected'
    }
  }

  /**
   * Отменить выбор оффера
   */
  unselect(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)
    if (offer && offer.status === 'selected') {
      offer.status = 'pending'
    }
  }

  /**
   * Подтвердить оффер (создаёт заказ)
   */
  confirm(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)
    if (!offer) return

    offer.status = 'confirmed'

    // Создаём заказ
    const orderId = mockOrders.createFromOffer(frId, offer)

    // Отправляем событие
    tutorialBus.emit('order:created', { id: orderId })
  }

  /**
   * Отозвать оффер
   */
  withdraw(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)
    if (offer) {
      offer.status = 'withdrawn'
    }

    // Удаляем из моих офферов
    this.myOffers.delete(offerId)
  }

  /**
   * Симуляция: отклонение моего оффера
   */
  async simulateRejection(offerId: string): Promise<void> {
    const offer = this.myOffers.get(offerId)
    if (offer) {
      offer.status = 'rejected'
    }
  }

  /**
   * Симуляция: выбор моего оффера
   */
  async simulateSelection(offerId: string): Promise<void> {
    const offer = this.myOffers.get(offerId)
    if (offer) {
      offer.status = 'selected'
    }
  }

  /**
   * Очистить store
   */
  clear(): void {
    this.offersByFr.clear()
    this.myOffers.clear()
    this.autoConfirmOffers.clear()
  }
}

export const mockOffers = new MockOfferStore()
