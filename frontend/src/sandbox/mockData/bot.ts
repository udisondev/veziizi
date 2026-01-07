/**
 * Mock Bot
 * Симуляция ответов контрагента в чате заказа
 */

import { mockOrders } from './orders'
import { generateId } from './generators'

// Заготовленные реплики бота
const BOT_REPLIES = [
  'Добрый день! Готов к работе.',
  'Подтверждаю, груз принят.',
  'Ориентировочно буду на месте через 2 часа.',
  'Документы подготовлены, отправлю после разгрузки.',
  'Всё в порядке, движемся по графику.',
  'Спасибо за заказ! Был рад сотрудничеству.',
  'Принял. Выезжаю.',
  'Загрузка завершена, начинаю движение.',
  'Есть вопрос по адресу разгрузки, можете уточнить?',
  'Прибыл на место, ожидаю.',
]

// Контекстные ответы (на определённые ключевые слова)
const CONTEXTUAL_REPLIES: Record<string, string[]> = {
  привет: ['Здравствуйте!', 'Добрый день!', 'Приветствую!'],
  когда: ['Ориентировочно через 2-3 часа.', 'Скоро буду, не переживайте.'],
  адрес: ['Еду по навигатору, адрес верный.', 'Да, адрес понятен.'],
  документ: ['Все документы при себе.', 'Документы подготовлю к выгрузке.'],
  спасибо: ['Вам спасибо!', 'Обращайтесь!', 'Всегда рад помочь.'],
  готов: ['Отлично, начинаем!', 'Принял, приступаю.'],
}

class MockBot {
  private replyIndex = 0
  private pendingReplies: Map<string, NodeJS.Timeout> = new Map()

  /**
   * Запланировать ответ бота
   */
  async scheduleReply(orderId: string, delayMs: number = 1500): Promise<void> {
    // Отменяем предыдущий pending reply если есть
    const existing = this.pendingReplies.get(orderId)
    if (existing) {
      clearTimeout(existing)
    }

    const timeout = setTimeout(() => {
      this.sendReply(orderId)
      this.pendingReplies.delete(orderId)
    }, delayMs)

    this.pendingReplies.set(orderId, timeout)
  }

  /**
   * Отправить ответ
   */
  private sendReply(orderId: string, customMessage?: string): void {
    const order = mockOrders.get(orderId)
    if (!order) return

    const message = customMessage || this.getNextReply(order)

    mockOrders.addMessage(orderId, {
      sender_org_id: order.carrier_org_id,
      sender_member_id: order.carrier_member_id,
      content: message,
    })
  }

  /**
   * Получить следующую реплику
   */
  private getNextReply(order: any): string {
    // Проверяем последнее сообщение пользователя на ключевые слова
    const lastUserMessage = order.messages
      .filter((m: any) => m.sender_org_id === 'sandbox-org-1')
      .pop()

    if (lastUserMessage) {
      const content = lastUserMessage.content.toLowerCase()

      for (const [keyword, replies] of Object.entries(CONTEXTUAL_REPLIES)) {
        if (content.includes(keyword)) {
          return replies[Math.floor(Math.random() * replies.length)]
        }
      }
    }

    // Если контекст не найден — берём следующую по порядку
    const reply = BOT_REPLIES[this.replyIndex % BOT_REPLIES.length]
    this.replyIndex++
    return reply
  }

  /**
   * Сбросить состояние бота
   */
  reset(): void {
    this.replyIndex = 0
    for (const timeout of this.pendingReplies.values()) {
      clearTimeout(timeout)
    }
    this.pendingReplies.clear()
  }
}

// Глобальный singleton для корректной работы при HMR
declare global {
  interface Window {
    __mockBot?: MockBot
  }
}

if (!window.__mockBot) {
  window.__mockBot = new MockBot()
}

export const mockBot = window.__mockBot
