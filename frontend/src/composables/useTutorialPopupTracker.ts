/**
 * Tutorial Popup Tracker
 * Отслеживает открытые popup/dropdown элементы для динамического
 * расширения области подсветки tutorial и позиционирования tooltip
 */

import { ref, type Ref } from 'vue'
import { TUTORIAL_POPUP_MAX_DISTANCE } from '@/sandbox/constants'

// Универсальные селекторы для определения открытых popup
const POPUP_SELECTORS = [
  // reka-ui Select/Popover/Dialog (используют data-state)
  '[data-state="open"][role="listbox"]',
  '[data-state="open"][role="dialog"]',
  '[data-state="open"][role="menu"]',

  // Reka-ui / Radix portals
  '[data-reka-popper-content-wrapper]',
  '[data-radix-popper-content-wrapper]',

  // Универсальный атрибут для кастомных dropdown
  '[data-tutorial-popup]',
]

export type PopupDirection = 'up' | 'down' | 'left' | 'right'

export interface PopupInfo {
  element: Element
  rect: DOMRect
  direction: PopupDirection
}

export interface CombinedRect {
  top: number
  left: number
  width: number
  height: number
  right: number
  bottom: number
}

export function useTutorialPopupTracker() {
  const activePopups: Ref<PopupInfo[]> = ref([])
  const currentPopupDirection: Ref<PopupDirection | null> = ref(null)

  /**
   * Определяет направление popup относительно target
   */
  function getPopupDirection(targetRect: DOMRect, popupRect: DOMRect): PopupDirection {
    const targetCenterY = targetRect.top + targetRect.height / 2
    const targetCenterX = targetRect.left + targetRect.width / 2
    const popupCenterY = popupRect.top + popupRect.height / 2
    const popupCenterX = popupRect.left + popupRect.width / 2

    const deltaY = popupCenterY - targetCenterY
    const deltaX = popupCenterX - targetCenterX

    // Определяем основное направление
    if (Math.abs(deltaY) > Math.abs(deltaX)) {
      return deltaY < 0 ? 'up' : 'down'
    } else {
      return deltaX < 0 ? 'left' : 'right'
    }
  }

  /**
   * Находит все открытые popup элементы рядом с target
   */
  function findPopupsNear(targetRect: DOMRect): PopupInfo[] {
    const popups: PopupInfo[] = []
    const maxDistance = TUTORIAL_POPUP_MAX_DISTANCE

    for (const selector of POPUP_SELECTORS) {
      try {
        const elements = document.querySelectorAll(selector)

        for (const el of elements) {
          // Проверяем что элемент видим
          const style = window.getComputedStyle(el)
          if (style.display === 'none' || style.visibility === 'hidden') {
            continue
          }

          const popupRect = el.getBoundingClientRect()

          // Проверяем что popup имеет размеры
          if (popupRect.width === 0 || popupRect.height === 0) {
            continue
          }

          // Проверяем что popup "связан" с target
          // (перекрывается или находится рядом)
          const isNear =
            // Горизонтальное пересечение или близость
            (popupRect.left < targetRect.right + maxDistance &&
             popupRect.right > targetRect.left - maxDistance) &&
            // Вертикальное пересечение или близость
            (popupRect.top < targetRect.bottom + maxDistance &&
             popupRect.bottom > targetRect.top - maxDistance)

          if (isNear) {
            popups.push({
              element: el,
              rect: popupRect,
              direction: getPopupDirection(targetRect, popupRect),
            })
          }
        }
      } catch (e) {
        // Некоторые селекторы могут быть невалидными в определённых браузерах
        if (import.meta.env.DEV) {
          console.warn(`[PopupTracker] Invalid selector: ${selector}`, e)
        }
      }
    }

    return popups
  }

  /**
   * Вычисляет объединённый rect (target + все popup)
   */
  function getCombinedRect(targetRect: DOMRect, popups: PopupInfo[]): CombinedRect {
    let minTop = targetRect.top
    let minLeft = targetRect.left
    let maxBottom = targetRect.bottom
    let maxRight = targetRect.right

    for (const popup of popups) {
      minTop = Math.min(minTop, popup.rect.top)
      minLeft = Math.min(minLeft, popup.rect.left)
      maxBottom = Math.max(maxBottom, popup.rect.bottom)
      maxRight = Math.max(maxRight, popup.rect.right)
    }

    return {
      top: minTop,
      left: minLeft,
      width: maxRight - minLeft,
      height: maxBottom - minTop,
      right: maxRight,
      bottom: maxBottom,
    }
  }

  /**
   * Получает оптимальную позицию tooltip, учитывая popup и preferredPosition
   *
   * Логика:
   * - Если preferredPosition НЕ конфликтует с popup, оставляем её
   * - Если конфликтует (tooltip и popup в одном месте), выбираем противоположную
   *
   * Конфликты:
   * - popup down + tooltip bottom → конфликт
   * - popup up + tooltip top → конфликт
   * - popup left + tooltip left → конфликт
   * - popup right + tooltip right → конфликт
   */
  function getOptimalTooltipPosition(
    popupDirection: PopupDirection | null,
    preferredPosition: 'top' | 'bottom' | 'left' | 'right' = 'bottom'
  ): 'top' | 'bottom' | 'left' | 'right' {
    if (!popupDirection) {
      return preferredPosition
    }

    // Маппинг: какая позиция tooltip конфликтует с каким направлением popup
    const conflictMap: Record<PopupDirection, 'top' | 'bottom' | 'left' | 'right'> = {
      'down': 'bottom',  // popup вниз конфликтует с tooltip снизу
      'up': 'top',       // popup вверх конфликтует с tooltip сверху
      'left': 'left',    // popup влево конфликтует с tooltip слева
      'right': 'right',  // popup вправо конфликтует с tooltip справа
    }

    const conflictingPosition = conflictMap[popupDirection]

    // Если preferredPosition не конфликтует, оставляем её
    if (preferredPosition !== conflictingPosition) {
      return preferredPosition
    }

    // Иначе возвращаем противоположную
    const oppositePositions: Record<PopupDirection, 'top' | 'bottom' | 'left' | 'right'> = {
      'up': 'bottom',
      'down': 'top',
      'left': 'right',
      'right': 'left',
    }

    return oppositePositions[popupDirection]
  }

  /**
   * Основной метод: анализирует target и возвращает информацию о popup
   */
  function analyzeTarget(targetRect: DOMRect): {
    popups: PopupInfo[]
    combinedRect: CombinedRect
    hasPopups: boolean
    primaryDirection: PopupDirection | null
  } {
    const popups = findPopupsNear(targetRect)
    activePopups.value = popups

    const hasPopups = popups.length > 0
    const primaryDirection = hasPopups ? popups[0].direction : null
    currentPopupDirection.value = primaryDirection

    const combinedRect = hasPopups
      ? getCombinedRect(targetRect, popups)
      : {
          top: targetRect.top,
          left: targetRect.left,
          width: targetRect.width,
          height: targetRect.height,
          right: targetRect.right,
          bottom: targetRect.bottom,
        }

    return {
      popups,
      combinedRect,
      hasPopups,
      primaryDirection,
    }
  }

  /**
   * Сбрасывает состояние трекера
   */
  function reset() {
    activePopups.value = []
    currentPopupDirection.value = null
  }

  return {
    // State
    activePopups,
    currentPopupDirection,

    // Methods
    findPopupsNear,
    getCombinedRect,
    getPopupDirection,
    getOptimalTooltipPosition,
    analyzeTarget,
    reset,
  }
}

// ============================================
// Shared MutationObserver для popup изменений
// ============================================

// Singleton MutationObserver для отслеживания изменений popup
let sharedObserver: MutationObserver | null = null
let popupChangeDebounceTimer: ReturnType<typeof setTimeout> | null = null
const popupChangeListeners = new Set<() => void>()

/**
 * Подписаться на изменения popup (появление/исчезновение)
 * Возвращает функцию отписки
 *
 * @example
 * ```ts
 * onMounted(() => {
 *   const unsubscribe = onPopupChange(() => updatePosition())
 *   onUnmounted(() => unsubscribe())
 * })
 * ```
 */
export function onPopupChange(callback: () => void): () => void {
  popupChangeListeners.add(callback)
  ensureSharedObserver()

  return () => {
    popupChangeListeners.delete(callback)
    // Если нет подписчиков, останавливаем observer
    if (popupChangeListeners.size === 0) {
      stopSharedObserver()
    }
  }
}

/**
 * Создаёт shared observer если его нет
 */
function ensureSharedObserver(): void {
  if (sharedObserver) return

  sharedObserver = new MutationObserver((mutations) => {
    // Проверяем, затрагивают ли мутации popup элементы
    let hasPopupChange = false

    for (const mutation of mutations) {
      if (mutation.type === 'attributes' && mutation.attributeName === 'data-state') {
        hasPopupChange = true
        break
      }

      if (mutation.type === 'childList') {
        // Проверяем добавленные/удалённые узлы
        for (const node of [...mutation.addedNodes, ...mutation.removedNodes]) {
          if (node instanceof Element) {
            const isPopup =
              node.matches?.('[data-reka-popper-content-wrapper]') ||
              node.matches?.('[data-radix-popper-content-wrapper]') ||
              node.matches?.('[data-state][role="listbox"]') ||
              node.matches?.('[data-state][role="dialog"]') ||
              node.matches?.('[data-state][role="menu"]') ||
              node.matches?.('[data-tutorial-popup]') ||
              node.querySelector?.('[data-reka-popper-content-wrapper]') ||
              node.querySelector?.('[data-state="open"]') ||
              node.querySelector?.('[data-tutorial-popup]')

            if (isPopup) {
              hasPopupChange = true
              break
            }
          }
        }
        if (hasPopupChange) break
      }
    }

    if (hasPopupChange) {
      // Debounce multiple rapid changes (e.g., animation frames)
      if (popupChangeDebounceTimer) {
        clearTimeout(popupChangeDebounceTimer)
      }
      // Небольшая задержка для завершения анимации popup + debounce
      popupChangeDebounceTimer = setTimeout(() => {
        popupChangeListeners.forEach(cb => cb())
        popupChangeDebounceTimer = null
      }, 50)
    }
  })

  sharedObserver.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['data-state'],
  })
}

/**
 * Останавливает shared observer
 */
function stopSharedObserver(): void {
  if (popupChangeDebounceTimer) {
    clearTimeout(popupChangeDebounceTimer)
    popupChangeDebounceTimer = null
  }
  if (sharedObserver) {
    sharedObserver.disconnect()
    sharedObserver = null
  }
}
