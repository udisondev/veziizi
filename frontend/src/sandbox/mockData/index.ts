/**
 * Mock Data Index
 * Экспорт всех mock stores
 */

export * from './generators'
export { mockFreightRequests } from './freightRequests'
export { mockOffers } from './offers'
export { mockBot } from './bot'
export { mockMembers } from './members'
export { mockNotifications } from './notifications'

// Импорты для resetAllMockData
import { mockFreightRequests } from './freightRequests'
import { mockOffers } from './offers'
import { mockBot } from './bot'
import { mockMembers } from './members'
import { mockNotifications } from './notifications'

/**
 * Сбросить все mock stores
 */
export function resetAllMockData(): void {
  mockFreightRequests.clear()
  mockOffers.clear()
  mockBot.reset()
  mockMembers.clear()
  mockNotifications.clear()
}
