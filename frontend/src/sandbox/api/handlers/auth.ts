/**
 * Mock Handlers for Authentication
 * Password reset flow for sandbox mode
 */

import { registerHandler } from './index'

// Симуляция хранилища токенов сброса пароля
const passwordResetTokens = new Map<string, { email: string; expiresAt: Date }>()

// Валидный тестовый токен для демонстрации
const TEST_RESET_TOKEN = 'test-reset-token-123'

export function authHandlers(): void {
  // POST /auth/forgot-password
  registerHandler('POST', '/auth/forgot-password', (_params, body) => {
    const { email } = body as { email: string }

    if (!email) {
      return {
        status: 400,
        data: { error: 'Email обязателен' },
      }
    }

    // Генерируем токен (в sandbox всегда успех, email "отправлен")
    const token = `reset-${Date.now()}-${Math.random().toString(36).slice(2)}`
    const expiresAt = new Date(Date.now() + 60 * 60 * 1000) // 1 час

    passwordResetTokens.set(token, { email, expiresAt })

    // Также добавляем тестовый токен для удобства демо
    passwordResetTokens.set(TEST_RESET_TOKEN, { email, expiresAt })

    console.log('[Sandbox] Password reset requested for:', email)
    console.log('[Sandbox] Reset token:', token)
    console.log('[Sandbox] Test token available:', TEST_RESET_TOKEN)

    // Всегда возвращаем успех (не раскрываем существует ли email)
    return {
      status: 204,
      data: null,
    }
  })

  // POST /auth/reset-password
  registerHandler('POST', '/auth/reset-password', (_params, body) => {
    const { token, password } = body as { token: string; password: string }

    if (!token) {
      return {
        status: 400,
        data: { error: 'Токен обязателен' },
      }
    }

    if (!password) {
      return {
        status: 400,
        data: { error: 'Пароль обязателен' },
      }
    }

    if (password.length < 8) {
      return {
        status: 400,
        data: { error: 'Пароль должен быть не менее 8 символов' },
      }
    }

    // Проверяем токен
    const tokenData = passwordResetTokens.get(token)

    if (!tokenData) {
      return {
        status: 400,
        data: { error: 'Недействительная или истёкшая ссылка для сброса пароля' },
      }
    }

    if (tokenData.expiresAt < new Date()) {
      passwordResetTokens.delete(token)
      return {
        status: 400,
        data: { error: 'Ссылка для сброса пароля истекла' },
      }
    }

    // Успешный сброс
    passwordResetTokens.delete(token)
    console.log('[Sandbox] Password reset successful for:', tokenData.email)

    return {
      status: 204,
      data: null,
    }
  })
}
