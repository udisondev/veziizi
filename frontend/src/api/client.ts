import { ApiError, type ApiErrorResponse } from '@/api/errors'

// Callback для обработки блокировки аккаунта
// Устанавливается в main.ts для обновления статуса в auth store
let accountBlockedHandler: (() => Promise<void>) | null = null

export function setAccountBlockedHandler(handler: () => Promise<void>): void {
  accountBlockedHandler = handler
}

export class ApiClient {
  private baseUrl: string

  constructor(baseUrl = '/api/v1') {
    this.baseUrl = baseUrl
  }

  private async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest', // SEC-005: CSRF protection
        ...options.headers,
      },
      credentials: 'include',
    })

    if (!response.ok) {
      const errorData: ApiErrorResponse = await response.json().catch(() => ({
        error: `HTTP ${response.status}`,
      }))
      const error = new ApiError(
        response.status,
        errorData.error,
        errorData.error_code,
        errorData.details
      )

      // Если аккаунт заблокирован - обновляем статус в auth store
      if (error.isAccountBlocked() && accountBlockedHandler) {
        // Вызываем handler асинхронно, не блокируя основной поток
        accountBlockedHandler().catch(() => {
          // Игнорируем ошибки при обновлении статуса
        })
      }

      throw error
    }

    if (response.status === 204) {
      return undefined as T
    }

    return response.json()
  }

  get<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: 'GET' })
  }

  post<T>(
    path: string,
    body?: unknown,
    options?: { headers?: Record<string, string> }
  ): Promise<T> {
    return this.request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
      headers: options?.headers,
    })
  }

  put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'PATCH',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  delete<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'DELETE',
      body: body ? JSON.stringify(body) : undefined,
    })
  }
}

export const api = new ApiClient()
