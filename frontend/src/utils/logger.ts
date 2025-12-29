/**
 * Simple logger utility for frontend
 * Provides consistent logging with context
 */

export interface LogContext {
  [key: string]: unknown
}

/**
 * Log levels
 */
export type LogLevel = 'debug' | 'info' | 'warn' | 'error'

/**
 * Logger configuration
 */
const config = {
  /** Enable debug logs in development */
  debug: import.meta.env.DEV,
  /** Prefix for all log messages */
  prefix: '[veziizi]',
}

/**
 * Format context object for logging
 */
function formatContext(context?: LogContext): string {
  if (!context || Object.keys(context).length === 0) return ''
  return ' ' + JSON.stringify(context)
}

/**
 * Logger object with methods for each log level
 */
export const logger = {
  /**
   * Debug log - only in development
   */
  debug(message: string, context?: LogContext): void {
    if (config.debug) {
      console.debug(`${config.prefix} ${message}${formatContext(context)}`)
    }
  },

  /**
   * Info log
   */
  info(message: string, context?: LogContext): void {
    console.info(`${config.prefix} ${message}${formatContext(context)}`)
  },

  /**
   * Warning log
   */
  warn(message: string, context?: LogContext): void {
    console.warn(`${config.prefix} ${message}${formatContext(context)}`)
  },

  /**
   * Error log
   */
  error(message: string, error?: unknown, context?: LogContext): void {
    const errorMessage = error instanceof Error ? error.message : String(error ?? '')
    const fullContext = {
      ...context,
      ...(errorMessage && { error: errorMessage }),
    }
    console.error(`${config.prefix} ${message}${formatContext(fullContext)}`)

    // In development, also log the full error for debugging
    if (config.debug && error instanceof Error) {
      console.error(error)
    }
  },
}

export default logger
