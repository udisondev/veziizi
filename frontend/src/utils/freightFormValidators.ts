/**
 * Validators for freight request form fields
 */

import type { RoutePoint } from '@/types/freightRequest'

/**
 * Form field validators with Russian error messages
 */
export const freightFormValidators = {
  /**
   * Validates that value is not empty
   */
  required(value: unknown): string | null {
    if (value === undefined || value === null || value === '') return 'Обязательное поле'
    if (Array.isArray(value) && value.length === 0) return 'Выберите хотя бы один вариант'
    return null
  },

  /**
   * Validates that number is positive
   */
  positiveNumber(value: number | undefined | null): string | null {
    if (value === undefined || value === null || value === 0) return 'Обязательное поле'
    if (value <= 0) return 'Должно быть больше 0'
    return null
  },

  /**
   * Validates route has at least 2 points with loading and unloading
   */
  minRoutePoints(points: RoutePoint[]): string | null {
    if (points.length < 2) return 'Минимум 2 точки маршрута'
    const hasLoading = points.some((p) => p.is_loading)
    const hasUnloading = points.some((p) => p.is_unloading)
    if (!hasLoading) return 'Добавьте точку погрузки'
    if (!hasUnloading) return 'Добавьте точку разгрузки'
    return null
  },

  /**
   * Validates date is not in the past
   */
  dateNotInPast(value: string): string | null {
    if (!value) return null
    const date = new Date(value)
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    if (date < today) {
      return 'Дата не может быть в прошлом'
    }
    return null
  },

  /**
   * Validates date sequence between route points
   */
  dateSequence(currentDateFrom: string, prevDateTo: string | undefined, prevDateFrom: string): string | null {
    if (!currentDateFrom || !prevDateFrom) return null
    const current = new Date(currentDateFrom)
    const prevEnd = prevDateTo ? new Date(prevDateTo) : new Date(prevDateFrom)
    if (current < prevEnd) {
      return 'Дата должна быть не раньше даты предыдущей точки'
    }
    return null
  },

  /**
   * Validates that date_to is not before date_from
   */
  dateToAfterFrom(dateFrom: string, dateTo: string): string | null {
    if (!dateFrom || !dateTo) return null
    if (new Date(dateTo) < new Date(dateFrom)) {
      return 'Дата окончания не может быть раньше даты начала'
    }
    return null
  },
}
