/**
 * Composable для асинхронной загрузки данных
 * Устраняет дублирование паттерна loading/error/data во всех views
 */

import { ref, readonly, onMounted, type Ref, type DeepReadonly } from 'vue'
import { getErrorMessage } from '@/api/errors'
import { logger } from '@/utils/logger'

export interface UseAsyncDataOptions<T> {
  /**
   * Загрузить данные автоматически при монтировании
   * @default true
   */
  immediate?: boolean

  /**
   * Начальное значение данных
   */
  initialValue?: T

  /**
   * Callback при успешной загрузке
   */
  onSuccess?: (data: T) => void

  /**
   * Callback при ошибке
   */
  onError?: (error: Error) => void
}

export interface UseAsyncDataReturn<T> {
  /**
   * Загруженные данные
   */
  data: DeepReadonly<Ref<T | null>>

  /**
   * Состояние загрузки
   */
  isLoading: DeepReadonly<Ref<boolean>>

  /**
   * Сообщение об ошибке
   */
  error: DeepReadonly<Ref<string | null>>

  /**
   * Выполнить загрузку данных
   */
  execute: () => Promise<void>

  /**
   * Обновить данные (alias для execute)
   */
  refresh: () => Promise<void>

  /**
   * Сбросить состояние
   */
  reset: () => void
}

/**
 * Composable для загрузки данных с состоянием loading/error
 *
 * @example
 * ```ts
 * const { data: items, isLoading, error, execute } = useAsyncData(
 *   () => freightRequestsApi.list(params)
 * )
 *
 * // В шаблоне
 * <LoadingSpinner v-if="isLoading" />
 * <ErrorBanner v-else-if="error" :message="error" @retry="execute" />
 * <div v-else>{{ items }}</div>
 * ```
 */
export function useAsyncData<T>(
  fetcher: () => Promise<T>,
  options: UseAsyncDataOptions<T> = {}
): UseAsyncDataReturn<T> {
  const { immediate = true, initialValue = null, onSuccess, onError } = options

  const data = ref<T | null>(initialValue) as Ref<T | null>
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function execute(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const result = await fetcher()
      data.value = result
      onSuccess?.(result)
    } catch (e) {
      const message = getErrorMessage(e)
      error.value = message
      logger.error('useAsyncData error', e)
      onError?.(e instanceof Error ? e : new Error(message))
    } finally {
      isLoading.value = false
    }
  }

  function reset(): void {
    data.value = initialValue
    isLoading.value = false
    error.value = null
  }

  if (immediate) {
    onMounted(() => execute())
  }

  return {
    data: readonly(data),
    isLoading: readonly(isLoading),
    error: readonly(error),
    execute,
    refresh: execute,
    reset,
  }
}

/**
 * Composable для загрузки списка данных
 * Удобная обёртка для массивов
 */
export function useAsyncList<T>(
  fetcher: () => Promise<T[]>,
  options: UseAsyncDataOptions<T[]> = {}
): UseAsyncDataReturn<T[]> & {
  isEmpty: DeepReadonly<Ref<boolean>>
} {
  // Передаём immediate: false чтобы избежать двойного вызова
  const result = useAsyncData(fetcher, {
    ...options,
    immediate: false,
    initialValue: options.initialValue ?? [],
  })

  const isEmpty = ref(false)

  // Переопределяем execute чтобы обновлять isEmpty
  const originalExecute = result.execute
  const execute = async () => {
    await originalExecute()
    isEmpty.value = (result.data.value?.length ?? 0) === 0
  }

  if (options.immediate !== false) {
    onMounted(() => execute())
  }

  return {
    ...result,
    execute,
    refresh: execute,
    isEmpty: readonly(isEmpty),
  }
}

/**
 * Composable для выполнения действия (мутации)
 * Используется для POST/PUT/DELETE операций
 *
 * @example
 * ```ts
 * const { execute: cancelRequest, isLoading } = useAsyncAction(
 *   (id: string) => freightRequestsApi.cancel(id)
 * )
 *
 * await cancelRequest('request-id')
 * ```
 */
export function useAsyncAction<TParams extends unknown[], TResult = void>(
  action: (...params: TParams) => Promise<TResult>,
  options: {
    onSuccess?: (result: TResult) => void
    onError?: (error: Error) => void
  } = {}
): {
  execute: (...params: TParams) => Promise<TResult | undefined>
  isLoading: DeepReadonly<Ref<boolean>>
  error: DeepReadonly<Ref<string | null>>
  reset: () => void
} {
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function execute(...params: TParams): Promise<TResult | undefined> {
    isLoading.value = true
    error.value = null

    try {
      const result = await action(...params)
      options.onSuccess?.(result)
      return result
    } catch (e) {
      const message = getErrorMessage(e)
      error.value = message
      logger.error('useAsyncAction error', e)
      options.onError?.(e instanceof Error ? e : new Error(message))
      return undefined
    } finally {
      isLoading.value = false
    }
  }

  function reset(): void {
    isLoading.value = false
    error.value = null
  }

  return {
    execute,
    isLoading: readonly(isLoading),
    error: readonly(error),
    reset,
  }
}
