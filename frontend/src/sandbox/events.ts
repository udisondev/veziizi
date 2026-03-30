/**
 * Tutorial Event Bus
 * Event bus для отслеживания действий пользователя в sandbox режиме
 */

import mitt, { type Emitter } from 'mitt'
import type { TutorialEvents } from '@/types/tutorial'

// Создаём типизированный event emitter
// Используем type assertion для обхода строгой проверки mitt
export const tutorialBus = mitt() as unknown as Emitter<TutorialEvents>

// Хелпер для дебага событий (только в dev mode)
if (import.meta.env.DEV) {
  tutorialBus.on('*', (type, event) => {
    console.log('[Tutorial Event]', type, event)
  })
}
