/**
 * Mock Handlers Index
 * Регистрация и маршрутизация mock handlers
 */

import { freightRequestsHandlers } from './freightRequests'
import { offersHandlers } from './offers'
import { ordersHandlers } from './orders'
import { membersHandlers } from './members'
import { notificationsHandlers } from './notifications'

export interface MockResponse {
  status?: number
  data?: unknown
  headers?: Record<string, string>
}

export type MockHandler = (
  params: Record<string, string>,
  body?: unknown,
  query?: URLSearchParams
) => Promise<MockResponse> | MockResponse

interface RoutePattern {
  pattern: RegExp
  paramNames: string[]
  handler: MockHandler
}

interface RouteMap {
  GET: RoutePattern[]
  POST: RoutePattern[]
  PUT: RoutePattern[]
  PATCH: RoutePattern[]
  DELETE: RoutePattern[]
}

// Глобальные singletons для избежания проблем с HMR
declare global {
  interface Window {
    __sandboxRoutes?: RouteMap
    __sandboxHandlersRegistered?: boolean
  }
}

if (!window.__sandboxRoutes) {
  window.__sandboxRoutes = {
    GET: [],
    POST: [],
    PUT: [],
    PATCH: [],
    DELETE: [],
  }
}

const routes = window.__sandboxRoutes

/**
 * Зарегистрировать handler
 */
export function registerHandler(
  method: keyof RouteMap,
  path: string,
  handler: MockHandler
): void {
  const { pattern, paramNames } = pathToRegex(path)
  routes[method].push({ pattern, paramNames, handler })
}

/**
 * Преобразовать путь в regex с named parameters
 * Например: /freight-requests/:id/offers/:offerId
 */
function pathToRegex(path: string): { pattern: RegExp; paramNames: string[] } {
  const paramNames: string[] = []

  // ВАЖНО: сначала извлекаем параметры, потом escape слешей!
  // Иначе :frId/ станет :frId\/ и regex захватит лишний backslash
  const regexStr = path
    .replace(/:([^/]+)/g, (_, name) => {
      paramNames.push(name)
      return '([^/]+)'
    })
    .replace(/\//g, '\\/')

  return {
    pattern: new RegExp(`^${regexStr}(?:\\?.*)?$`),
    paramNames,
  }
}

/**
 * Найти и выполнить handler для запроса
 */
export async function handleSandboxRequest(
  method: string,
  path: string,
  body?: unknown
): Promise<MockResponse | null> {
  const routeList = routes[method as keyof RouteMap]
  if (!routeList) return null

  // Разделяем path и query string
  const [pathPart, queryString] = path.split('?')
  const query = queryString ? new URLSearchParams(queryString) : new URLSearchParams()

  for (const route of routeList) {
    const match = pathPart.match(route.pattern)
    if (match) {
      // Извлекаем параметры из URL
      const params: Record<string, string> = {}
      route.paramNames.forEach((name, index) => {
        params[name] = match[index + 1]
      })

      try {
        return await route.handler(params, body, query)
      } catch (error) {
        console.error('[Sandbox] Handler error:', error)
        return {
          status: 500,
          data: { error: 'Internal sandbox error' },
        }
      }
    }
  }

  return null
}

// Регистрируем handlers только один раз
if (!window.__sandboxHandlersRegistered) {
  freightRequestsHandlers()
  offersHandlers()
  ordersHandlers()
  membersHandlers()
  notificationsHandlers()
  window.__sandboxHandlersRegistered = true
  console.log('[Sandbox] Mock handlers registered')
}
