export type RoutePointRole = 'any' | 'origin' | 'destination'

export interface RoutePointData {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
  role?: RoutePointRole
}

/**
 * normalizeRoleByPosition — единственный источник правды для правил
 * совместимости роли точки с её позицией в списке:
 *   - одна точка        → любая роль допустима;
 *   - первая (≥2 точки) → 'origin' или 'any';
 *   - последняя         → 'destination' или 'any';
 *   - средняя           → 'any'.
 *
 * Используется в двух местах одновременно:
 *   1. buildRouteParams (stores/freightFilters) — последний рубеж перед
 *      отправкой query на бэк;
 *   2. SubscriptionRoutePointCard watcher — реактивная нормализация при
 *      drag-and-drop, чтобы UI не показывал «destination» на первой точке.
 *
 * Если правила меняются — правишь ЗДЕСЬ и больше нигде.
 */
export function normalizeRoleByPosition(
  role: RoutePointRole | undefined,
  index: number,
  total: number,
): RoutePointRole {
  const r = role ?? 'any'
  if (total <= 1) return r
  if (index === 0) return r === 'destination' ? 'any' : r
  if (index === total - 1) return r === 'origin' ? 'any' : r
  return 'any'
}
