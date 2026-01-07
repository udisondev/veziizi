/**
 * Debounce utility for preventing excessive function calls
 */

/**
 * Creates a debounced version of a function
 *
 * @param fn - Function to debounce
 * @param delay - Delay in milliseconds
 * @returns Debounced function with cancel method
 *
 * @example
 * ```ts
 * const debouncedSearch = debounce((query: string) => {
 *   api.search(query)
 * }, 300)
 *
 * // Call multiple times - only last call executes after 300ms
 * debouncedSearch('a')
 * debouncedSearch('ab')
 * debouncedSearch('abc') // Only this executes
 *
 * // Cancel pending call
 * debouncedSearch.cancel()
 * ```
 */
export function debounce<T extends (...args: Parameters<T>) => void>(
  fn: T,
  delay: number
): T & { cancel: () => void } {
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  const debounced = ((...args: Parameters<T>) => {
    if (timeoutId) {
      clearTimeout(timeoutId)
    }
    timeoutId = setTimeout(() => {
      fn(...args)
      timeoutId = null
    }, delay)
  }) as T & { cancel: () => void }

  debounced.cancel = () => {
    if (timeoutId) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
  }

  return debounced
}

/**
 * Default debounce delay for filter inputs (ms)
 */
export const FILTER_DEBOUNCE_DELAY = 300

/**
 * Default debounce delay for search inputs (ms)
 */
export const SEARCH_DEBOUNCE_DELAY = 300

/**
 * Creates a throttled version of a function
 * Function executes immediately, then at most once per `wait` ms
 *
 * @param fn - Function to throttle
 * @param wait - Minimum time between calls in milliseconds
 * @returns Throttled function
 *
 * @example
 * ```ts
 * const throttledScroll = throttle(() => {
 *   updatePosition()
 * }, 50)
 *
 * window.addEventListener('scroll', throttledScroll)
 * ```
 */
export function throttle<T extends (...args: Parameters<T>) => void>(
  fn: T,
  wait: number
): T {
  let lastTime = 0
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  return ((...args: Parameters<T>) => {
    const now = Date.now()
    const remaining = wait - (now - lastTime)

    if (remaining <= 0) {
      // Enough time has passed, execute immediately
      if (timeoutId) {
        clearTimeout(timeoutId)
        timeoutId = null
      }
      lastTime = now
      fn(...args)
    } else if (!timeoutId) {
      // Schedule execution for remaining time
      timeoutId = setTimeout(() => {
        lastTime = Date.now()
        timeoutId = null
        fn(...args)
      }, remaining)
    }
  }) as T
}

/**
 * Default throttle delay for scroll handlers (ms)
 */
export const SCROLL_THROTTLE_DELAY = 50
