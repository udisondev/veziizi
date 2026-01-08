/**
 * Sandbox API Interceptor
 * Перехватывает API вызовы в режиме песочницы и маршрутизирует на mock handlers
 */

import { useOnboardingStore } from '@/stores/onboarding'
import { handleSandboxRequest, type MockResponse } from './handlers'

/**
 * Обёртка над fetch для перехвата запросов в sandbox режиме
 */
export function createSandboxFetch(originalFetch: typeof fetch): typeof fetch {
  return async function sandboxFetch(
    input: RequestInfo | URL,
    init?: RequestInit
  ): Promise<Response> {
    const onboarding = useOnboardingStore()

    // Если не в sandbox режиме — используем оригинальный fetch
    if (!onboarding.isSandboxMode) {
      return originalFetch(input, init)
    }

    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url
    const method = init?.method?.toUpperCase() || 'GET'

    // Проверяем только API запросы
    if (!url.startsWith('/api/v1/')) {
      return originalFetch(input, init)
    }

    // Извлекаем путь без /api/v1
    const path = url.replace('/api/v1', '')

    // Парсим body если есть
    let body: unknown = undefined
    if (init?.body && typeof init.body === 'string') {
      try {
        body = JSON.parse(init.body)
      } catch {
        body = init.body
      }
    }

    // Пробуем обработать через mock handlers
    const mockResponse = await handleSandboxRequest(method, path, body)

    if (mockResponse) {
      return createMockResponse(mockResponse)
    }

    // Если handler не найден — пропускаем к реальному API
    // (для запросов которые не нужно мокать, например auth/me)
    console.warn(`[Sandbox] No handler for ${method} ${path}, passing through`)
    return originalFetch(input, init)
  }
}

/**
 * Создаёт Response объект из mock данных
 */
function createMockResponse(mock: MockResponse): Response {
  const { status = 200, data, headers = {} } = mock

  // Для 204 No Content
  if (status === 204) {
    return new Response(null, {
      status: 204,
      statusText: 'No Content',
      headers: new Headers(headers),
    })
  }

  // Для ошибок
  if (status >= 400) {
    return new Response(
      JSON.stringify({ error: data?.error || 'Error', error_code: data?.error_code }),
      {
        status,
        statusText: status === 404 ? 'Not Found' : 'Error',
        headers: new Headers({
          'Content-Type': 'application/json',
          ...headers,
        }),
      }
    )
  }

  // Успешный ответ
  return new Response(JSON.stringify(data), {
    status,
    statusText: 'OK',
    headers: new Headers({
      'Content-Type': 'application/json',
      ...headers,
    }),
  })
}

/**
 * Инициализировать перехват fetch
 * Вызывать один раз при старте приложения
 */
let isInitialized = false

export function initSandboxInterceptor(): void {
  if (isInitialized) return

  const originalFetch = window.fetch.bind(window)
  window.fetch = createSandboxFetch(originalFetch)

  isInitialized = true
}
