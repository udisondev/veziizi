/**
 * Mock Bot
 * Заглушка (функционал удалён вместе с Order)
 */

class MockBot {
  reset(): void {
    // noop
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
