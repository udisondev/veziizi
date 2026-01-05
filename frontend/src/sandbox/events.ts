/**
 * Tutorial Event Bus
 * Event bus для отслеживания действий пользователя в sandbox режиме
 */

import mitt from 'mitt'
import type { TutorialEvents } from '@/types/tutorial'

// Создаём типизированный event emitter
export const tutorialBus = mitt<TutorialEvents>()

// Хелпер для дебага событий (только в dev mode)
if (import.meta.env.DEV) {
  tutorialBus.on('*', (type, event) => {
    console.log('[Tutorial Event]', type, event)
  })
}
