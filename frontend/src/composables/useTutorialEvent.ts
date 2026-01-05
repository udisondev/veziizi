/**
 * Tutorial Event Composable
 * Composable для отправки событий в tutorial систему
 */

import { tutorialBus } from '@/sandbox/events'
import type { TutorialEvents, TutorialEventKey } from '@/types/tutorial'

/**
 * Composable для отправки событий tutorial системе
 * События отправляются только в sandbox режиме (проверяется внутри)
 */
export function useTutorialEvent() {
  /**
   * Отправить событие в tutorial bus
   * @param event - Название события
   * @param payload - Данные события
   */
  function emit<K extends TutorialEventKey>(
    event: K,
    payload?: TutorialEvents[K]
  ) {
    // Событие отправляется всегда, store сам решает реагировать или нет
    tutorialBus.emit(event, payload as TutorialEvents[K])
  }

  /**
   * Подписаться на событие
   * @param event - Название события
   * @param handler - Обработчик
   * @returns Функция отписки
   */
  function on<K extends TutorialEventKey>(
    event: K,
    handler: (payload: TutorialEvents[K]) => void
  ) {
    tutorialBus.on(event, handler)
    return () => tutorialBus.off(event, handler)
  }

  /**
   * Отписаться от события
   * @param event - Название события
   * @param handler - Обработчик
   */
  function off<K extends TutorialEventKey>(
    event: K,
    handler: (payload: TutorialEvents[K]) => void
  ) {
    tutorialBus.off(event, handler)
  }

  return {
    emit,
    on,
    off,
  }
}
