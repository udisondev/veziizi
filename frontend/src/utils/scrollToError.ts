/**
 * Утилита для прокрутки к первой ошибке валидации формы
 */

import { useOnboardingStore } from '@/stores/onboarding'

/**
 * Маппинг ключа ошибки на CSS-селектор для поиска элемента
 */
function getErrorSelector(errorKey: string): string | null {
  // Общая ошибка маршрута - скроллим к началу списка точек
  if (errorKey === 'route') {
    return '[data-tutorial="route-points-list"]'
  }

  // Ошибки точек маршрута: point_0_date_from, point_1_address и т.д.
  const pointMatch = errorKey.match(/^point_(\d+)_(.+)$/)
  if (pointMatch && pointMatch[1]) {
    const index = parseInt(pointMatch[1])
    // Карточки точек 0, 1, 2 имеют data-tutorial="route-point-N"
    if (index <= 2) {
      return `[data-tutorial="route-point-${index}"]`
    }
    // Для точек без data-tutorial ищем в списке по порядку
    return `[data-tutorial="route-points-list"] > div:nth-child(${index + 1})`
  }

  // Ошибки полей груза (step 2)
  if (['description', 'weight', 'quantity', 'volume'].includes(errorKey)) {
    return '[data-tutorial="cargo-step"]'
  }

  // Ошибки транспорта (step 3)
  if (['vehicle_type', 'vehicle_subtype', 'temperature', 'temperature_min', 'temperature_max'].includes(errorKey)) {
    return '[data-tutorial="vehicle-step"]'
  }

  // Ошибки оплаты (step 4)
  if (['price', 'deferred_days'].includes(errorKey)) {
    return '[data-tutorial="payment-step"]'
  }

  return null
}

/**
 * Найти первую ошибку и прокрутить к ней
 * @param errors - объект ошибок формы
 * @returns true если ошибка найдена и выполнен скролл
 */
export function scrollToFirstError(errors: Record<string, string | null>): boolean {
  // Найти первый ключ с не-null ошибкой
  const firstErrorKey = Object.keys(errors).find(key => errors[key] !== null)

  if (!firstErrorKey) {
    return false
  }

  const selector = getErrorSelector(firstErrorKey)

  if (!selector) {
    return false
  }

  const element = document.querySelector(selector)

  if (!element) {
    return false
  }

  // Входим в режим исправления ошибки — overlay и tooltip скрываются
  const onboarding = useOnboardingStore()
  if (onboarding.isSandboxMode) {
    onboarding.enterValidationErrorMode()
  }

  // Вычисляем позицию для центрирования элемента на экране
  const rect = element.getBoundingClientRect()
  const elementCenter = rect.top + rect.height / 2
  const viewportCenter = window.innerHeight / 2
  const scrollTarget = window.scrollY + elementCenter - viewportCenter

  // Используем window.scrollTo вместо scrollIntoView
  window.scrollTo({
    top: Math.max(0, scrollTarget),
    behavior: 'smooth',
  })

  // Добавить временную подсветку для привлечения внимания
  element.classList.add('ring-2', 'ring-red-500', 'ring-offset-2')
  setTimeout(() => {
    element.classList.remove('ring-2', 'ring-red-500', 'ring-offset-2')
  }, 2000)

  return true
}
