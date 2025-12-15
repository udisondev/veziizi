import { ref, reactive, computed } from 'vue'
import type {
  RoutePoint,
  CargoInfo,
  CargoType,
  VehicleRequirements,
  BodyType,
  LoadingType,
  Payment,
  VatType,
  PaymentMethod,
  PaymentTerms,
  ADRClass,
  CreateFreightRequestRequest,
  FreightRequest,
} from '@/types/freightRequest'

// Валидаторы
const validators = {
  required(value: unknown): string | null {
    if (value === undefined || value === null || value === '') return 'Обязательное поле'
    if (Array.isArray(value) && value.length === 0) return 'Выберите хотя бы один вариант'
    return null
  },

  positiveNumber(value: number | undefined | null): string | null {
    if (value === undefined || value === null || value === 0) return 'Обязательное поле'
    if (value <= 0) return 'Должно быть больше 0'
    return null
  },

  minRoutePoints(points: RoutePoint[]): string | null {
    if (points.length < 2) return 'Минимум 2 точки маршрута'
    const hasLoading = points.some((p) => p.is_loading)
    const hasUnloading = points.some((p) => p.is_unloading)
    if (!hasLoading) return 'Добавьте точку погрузки'
    if (!hasUnloading) return 'Добавьте точку разгрузки'
    return null
  },
}

function createEmptyRoutePoint(isFirst: boolean, isLast: boolean): RoutePoint {
  return {
    is_loading: isFirst,
    is_unloading: isLast,
    address: '',
    date_from: '',
    date_to: undefined,
    time_from: undefined,
    time_to: undefined,
    contact_name: undefined,
    contact_phone: undefined,
    comment: undefined,
    coordinates: undefined,
  }
}

export function useFreightRequestForm() {
  const currentStep = ref(1)
  const totalSteps = 5

  // Step 1: Route
  const routePoints = ref<RoutePoint[]>([
    createEmptyRoutePoint(true, false),
    createEmptyRoutePoint(false, true),
  ])

  // Step 2: Cargo
  const cargo = reactive<CargoInfo>({
    description: '',
    weight: 0,
    volume: undefined,
    dimensions: undefined,
    type: 'general' as CargoType,
    adr_class: 'none' as ADRClass,
    quantity: undefined,
  })

  // Step 3: Vehicle
  const vehicle = reactive<VehicleRequirements>({
    body_types: [] as BodyType[],
    loading_types: [] as LoadingType[],
    capacity: undefined,
    volume: undefined,
    length: undefined,
    width: undefined,
    height: undefined,
    requires_adr: false,
    temperature: undefined,
  })

  // Step 4: Payment
  const payment = reactive<Payment>({
    price: undefined,
    vat_type: 'included' as VatType,
    method: 'bank_transfer' as PaymentMethod,
    terms: 'on_unloading' as PaymentTerms,
    deferred_days: undefined,
    no_price: false,
  })

  // Step 5: Additional
  const comment = ref('')
  const expiresAt = ref<string>('')

  // Errors
  const errors = reactive<Record<string, string | null>>({})

  // Route point management
  function addRoutePoint() {
    // Добавляем точку перед последней (которая всегда unloading)
    const newPoint = createEmptyRoutePoint(false, false)
    const insertIndex = routePoints.value.length - 1
    routePoints.value.splice(insertIndex, 0, newPoint)
  }

  function removeRoutePoint(index: number) {
    if (routePoints.value.length > 2) {
      routePoints.value.splice(index, 1)
      // Обеспечиваем, что первая точка loading, последняя unloading
      ensureRouteConstraints()
    }
  }

  function updateRoutePoint(index: number, updates: Partial<RoutePoint>) {
    if (routePoints.value[index]) {
      Object.assign(routePoints.value[index], updates)
    }
  }

  function reorderRoutePoints(newOrder: RoutePoint[]) {
    routePoints.value = newOrder
    ensureRouteConstraints()
  }

  function ensureRouteConstraints() {
    const points = routePoints.value
    if (points.length === 0) return

    const firstPoint = points[0]
    const lastPoint = points[points.length - 1]

    // Первая точка: всегда loading, никогда unloading
    if (firstPoint) {
      firstPoint.is_loading = true
      firstPoint.is_unloading = false
    }

    // Последняя точка: всегда unloading, никогда loading
    if (lastPoint && lastPoint !== firstPoint) {
      lastPoint.is_unloading = true
      lastPoint.is_loading = false
    }

    // Промежуточные точки: если нет ни одного флага, ставим оба
    for (let i = 1; i < points.length - 1; i++) {
      const point = points[i]
      if (point && !point.is_loading && !point.is_unloading) {
        point.is_loading = true
        point.is_unloading = true
      }
    }
  }

  // Validation
  function clearErrors() {
    Object.keys(errors).forEach((key) => {
      errors[key] = null
    })
  }

  function validateStep1(): boolean {
    clearErrors()

    // Гарантируем корректные флаги is_loading/is_unloading перед валидацией
    ensureRouteConstraints()

    errors.route = validators.minRoutePoints(routePoints.value)

    for (const [i, point] of routePoints.value.entries()) {
      errors[`point_${i}_address`] = validators.required(point.address)
      errors[`point_${i}_date_from`] = validators.required(point.date_from)
    }

    const hasRouteError = errors.route !== null
    const hasPointErrors = routePoints.value.some(
      (_, i) => errors[`point_${i}_address`] || errors[`point_${i}_date_from`]
    )

    return !hasRouteError && !hasPointErrors
  }

  function validateStep2(): boolean {
    clearErrors()

    errors.description = validators.required(cargo.description)
    errors.weight = validators.positiveNumber(cargo.weight)
    errors.cargo_type = validators.required(cargo.type)

    return !errors.description && !errors.weight && !errors.cargo_type
  }

  function validateStep3(): boolean {
    clearErrors()

    errors.body_types = validators.required(vehicle.body_types)

    return !errors.body_types
  }

  function validateStep4(): boolean {
    clearErrors()

    // Если галочка "не указывать цену" не отмечена, цена обязательна
    if (!payment.no_price && (!payment.price || payment.price.amount <= 0)) {
      errors.price = 'Укажите стоимость или отметьте "Не указывать цену"'
    }

    // Если выбрана отсрочка, то дни обязательны (только если цена указывается)
    if (!payment.no_price && payment.terms === 'deferred' && !payment.deferred_days) {
      errors.deferred_days = 'Укажите количество дней отсрочки'
    }

    return !errors.price && !errors.deferred_days
  }

  // Navigation
  function nextStep(): boolean {
    const stepValidators: Record<number, () => boolean> = {
      1: validateStep1,
      2: validateStep2,
      3: validateStep3,
      4: validateStep4,
    }

    const validator = stepValidators[currentStep.value]
    if (validator && validator()) {
      currentStep.value++
      return true
    }

    return currentStep.value === 5 // Шаг подтверждения не требует валидации
  }

  function prevStep() {
    if (currentStep.value > 1) {
      currentStep.value--
    }
  }

  function goToStep(step: number) {
    // Можно вернуться на предыдущие шаги, но не прыгнуть вперёд
    if (step >= 1 && step <= currentStep.value) {
      currentStep.value = step
    }
  }

  // Request data
  const requestData = computed<CreateFreightRequestRequest>(() => {
    // Фильтруем пустые опциональные поля
    const cleanedCargo: CargoInfo = {
      description: cargo.description,
      weight: cargo.weight,
      type: cargo.type,
    }

    if (cargo.volume) cleanedCargo.volume = cargo.volume
    if (cargo.quantity) cleanedCargo.quantity = cargo.quantity
    if (cargo.adr_class && cargo.adr_class !== 'none') cleanedCargo.adr_class = cargo.adr_class
    if (cargo.dimensions) {
      const { length, width, height } = cargo.dimensions
      if (length && width && height) {
        cleanedCargo.dimensions = { length, width, height }
      }
    }

    const cleanedVehicle: VehicleRequirements = {
      body_types: vehicle.body_types,
    }

    if (vehicle.loading_types && vehicle.loading_types.length > 0) {
      cleanedVehicle.loading_types = vehicle.loading_types
    }
    if (vehicle.capacity) cleanedVehicle.capacity = vehicle.capacity
    if (vehicle.volume) cleanedVehicle.volume = vehicle.volume
    if (vehicle.length) cleanedVehicle.length = vehicle.length
    if (vehicle.width) cleanedVehicle.width = vehicle.width
    if (vehicle.height) cleanedVehicle.height = vehicle.height
    if (vehicle.requires_adr) cleanedVehicle.requires_adr = vehicle.requires_adr
    if (vehicle.temperature) cleanedVehicle.temperature = vehicle.temperature

    const cleanedPayment: Payment = {
      vat_type: payment.vat_type,
      method: payment.method,
      terms: payment.terms,
    }

    // Добавляем цену и доп. поля только если цена указывается
    if (!payment.no_price && payment.price && payment.price.amount > 0) {
      cleanedPayment.price = { ...payment.price }

      if (payment.terms === 'deferred' && payment.deferred_days) {
        cleanedPayment.deferred_days = payment.deferred_days
      }
    }

    // Очистка точек маршрута
    const cleanedPoints = routePoints.value.map((point) => {
      const cleanedPoint: RoutePoint = {
        is_loading: point.is_loading,
        is_unloading: point.is_unloading,
        address: point.address,
        date_from: point.date_from,
      }

      if (point.coordinates) cleanedPoint.coordinates = point.coordinates
      if (point.date_to) cleanedPoint.date_to = point.date_to
      if (point.time_from) cleanedPoint.time_from = point.time_from
      if (point.time_to) cleanedPoint.time_to = point.time_to
      if (point.contact_name) cleanedPoint.contact_name = point.contact_name
      if (point.contact_phone) cleanedPoint.contact_phone = point.contact_phone
      if (point.comment) cleanedPoint.comment = point.comment

      return cleanedPoint
    })

    const request: CreateFreightRequestRequest = {
      route: { points: cleanedPoints },
      cargo: cleanedCargo,
      vehicle_requirements: cleanedVehicle,
      payment: cleanedPayment,
    }

    if (comment.value) request.comment = comment.value
    if (expiresAt.value) request.expires_at = expiresAt.value

    return request
  })

  // Field validation (on blur)
  function validateField(field: string) {
    // Route points
    const pointMatch = field.match(/^point_(\d+)_(.+)$/)
    if (pointMatch && pointMatch[1] && pointMatch[2]) {
      const index = parseInt(pointMatch[1])
      const pointField = pointMatch[2]
      const point = routePoints.value[index]

      if (point) {
        if (pointField === 'address') {
          errors[field] = validators.required(point.address)
        } else if (pointField === 'date_from') {
          errors[field] = validators.required(point.date_from)
        }
      }
      return
    }

    // Cargo
    if (field === 'description') {
      errors.description = validators.required(cargo.description)
    } else if (field === 'weight') {
      errors.weight = validators.positiveNumber(cargo.weight)
    } else if (field === 'cargo_type') {
      errors.cargo_type = validators.required(cargo.type)
    }

    // Vehicle
    if (field === 'body_types') {
      errors.body_types = validators.required(vehicle.body_types)
    }

    // Payment
    if (field === 'price' && !payment.no_price) {
      errors.price = (!payment.price || payment.price.amount <= 0)
        ? 'Укажите стоимость или отметьте "Не указывать цену"'
        : null
    }
    if (field === 'deferred_days' && !payment.no_price && payment.terms === 'deferred') {
      errors.deferred_days = payment.deferred_days
        ? null
        : 'Укажите количество дней отсрочки'
    }
  }

  // Reset form
  function resetForm() {
    currentStep.value = 1
    routePoints.value = [
      createEmptyRoutePoint(true, false),
      createEmptyRoutePoint(false, true),
    ]
    Object.assign(cargo, {
      description: '',
      weight: 0,
      volume: undefined,
      dimensions: undefined,
      type: 'general',
      adr_class: 'none',
      quantity: undefined,
    })
    Object.assign(vehicle, {
      body_types: [],
      loading_types: [],
      capacity: undefined,
      volume: undefined,
      length: undefined,
      width: undefined,
      height: undefined,
      requires_adr: false,
      temperature: undefined,
    })
    Object.assign(payment, {
      price: undefined,
      vat_type: 'included',
      method: 'bank_transfer',
      terms: 'on_unloading',
      deferred_days: undefined,
      no_price: false,
    })
    comment.value = ''
    expiresAt.value = ''
    clearErrors()
  }

  // Load from existing freight request (for edit mode)
  function loadFromRequest(fr: FreightRequest) {
    // Route points - преобразуем даты из ISO в YYYY-MM-DD для input type="date"
    routePoints.value = fr.route.points.map((p): RoutePoint => ({
      ...p,
      date_from: p.date_from?.split('T')[0] || '',
      date_to: p.date_to?.split('T')[0],
    }))

    // Гарантируем корректные флаги is_loading/is_unloading
    ensureRouteConstraints()

    // Cargo
    Object.assign(cargo, {
      description: fr.cargo.description,
      weight: fr.cargo.weight,
      volume: fr.cargo.volume,
      dimensions: fr.cargo.dimensions ? { ...fr.cargo.dimensions } : undefined,
      type: fr.cargo.type,
      adr_class: fr.cargo.adr_class || 'none',
      quantity: fr.cargo.quantity,
    })

    // Vehicle
    Object.assign(vehicle, {
      body_types: [...fr.vehicle_requirements.body_types],
      loading_types: fr.vehicle_requirements.loading_types ? [...fr.vehicle_requirements.loading_types] : [],
      capacity: fr.vehicle_requirements.capacity,
      volume: fr.vehicle_requirements.volume,
      length: fr.vehicle_requirements.length,
      width: fr.vehicle_requirements.width,
      height: fr.vehicle_requirements.height,
      requires_adr: fr.vehicle_requirements.requires_adr || false,
      temperature: fr.vehicle_requirements.temperature ? { ...fr.vehicle_requirements.temperature } : undefined,
    })

    // Payment
    const hasPrice = fr.payment.price && fr.payment.price.amount > 0
    Object.assign(payment, {
      price: hasPrice ? { ...fr.payment.price } : undefined,
      vat_type: fr.payment.vat_type,
      method: fr.payment.method,
      terms: fr.payment.terms,
      deferred_days: fr.payment.deferred_days,
      no_price: !hasPrice,
    })

    // Additional
    comment.value = fr.comment || ''
    expiresAt.value = fr.expires_at?.split('T')[0] ?? ''

    clearErrors()
  }

  return {
    // State
    currentStep,
    totalSteps,
    routePoints,
    cargo,
    vehicle,
    payment,
    comment,
    expiresAt,
    errors,

    // Route management
    addRoutePoint,
    removeRoutePoint,
    updateRoutePoint,
    reorderRoutePoints,

    // Navigation
    nextStep,
    prevStep,
    goToStep,

    // Validation
    validateStep1,
    validateStep2,
    validateStep3,
    validateStep4,
    validateField,

    // Data
    requestData,

    // Utils
    resetForm,
    loadFromRequest,
  }
}

export type FreightRequestFormReturn = ReturnType<typeof useFreightRequestForm>
