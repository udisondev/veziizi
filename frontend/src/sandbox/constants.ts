/**
 * Sandbox & Tutorial Constants
 * Централизованные константы для туториала и sandbox режима
 */

// === Tutorial Overlay ===
/** Отступ вокруг подсвечиваемого элемента (px) */
export const TUTORIAL_OVERLAY_PADDING = 8

/** Скругление углов подсветки (px) */
export const TUTORIAL_OVERLAY_BORDER_RADIUS = 8

/** Throttle для RAF loop — обновляем каждые N кадров (~20fps вместо 60fps) */
export const TUTORIAL_RAF_THROTTLE_FRAMES = 3

// === Popup Tracker ===
/** Максимальное расстояние до popup для его включения в область подсветки (px) */
export const TUTORIAL_POPUP_MAX_DISTANCE = 100

// === Sandbox Ready ===
/** Таймаут ожидания готовности sandbox (ms) */
export const SANDBOX_READY_TIMEOUT_MS = 5000

// === Auto Confirm ===
/** Задержка перед автоподтверждением оффера (ms) */
export const AUTO_CONFIRM_DELAY_MS = 100
