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
