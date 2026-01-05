/**
 * Mock Data Index
 * Экспорт всех mock stores
 */

export * from './generators'
export { mockFreightRequests } from './freightRequests'
export { mockOffers } from './offers'
export { mockOrders } from './orders'
export { mockBot } from './bot'
export { mockMembers } from './members'

/**
 * Сбросить все mock stores
 */
export function resetAllMockData(): void {
  const { mockFreightRequests } = require('./freightRequests')
  const { mockOffers } = require('./offers')
  const { mockOrders } = require('./orders')
  const { mockBot } = require('./bot')
  const { mockMembers } = require('./members')

  mockFreightRequests.clear()
  mockOffers.clear()
  mockOrders.clear()
  mockBot.reset()
  mockMembers.clear()
}
