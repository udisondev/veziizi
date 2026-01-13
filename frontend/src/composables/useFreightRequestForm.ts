import { ref, reactive, computed } from 'vue'
import type {
  RoutePoint,
  CargoInfo,
  VehicleRequirements,
  VehicleType,
  VehicleSubType,
  LoadingType,
  Payment,
  VatType,
  PaymentMethod,
  PaymentTerms,
  CreateFreightRequestRequest,
  FreightRequest,
} from '@/types/freightRequest'
import { freightFormValidators as validators } from '@/utils/freightFormValidators'
import { useRoutePointsManager } from '@/composables/useRoutePointsManager'

export function useFreightRequestForm() {
  const currentStep = ref(1)
  const totalSteps = 5

  // Step 1: Route - delegate to useRoutePointsManager
  const {
    routePoints,
    addRoutePoint,
    removeRoutePoint,
    updateRoutePoint,
    reorderRoutePoints,
    ensureRouteConstraints,
    resetRoutePoints,
    loadRoutePoints,
  } = useRoutePointsManager()

  // Step 2: Cargo
  const cargo = reactive<CargoInfo>({
    description: '',
    weight: 0,
    volume: undefined,
    dimensions: undefined,
    quantity: undefined,
    adr_class: undefined,
  })

  // Step 3: Vehicle
  const vehicle = reactive<VehicleRequirements>({
    vehicle_type: undefined as unknown as VehicleType,
    vehicle_subtype: undefined as unknown as VehicleSubType,
    loading_types: [] as LoadingType[],
    capacity: undefined,
    volume: undefined,
    length: undefined,
    width: undefined,
    height: undefined,
    temperature: undefined,
    thermograph: undefined,
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

  // Errors
  const errors = reactive<Record<string, string | null>>({})

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

      // Проверка date_from: обязательность
      const dateFromRequired = validators.required(point.date_from)
      if (dateFromRequired) {
        errors[`point_${i}_date_from`] = dateFromRequired
      } else {
        // Первая точка: не в прошлом
        if (i === 0) {
          errors[`point_${i}_date_from`] = validators.dateNotInPast(point.date_from)
        } else {
          // Последующие точки: >= предыдущей
          const prevPoint = routePoints.value[i - 1]
          errors[`point_${i}_date_from`] = validators.dateSequence(
            point.date_from,
            prevPoint.date_to,
            prevPoint.date_from
          )
        }
      }

      // Проверка date_to >= date_from
      if (point.date_to && point.date_from) {
        errors[`point_${i}_date_to`] = validators.dateToAfterFrom(point.date_from, point.date_to)
      }

      // Если хотя бы одно поле контакта заполнено, оба обязательны
      const hasContactName = !!point.contact_name?.trim()
      const hasContactPhone = !!point.contact_phone?.trim()
      if (hasContactName !== hasContactPhone) {
        if (!hasContactName) {
          errors[`point_${i}_contact_name`] = 'Укажите имя контакта'
        }
        if (!hasContactPhone) {
          errors[`point_${i}_contact_phone`] = 'Укажите телефон контакта'
        }
      }
    }

    const hasRouteError = errors.route !== null
    const hasPointErrors = routePoints.value.some(
      (_, i) =>
        errors[`point_${i}_address`] ||
        errors[`point_${i}_date_from`] ||
        errors[`point_${i}_date_to`] ||
        errors[`point_${i}_contact_name`] ||
        errors[`point_${i}_contact_phone`]
    )

    return !hasRouteError && !hasPointErrors
  }

  function validateStep2(): boolean {
    clearErrors()

    errors.description = validators.required(cargo.description)
    errors.weight = validators.positiveNumber(cargo.weight)
    errors.quantity = validators.positiveNumber(cargo.quantity)

    return !errors.description && !errors.weight && !errors.quantity
  }

  function validateStep3(): boolean {
    clearErrors()

    errors.vehicle_type = validators.required(vehicle.vehicle_type)
    errors.vehicle_subtype = validators.required(vehicle.vehicle_subtype)

    // Если температура включена, проверяем min и max
    if (vehicle.temperature !== undefined) {
      if (vehicle.temperature.min === undefined || vehicle.temperature.min === null) {
        errors.temperature_min = 'Укажите минимальную температуру'
      }
      if (vehicle.temperature.max === undefined || vehicle.temperature.max === null) {
        errors.temperature_max = 'Укажите максимальную температуру'
      }
      if (
        vehicle.temperature.min !== undefined &&
        vehicle.temperature.max !== undefined &&
        vehicle.temperature.min > vehicle.temperature.max
      ) {
        errors.temperature = 'Минимум не может быть больше максимума'
      }
    }

    return !errors.vehicle_type && !errors.vehicle_subtype && !errors.temperature_min && !errors.temperature_max && !errors.temperature
  }

  function validateStep4(): boolean {
    clearErrors()

    // Если галочка "не указывать цену" не отмечена, цена обязательна
    if (!payment.no_price && (!payment.price || payment.price.amount <= 0)) {
      errors.price = 'Укажите стоимость или отметьте "Не указывать цену"'
    }

    // Если выбрана отсрочка, то дни обязательны и должны быть > 0
    if (payment.terms === 'deferred' && (!payment.deferred_days || payment.deferred_days <= 0)) {
      errors.deferred_days = 'Укажите количество дней отсрочки (больше 0)'
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
    }

    if (cargo.volume) cleanedCargo.volume = cargo.volume
    if (cargo.quantity) cleanedCargo.quantity = cargo.quantity
    if (cargo.dimensions) {
      const { length, width, height } = cargo.dimensions
      if (length && width && height) {
        cleanedCargo.dimensions = { length, width, height }
      }
    }
    if (cargo.adr_class) cleanedCargo.adr_class = cargo.adr_class

    const cleanedVehicle: VehicleRequirements = {
      vehicle_type: vehicle.vehicle_type,
      vehicle_subtype: vehicle.vehicle_subtype,
    }

    if (vehicle.loading_types && vehicle.loading_types.length > 0) {
      cleanedVehicle.loading_types = vehicle.loading_types
    }
    if (vehicle.capacity) cleanedVehicle.capacity = vehicle.capacity
    if (vehicle.volume) cleanedVehicle.volume = vehicle.volume
    if (vehicle.length) cleanedVehicle.length = vehicle.length
    if (vehicle.width) cleanedVehicle.width = vehicle.width
    if (vehicle.height) cleanedVehicle.height = vehicle.height
    if (vehicle.temperature) cleanedVehicle.temperature = vehicle.temperature
    if (vehicle.thermograph) cleanedVehicle.thermograph = vehicle.thermograph

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

      // Geo IDs for filtering
      if (point.country_id) cleanedPoint.country_id = point.country_id
      if (point.city_id) cleanedPoint.city_id = point.city_id
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
          // Проверка обязательности
          const reqError = validators.required(point.date_from)
          if (reqError) {
            errors[field] = reqError
          } else if (index === 0) {
            // Первая точка: не в прошлом
            errors[field] = validators.dateNotInPast(point.date_from)
          } else {
            // Последующие точки: >= предыдущей
            const prevPoint = routePoints.value[index - 1]
            errors[field] = validators.dateSequence(
              point.date_from,
              prevPoint.date_to,
              prevPoint.date_from
            )
          }
        } else if (pointField === 'date_to') {
          // date_to >= date_from
          if (point.date_to && point.date_from) {
            errors[field] = validators.dateToAfterFrom(point.date_from, point.date_to)
          } else {
            errors[field] = null
          }
        } else if (pointField === 'contact_name' || pointField === 'contact_phone') {
          // Валидация контактов — оба поля обязательны если одно заполнено
          const hasContactName = !!point.contact_name?.trim()
          const hasContactPhone = !!point.contact_phone?.trim()
          if (hasContactName !== hasContactPhone) {
            if (!hasContactName) {
              errors[`point_${index}_contact_name`] = 'Укажите имя контакта'
            } else {
              errors[`point_${index}_contact_name`] = null
            }
            if (!hasContactPhone) {
              errors[`point_${index}_contact_phone`] = 'Укажите телефон контакта'
            } else {
              errors[`point_${index}_contact_phone`] = null
            }
          } else {
            errors[`point_${index}_contact_name`] = null
            errors[`point_${index}_contact_phone`] = null
          }
        }
      }
      return
    }

    // Cargo
    if (field === 'description') {
      errors.description = validators.required(cargo.description)
    } else if (field === 'weight') {
      errors.weight = validators.positiveNumber(cargo.weight)
    } else if (field === 'quantity') {
      errors.quantity = validators.positiveNumber(cargo.quantity)
    }

    // Vehicle
    if (field === 'vehicle_type') {
      errors.vehicle_type = validators.required(vehicle.vehicle_type)
    }
    if (field === 'vehicle_subtype') {
      errors.vehicle_subtype = validators.required(vehicle.vehicle_subtype)
    }

    // Payment
    if (field === 'price' && !payment.no_price) {
      errors.price = (!payment.price || payment.price.amount <= 0)
        ? 'Укажите стоимость или отметьте "Не указывать цену"'
        : null
    }
    if (field === 'deferred_days' && payment.terms === 'deferred') {
      errors.deferred_days = (payment.deferred_days && payment.deferred_days > 0)
        ? null
        : 'Укажите количество дней отсрочки (больше 0)'
    }
  }

  // Reset form
  function resetForm() {
    currentStep.value = 1
    resetRoutePoints()
    Object.assign(cargo, {
      description: '',
      weight: 0,
      volume: undefined,
      dimensions: undefined,
      quantity: undefined,
      adr_class: undefined,
    })
    Object.assign(vehicle, {
      vehicle_type: undefined,
      vehicle_subtype: undefined,
      loading_types: [],
      capacity: undefined,
      volume: undefined,
      length: undefined,
      width: undefined,
      height: undefined,
      temperature: undefined,
      thermograph: undefined,
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
    clearErrors()
  }

  // Load from existing freight request (for edit mode)
  function loadFromRequest(fr: FreightRequest) {
    // Route points - delegate to manager
    loadRoutePoints(fr.route.points)

    // Cargo
    Object.assign(cargo, {
      description: fr.cargo.description,
      weight: fr.cargo.weight,
      volume: fr.cargo.volume,
      dimensions: fr.cargo.dimensions ? { ...fr.cargo.dimensions } : undefined,
      quantity: fr.cargo.quantity,
      adr_class: fr.cargo.adr_class,
    })

    // Vehicle
    Object.assign(vehicle, {
      vehicle_type: fr.vehicle_requirements.vehicle_type,
      vehicle_subtype: fr.vehicle_requirements.vehicle_subtype,
      loading_types: fr.vehicle_requirements.loading_types ? [...fr.vehicle_requirements.loading_types] : [],
      capacity: fr.vehicle_requirements.capacity,
      volume: fr.vehicle_requirements.volume,
      length: fr.vehicle_requirements.length,
      width: fr.vehicle_requirements.width,
      height: fr.vehicle_requirements.height,
      temperature: fr.vehicle_requirements.temperature ? { ...fr.vehicle_requirements.temperature } : undefined,
      thermograph: fr.vehicle_requirements.thermograph,
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
