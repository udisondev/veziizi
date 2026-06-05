/**
 * SSE-клиент: единственный EventSource на всё приложение
 * (GET /api/v1/events/stream), раздаёт события подписчикам по имени.
 *
 * Семантика push-to-refetch: сервер шлёт лёгкие «пинки»
 * {entity_id, event_type}, данные подписчик перечитывает обычными API-вызовами.
 * Реконнект делает сам браузер; при каждом open подписчики onConnected
 * рефетчат состояние — пропуски между соединениями не теряются.
 */
import { ref, readonly } from 'vue'
import { logger } from '@/utils/logger'

/** Имена SSE-событий — зеркало backend/internal/infrastructure/sse/event.go */
export type StreamEventName =
  | 'notification'
  | 'unread'
  | 'freight_request'
  | 'support_ticket'

export interface StreamEvent {
  entity_id: string
  event_type: string
}

type Listener = (event: StreamEvent) => void

const STREAM_URL = '/api/v1/events/stream'

const EVENT_NAMES: StreamEventName[] = [
  'notification',
  'unread',
  'freight_request',
  'support_ticket',
]

/**
 * Через сколько молчания считаем соединение «тихо умершим». Сервер шлёт
 * `event: ping` каждые ~25с (SSE_HEARTBEAT_INTERVAL) — три пропущенных
 * heartbeat'а подряд означают мёртвый сокет (laptop sleep, смена сети без
 * RST), о котором браузер может не узнать и не вызвать onerror.
 */
const STALE_AFTER_MS = 90_000
const WATCHDOG_INTERVAL_MS = 30_000

let eventSource: EventSource | null = null
let watchdogTimer: number | null = null
let lastActivityAt = 0
const listeners = new Map<StreamEventName, Set<Listener>>()
const connectedListeners = new Set<() => void>()

const isConnected = ref(false)

/**
 * Живо ли соединение фактически: открыто И недавно подавало признаки жизни
 * (heartbeat/событие). По этому флагу store решает, можно ли пропустить
 * polling-тик — по одному isConnected нельзя: он не видит тихую смерть сокета.
 */
function isHealthy(): boolean {
  return isConnected.value && Date.now() - lastActivityAt < STALE_AFTER_MS
}

function dispatch(name: StreamEventName, e: MessageEvent): void {
  let event: StreamEvent
  try {
    event = JSON.parse(e.data as string) as StreamEvent
  } catch {
    logger.warn('eventStream: malformed event payload', { name })
    return
  }
  listeners.get(name)?.forEach(cb => {
    try {
      cb(event)
    } catch (err) {
      logger.error('eventStream: listener failed', { name, err })
    }
  })
}

function connect(): void {
  if (eventSource) return

  const es = new EventSource(STREAM_URL, { withCredentials: true })
  eventSource = es

  es.onopen = () => {
    isConnected.value = true
    lastActivityAt = Date.now()
    // Refetch-on-(re)connect: подписчики перечитывают состояние, закрывая
    // окно между соединениями (или после рестарта API).
    connectedListeners.forEach(cb => {
      try {
        cb()
      } catch (err) {
        logger.error('eventStream: onConnected listener failed', { err })
      }
    })
  }

  es.onerror = () => {
    isConnected.value = false
    if (es.readyState === EventSource.CLOSED) {
      // Браузер прекратил реконнекты (например 401 после логаута) — store
      // вернётся к polling-fallback'у по isConnected=false.
      logger.warn('eventStream: connection closed by server')
      disconnect()
    }
    // CONNECTING — браузер сам переподключается, ничего не делаем.
  }

  // Heartbeat сервера: единственный сигнал жизни в тихие периоды.
  es.addEventListener('ping', () => {
    lastActivityAt = Date.now()
  })

  for (const name of EVENT_NAMES) {
    es.addEventListener(name, e => {
      lastActivityAt = Date.now()
      dispatch(name, e)
    })
  }

  startWatchdog()
}

/**
 * Watchdog: браузер не замечает тихо умерший сокет (нет RST после сна/смены
 * сети) и не реконнектится сам. Если соединение «открыто», но heartbeat'ов
 * давно нет — пересоздаём EventSource вручную; onopen вызовет refetch.
 */
function startWatchdog(): void {
  if (watchdogTimer !== null) return
  watchdogTimer = window.setInterval(() => {
    if (eventSource && isConnected.value && !isHealthy()) {
      logger.warn('eventStream: stale connection detected, reconnecting')
      eventSource.close()
      eventSource = null
      isConnected.value = false
      connect()
    }
  }, WATCHDOG_INTERVAL_MS)
}

function stopWatchdog(): void {
  if (watchdogTimer !== null) {
    window.clearInterval(watchdogTimer)
    watchdogTimer = null
  }
}

function disconnect(): void {
  stopWatchdog()
  eventSource?.close()
  eventSource = null
  isConnected.value = false
}

function on(name: StreamEventName, cb: Listener): void {
  if (!listeners.has(name)) {
    listeners.set(name, new Set())
  }
  listeners.get(name)!.add(cb)
}

function off(name: StreamEventName, cb: Listener): void {
  listeners.get(name)?.delete(cb)
}

function onConnected(cb: () => void): void {
  connectedListeners.add(cb)
}

function offConnected(cb: () => void): void {
  connectedListeners.delete(cb)
}

export const eventStream = {
  connect,
  disconnect,
  on,
  off,
  onConnected,
  offConnected,
  isConnected: readonly(isConnected),
  isHealthy,
}
