/**
 * Tutorial Popup Tracker
 * Отслеживает открытые popup/dropdown элементы для динамического
 * расширения области подсветки tutorial и позиционирования tooltip
 */

import { ref, type Ref } from 'vue'

// Универсальные селекторы для определения открытых popup
const POPUP_SELECTORS = [
  // reka-ui Select/Popover/Dialog (используют data-state)
  '[data-state="open"][role="listbox"]',
  '[data-state="open"][role="dialog"]',
  '[data-state="open"][role="menu"]',

  // Reka-ui / Radix portals
  '[data-reka-popper-content-wrapper]',
  '[data-radix-popper-content-wrapper]',

  // Кастомные dropdown (абсолютное позиционирование + z-index)
  // City autocomplete dropdown pattern
  '.absolute.z-50.bg-white.border.rounded-md.shadow-lg',
  '.absolute.z-50.bg-popover',
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
    const maxDistance = 100 // px - максимальное расстояние до popup

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
        console.warn(`[PopupTracker] Invalid selector: ${selector}`, e)
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
   * Получает оптимальную позицию tooltip (противоположную от popup)
   */
  function getOptimalTooltipPosition(
    popupDirection: PopupDirection | null,
    preferredPosition: 'top' | 'bottom' | 'left' | 'right' = 'bottom'
  ): 'top' | 'bottom' | 'left' | 'right' {
    if (!popupDirection) {
      return preferredPosition
    }

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
