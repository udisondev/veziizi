/**
 * Composable для ожидания готовности sandbox
 *
 * Решает race condition: когда sandbox восстанавливается из localStorage,
 * initialize() сценария может не успеть создать mock данные до того,
 * как компоненты начнут делать API запросы.
 */

import { watch } from 'vue'
import { useOnboardingStore } from '@/stores/onboarding'
import { SANDBOX_READY_TIMEOUT_MS } from '@/sandbox/constants'

export function useSandboxReady() {
  const onboarding = useOnboardingStore()

  /**
   * Ожидает готовности sandbox перед выполнением запросов.
   * Если не в sandbox режиме или sandbox уже готов — возвращается сразу.
   * @param timeoutMs - максимальное время ожидания
   */
  async function waitForReady(timeoutMs = SANDBOX_READY_TIMEOUT_MS): Promise<void> {
    // Если не в sandbox режиме или уже готов — сразу возвращаем
    if (!onboarding.isSandboxMode || onboarding.sandboxReady) {
      return
    }

    // Ждём готовности с таймаутом
    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        unwatch()
        // Не отклоняем Promise, просто продолжаем — возможно sandbox не нужен
        console.warn('[useSandboxReady] Timeout waiting for sandbox ready, continuing anyway')
        resolve()
      }, timeoutMs)

      const unwatch = watch(
        () => onboarding.sandboxReady,
        (ready) => {
          if (ready) {
            clearTimeout(timeout)
            unwatch()
            resolve()
          }
        },
        { immediate: true }
      )
    })
  }

  return { waitForReady }
}
