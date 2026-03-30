/**
 * Composable для infinite scroll с IntersectionObserver
 */

import { ref, onMounted, onUnmounted, watch, type Ref } from 'vue'

export interface UseInfiniteScrollOptions {
  /**
   * Расстояние от низа viewport до trigger (px)
   * @default 300
   */
  threshold?: number

  /**
   * Включён ли infinite scroll
   * @default true
   */
  enabled?: Ref<boolean>
}

export interface UseInfiniteScrollReturn {
  /**
   * Ref для sentinel элемента (добавить в template)
   */
  sentinelRef: Ref<HTMLElement | null>

  /**
   * Идёт ли загрузка следующей страницы
   */
  isLoadingMore: Ref<boolean>

  /**
   * Пересоздать observer (вызвать при сбросе списка)
   */
  reset: () => void
}

export function useInfiniteScroll(
  loadMore: () => Promise<void>,
  options: UseInfiniteScrollOptions = {}
): UseInfiniteScrollReturn {
  const { threshold = 300 } = options
  const enabled = options.enabled ?? ref(true)

  const sentinelRef = ref<HTMLElement | null>(null)
  const isLoadingMore = ref(false)
  let observer: IntersectionObserver | null = null

  async function handleIntersect(entries: IntersectionObserverEntry[]) {
    const entry = entries[0]
    if (entry?.isIntersecting && !isLoadingMore.value && enabled.value) {
      isLoadingMore.value = true
      try {
        await loadMore()
      } finally {
        isLoadingMore.value = false
      }
    }
  }

  function setupObserver() {
    cleanup()
    if (!sentinelRef.value) return

    observer = new IntersectionObserver(handleIntersect, {
      root: null,
      rootMargin: `${threshold}px`,
      threshold: 0,
    })
    observer.observe(sentinelRef.value)
  }

  function cleanup() {
    if (observer) {
      observer.disconnect()
      observer = null
    }
  }

  // Watch для перезапуска при изменении sentinel ref
  watch(sentinelRef, (newVal) => {
    if (newVal) {
      setupObserver()
    }
  })

  // Watch для enabled — пересоздаём observer при изменении
  watch(enabled, () => {
    if (sentinelRef.value) {
      setupObserver()
    }
  })

  onMounted(() => {
    // nextTick для гарантии что ref установлен
    setTimeout(setupObserver, 0)
  })

  onUnmounted(cleanup)

  return {
    sentinelRef,
    isLoadingMore,
    reset: setupObserver,
  }
}
