// Event type labels in Russian
export const eventTypeLabels: Record<string, string> = {
  // Organization events
  'organization.created': 'Организация создана',
  'organization.approved': 'Организация одобрена',
  'organization.rejected': 'Организация отклонена',
  'organization.suspended': 'Организация заблокирована',
  'organization.updated': 'Данные организации обновлены',

  // Member events
  'member.added': 'Добавлен сотрудник',
  'member.removed': 'Удалён сотрудник',
  'member.role_changed': 'Изменена роль сотрудника',
  'member.blocked': 'Сотрудник заблокирован',
  'member.unblocked': 'Сотрудник разблокирован',

  // Invitation events
  'invitation.created': 'Создано приглашение',
  'invitation.accepted': 'Приглашение принято',
  'invitation.expired': 'Приглашение истекло',
  'invitation.cancelled': 'Приглашение отменено',

  // FreightRequest events
  'freight_request.created': 'Заявка создана',
  'freight_request.updated': 'Заявка обновлена',
  'freight_request.reassigned': 'Изменён ответственный',
  'freight_request.cancelled': 'Заявка отменена',
  'freight_request.expired': 'Заявка истекла',

  // Offer events
  'offer.made': 'Получено предложение',
  'offer.withdrawn': 'Предложение отозвано',
  'offer.selected': 'Предложение выбрано',
  'offer.rejected': 'Предложение отклонено',
  'offer.confirmed': 'Предложение подтверждено',
  'offer.declined': 'Отказ от предложения',

  // Order events
  'order.created': 'Заказ создан',
  'order.cancelled': 'Заказ отменён',
  'order.customer_completed': 'Заказчик завершил',
  'order.carrier_completed': 'Перевозчик завершил',
  'order.completed': 'Заказ завершён',
  'order.message_sent': 'Отправлено сообщение',
  'order.document_attached': 'Прикреплён документ',
  'order.document_removed': 'Удалён документ',
  'order.review_left': 'Оставлен отзыв',
}

// Role labels
const roleLabels: Record<string, string> = {
  owner: 'Владелец',
  administrator: 'Администратор',
  employee: 'Сотрудник',
}

// Format event details based on event type
export function formatEventDetails(eventType: string, data: Record<string, unknown>): string {
  switch (eventType) {
    // Organization events
    case 'organization.created':
      return `Название: ${data.name || '—'}`

    case 'organization.rejected':
    case 'organization.suspended':
      return data.reason ? `Причина: ${data.reason}` : ''

    case 'organization.updated': {
      const changes: string[] = []
      if (data.name) changes.push('название')
      if (data.phone) changes.push('телефон')
      if (data.email) changes.push('email')
      if (data.address) changes.push('адрес')
      return changes.length > 0 ? `Изменено: ${changes.join(', ')}` : ''
    }

    // Member events
    case 'member.added':
      return `Email: ${data.email || '—'}, Роль: ${roleLabels[data.role as string] || data.role}`

    case 'member.role_changed': {
      const oldRole = roleLabels[data.old_role as string] || data.old_role
      const newRole = roleLabels[data.new_role as string] || data.new_role
      return `${oldRole} → ${newRole}`
    }

    case 'member.blocked':
      return data.reason ? `Причина: ${data.reason}` : ''

    // Invitation events
    case 'invitation.created':
      return `Email: ${data.email || '—'}, Роль: ${roleLabels[data.role as string] || data.role}`

    // FreightRequest events
    case 'freight_request.created':
      return data.request_number ? `Номер: ${data.request_number}` : ''

    case 'freight_request.updated': {
      const frChanges: string[] = []
      if (data.route) frChanges.push('маршрут')
      if (data.cargo) frChanges.push('груз')
      if (data.vehicle_requirements) frChanges.push('требования к ТС')
      if (data.payment) frChanges.push('оплата')
      if (data.comment !== undefined) frChanges.push('комментарий')
      return frChanges.length > 0 ? `Изменено: ${frChanges.join(', ')}` : ''
    }

    case 'freight_request.cancelled':
    case 'offer.withdrawn':
    case 'offer.rejected':
    case 'offer.declined':
      return data.reason ? `Причина: ${data.reason}` : ''

    // Offer events
    case 'offer.made': {
      const price = data.price as Record<string, unknown> | undefined
      if (price?.amount) {
        const amount = (price.amount as number) / 100
        const currency = price.currency || 'RUB'
        return `Цена: ${amount.toLocaleString('ru-RU')} ${currency}`
      }
      return ''
    }

    // Order events
    case 'order.created':
      return data.order_number ? `Номер: ${data.order_number}` : ''

    case 'order.cancelled':
      return data.reason ? `Причина: ${data.reason}` : ''

    case 'order.message_sent':
      return data.content ? `"${(data.content as string).substring(0, 50)}${(data.content as string).length > 50 ? '...' : ''}"` : ''

    case 'order.document_attached':
      return data.name ? `Файл: ${data.name}` : ''

    case 'order.review_left': {
      const rating = data.rating as number | undefined
      if (rating) {
        const stars = '★'.repeat(rating) + '☆'.repeat(5 - rating)
        return `Оценка: ${stars}`
      }
      return ''
    }

    default:
      return ''
  }
}

// Check if event is automatic (no actor)
export function isAutomaticEvent(eventType: string): boolean {
  const automaticEvents = [
    'organization.created',
    'freight_request.expired',
    'invitation.expired',
    'invitation.accepted',
    'order.created',
    'order.completed',
    'member.removed',
  ]
  return automaticEvents.includes(eventType)
}
