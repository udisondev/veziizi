/**
 * Mock Offers Store
 * Mock данные для предложений перевозчиков
 */

import type { Offer, MakeOfferRequest } from '@/types/freightRequest'
import { generateId, randomInt } from './generators'
import { tutorialBus } from '@/sandbox/events'
import { AUTO_CONFIRM_DELAY_MS } from '@/sandbox/constants'

// Ленивый импорт для избежания циклических зависимостей
function getMockFreightRequests() {
  return window.__mockFreightRequests!
}

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
  private offerCounter = 0

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

      this.offerCounter++
      const carrier = CARRIERS[i % CARRIERS.length]!
      const basePrice = randomInt(30000, 80000) * 100 // 30k-80k рублей в копейках

      const offer: Offer = {
        id: `sandbox-offer-${this.offerCounter}`,
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
    const offer = offers.find((o) => o.id === offerId)

    if (!offer) {
      console.warn(`[mockOffers] Offer not found: ${offerId}`)
      return
    }

    // Сбрасываем предыдущий selected
    offers.forEach((o) => {
      if (o.status === 'selected') o.status = 'pending'
    })

    offer.status = 'selected'

    // Обновляем статус заявки
    const fr = getMockFreightRequests()?.get(frId)
    if (fr) {
      fr.status = 'selected'
    }

    // Если autoconfirm — сразу confirm (с минимальной задержкой для UI)
    if (this.autoConfirmOffers.has(offerId)) {
      setTimeout(() => {
        this.confirm(frId, offerId)
      }, AUTO_CONFIRM_DELAY_MS)
    }
  }

  /**
   * Отклонить оффер
   */
  reject(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)

    if (!offer) {
      console.warn(`[mockOffers] Offer not found for reject: ${offerId}`)
      return
    }

    offer.status = 'rejected'
  }

  /**
   * Отменить выбор оффера
   */
  unselect(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)

    if (!offer) {
      console.warn(`[mockOffers] Offer not found for unselect: ${offerId}`)
      return
    }

    if (offer.status !== 'selected') {
      console.warn(`[mockOffers] Cannot unselect offer with status: ${offer.status}`)
      return
    }

    offer.status = 'pending'

    // Возвращаем статус заявки на published
    const fr = getMockFreightRequests()?.get(frId)
    if (fr) {
      fr.status = 'published'
    }
  }

  /**
   * Подтвердить оффер
   */
  confirm(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)

    if (!offer) {
      console.warn(`[mockOffers] Offer not found for confirm: ${offerId}`)
      return
    }

    offer.status = 'confirmed'

    // Отклоняем все другие pending офферы
    offers.forEach((o) => {
      if (o.id !== offerId && o.status === 'pending') {
        o.status = 'rejected'
      }
    })

    // Обновляем статус заявки и копируем информацию о перевозчике
    const fr = getMockFreightRequests()?.get(frId)
    if (fr) {
      fr.status = 'confirmed'
      // Копируем carrier info из оффера
      fr.carrier_org_id = offer.carrier_org_id
      fr.carrier_org_name = offer.carrier_org_name
      fr.carrier_member_id = offer.carrier_member_id
      fr.carrier_member_name = offer.carrier_member_name
    }

    // Отправляем событие
    tutorialBus.emit('offer:confirmed', { frId, offerId })
  }

  /**
   * Отклонить выбранный оффер (перевозчик отказывается)
   */
  decline(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)

    if (!offer) {
      console.warn(`[mockOffers] Offer not found for decline: ${offerId}`)
      return
    }

    if (offer.status !== 'selected') {
      console.warn(`[mockOffers] Cannot decline offer with status: ${offer.status}`)
      return
    }

    offer.status = 'declined'

    // Возвращаем статус заявки на published
    const fr = getMockFreightRequests()?.get(frId)
    if (fr) {
      fr.status = 'published'
    }
  }

  /**
   * Отозвать оффер
   */
  withdraw(frId: string, offerId: string): void {
    const offers = this.offersByFr.get(frId) || []
    const offer = offers.find((o) => o.id === offerId)

    if (!offer) {
      console.warn(`[mockOffers] Offer not found for withdraw: ${offerId}`)
      return
    }

    offer.status = 'withdrawn'

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
    this.offerCounter = 0
  }
}

// Глобальный singleton для корректной работы при HMR
declare global {
  interface Window {
    __mockOffers?: MockOfferStore
  }
}

if (!window.__mockOffers) {
  window.__mockOffers = new MockOfferStore()
}

export const mockOffers = window.__mockOffers
