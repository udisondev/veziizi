/**
 * Типизированные ошибки API для фронтенда
 * Позволяют дифференцировать ошибки по статусу и коду
 */

// V8-специфичный API для stack trace (Node.js/Chrome)
interface ErrorWithCaptureStackTrace extends ErrorConstructor {
  captureStackTrace(targetObject: object, constructorOpt?: Function): void
}

/**
 * HTTP статус коды
 */
export const HttpStatus = {
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  CONFLICT: 409,
  UNPROCESSABLE_ENTITY: 422,
  TOO_MANY_REQUESTS: 429,
  INTERNAL_SERVER_ERROR: 500,
  BAD_GATEWAY: 502,
  SERVICE_UNAVAILABLE: 503,
} as const

export type HttpStatusCode = (typeof HttpStatus)[keyof typeof HttpStatus]

/**
 * Структура ответа ошибки от API
 */
export interface ApiErrorResponse {
  error: string
  error_code?: string
  details?: Record<string, string>
}

/**
 * Класс ошибки API с типизацией
 */
export class ApiError extends Error {
  public readonly status: number
  public readonly code?: string
  public readonly details?: Record<string, string>

  constructor(
    status: number,
    message: string,
    code?: string,
    details?: Record<string, string>
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details

    // Maintain proper stack trace in V8 (Node.js/Chrome)
    if ('captureStackTrace' in Error) {
      ;(Error as ErrorWithCaptureStackTrace).captureStackTrace(this, ApiError)
    }
  }

  /**
   * Проверяет, является ли ошибка ошибкой авторизации
   */
  isUnauthorized(): boolean {
    return this.status === HttpStatus.UNAUTHORIZED
  }

  /**
   * Проверяет, является ли ошибка ошибкой доступа
   */
  isForbidden(): boolean {
    return this.status === HttpStatus.FORBIDDEN
  }

  /**
   * Проверяет, является ли ошибка ошибкой "не найдено"
   */
  isNotFound(): boolean {
    return this.status === HttpStatus.NOT_FOUND
  }

  /**
   * Проверяет, является ли ошибка ошибкой конфликта (например, concurrent modification)
   */
  isConflict(): boolean {
    return this.status === HttpStatus.CONFLICT
  }

  /**
   * Проверяет, является ли ошибка ошибкой валидации
   */
  isValidationError(): boolean {
    return this.status === HttpStatus.BAD_REQUEST || this.status === HttpStatus.UNPROCESSABLE_ENTITY
  }

  /**
   * Проверяет, является ли ошибка серверной ошибкой
   */
  isServerError(): boolean {
    return this.status >= 500
  }

  /**
   * Проверяет, является ли ошибка ошибкой rate limiting
   */
  isRateLimited(): boolean {
    return this.status === HttpStatus.TOO_MANY_REQUESTS
  }

  /**
   * Проверяет, является ли ошибка блокировкой аккаунта
   */
  isAccountBlocked(): boolean {
    return (
      this.status === HttpStatus.FORBIDDEN &&
      this.message.toLowerCase().includes('account is blocked')
    )
  }
}

/**
 * Проверяет, является ли объект ApiError
 */
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}

/**
 * Извлекает сообщение об ошибке из любого типа ошибки
 */
export function getErrorMessage(error: unknown): string {
  if (isApiError(error)) {
    return error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === 'string') {
    return error
  }
  return 'Произошла неизвестная ошибка'
}

/**
 * Стандартные сообщения об ошибках для UI
 */
export const errorMessages: Record<number, string> = {
  [HttpStatus.UNAUTHORIZED]: 'Необходима авторизация',
  [HttpStatus.FORBIDDEN]: 'Недостаточно прав для выполнения операции',
  [HttpStatus.NOT_FOUND]: 'Ресурс не найден',
  [HttpStatus.CONFLICT]: 'Конфликт данных. Попробуйте обновить страницу',
  [HttpStatus.TOO_MANY_REQUESTS]: 'Слишком много запросов. Подождите немного',
  [HttpStatus.INTERNAL_SERVER_ERROR]: 'Ошибка сервера. Попробуйте позже',
  [HttpStatus.BAD_GATEWAY]: 'Сервер временно недоступен',
  [HttpStatus.SERVICE_UNAVAILABLE]: 'Сервис временно недоступен',
}

/**
 * Специальное сообщение для блокировки аккаунта
 */
export const ACCOUNT_BLOCKED_MESSAGE =
  'Ваш аккаунт заблокирован. Обратитесь к администратору организации.'

/**
 * Получает пользовательское сообщение об ошибке по статусу
 */
export function getStatusErrorMessage(status: number): string {
  return errorMessages[status] ?? 'Произошла ошибка'
}
