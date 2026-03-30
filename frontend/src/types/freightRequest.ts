// Enums (синхронизированы с backend)

// 8 типов транспортных средств
export type VehicleType =
  | 'van'
  | 'flatbed'
  | 'tanker'
  | 'dump_truck'
  | 'specialized_truck'
  | 'light_truck'
  | 'medium_truck'
  | 'heavy_truck'

// 35 подтипов транспортных средств
export type VehicleSubType =
  // Van (фургон)
  | 'dry_van'
  | 'insulated'
  | 'refrigerator'
  | 'curtain_side'
  | 'box_truck'
  | 'furniture_van'
  // Flatbed (платформа)
  | 'standard_flatbed'
  | 'drop_deck'
  | 'lowboy'
  | 'extendable'
  | 'conestoga'
  // Tanker (цистерна)
  | 'liquid_tanker'
  | 'gas_tanker'
  | 'chemical_tanker'
  | 'food_tanker'
  | 'bitumen_tanker'
  // Dump truck (самосвал)
  | 'rear_dump'
  | 'side_dump'
  | 'bottom_dump'
  // Specialized truck (специализированный)
  | 'car_carrier'
  | 'timber_truck'
  | 'grain_truck'
  | 'livestock_carrier'
  | 'concrete_mixer'
  | 'container_chassis'
  | 'tow_truck'
  | 'crane_truck'
  // Light truck (легкий грузовик)
  | 'city_van'
  | 'pickup'
  | 'minivan_cargo'
  // Medium truck (средний грузовик)
  | 'medium_box'
  | 'medium_flatbed'
  // Heavy truck (тяжелый грузовик)
  | 'semi_trailer'
  | 'road_train'
  | 'mega_trailer'

export type LoadingType = 'rear' | 'side' | 'top' | 'full_untarp'

export type Currency = 'RUB' | 'KZT' | 'BYN' | 'EUR' | 'USD'

export type VatType = 'included' | 'excluded' | 'none'

export type PaymentMethod = 'bank_transfer' | 'cash' | 'card'

export type PaymentTerms = 'prepaid' | 'on_loading' | 'on_unloading' | 'deferred'

export type ADRClass =
  | 'none'
  | 'class1'
  | 'class2'
  | 'class3'
  | 'class4'
  | 'class5'
  | 'class6'
  | 'class7'
  | 'class8'
  | 'class9'

export type FreightRequestStatus =
  | 'published'
  | 'selected'
  | 'confirmed'
  | 'partially_completed'
  | 'completed'
  | 'cancelled'
  | 'cancelled_after_confirmed'
  | 'expired'

// Interfaces
export interface Coordinates {
  latitude: number
  longitude: number
}

export interface RoutePoint {
  // Internal ID for Vue reactivity (not sent to backend)
  _uid?: string

  is_loading: boolean
  is_unloading: boolean

  // Structured location (new)
  country_id?: number
  city_id?: number

  // Legacy address field (for backward compatibility)
  address: string
  coordinates?: Coordinates

  date_from: string // ISO date
  date_to?: string
  time_from?: string // HH:mm
  time_to?: string
  contact_name?: string
  contact_phone?: string
  comment?: string
}

export interface Route {
  points: RoutePoint[]
}

export interface Dimensions {
  length: number
  width: number
  height: number
}

export interface CargoInfo {
  description: string
  weight: number
  volume?: number
  dimensions?: Dimensions
  adr_class?: ADRClass
  quantity?: number
}

export interface Temperature {
  min: number
  max: number
}

export interface VehicleRequirements {
  vehicle_type: VehicleType
  vehicle_subtype: VehicleSubType
  loading_types?: LoadingType[]
  capacity?: number
  volume?: number
  length?: number
  width?: number
  height?: number
  requires_adr?: boolean
  temperature?: Temperature
  thermograph?: boolean // устройство фиксации температуры в пути
}

export interface Money {
  amount: number // в копейках/центах
  currency: Currency
}

export interface Payment {
  price?: Money
  vat_type: VatType
  method: PaymentMethod
  terms: PaymentTerms
  deferred_days?: number
  no_price?: boolean // флаг для формы, не отправляется на сервер
}

export interface CreateFreightRequestRequest {
  route: Route
  cargo: CargoInfo
  vehicle_requirements: VehicleRequirements
  payment: Payment
  comment?: string
  expires_at?: string // ISO date
}

export interface CreateFreightRequestResponse {
  id: string
}

export interface FreightReview {
  id: string
  rating: number
  comment?: string
  created_at: string
  can_edit: boolean
  edit_expires_at?: string
}

export interface FreightRequest {
  id: string
  request_number: number
  customer_org_id: string
  customer_org_name: string
  customer_member_id: string
  customer_member_name?: string
  route: Route
  cargo: CargoInfo
  vehicle_requirements: VehicleRequirements
  payment: Payment
  comment?: string
  status: FreightRequestStatus
  freight_version: number
  expires_at: string
  created_at: string
  // Carrier info (видно при confirmed или перевозчику)
  carrier_org_id?: string
  carrier_org_name?: string
  carrier_member_id?: string
  carrier_member_name?: string
  // Completion status
  customer_completed: boolean
  customer_completed_at?: string
  carrier_completed: boolean
  carrier_completed_at?: string
  completed_at?: string
  // Reviews
  customer_review?: FreightReview
  carrier_review?: FreightReview
}

// List item with display data (from projection)
export interface FreightRequestListItem {
  id: string
  request_number: number
  customer_org_id: string
  status: FreightRequestStatus
  expires_at: string
  created_at: string
  origin_address?: string
  destination_address?: string
  route?: Route
  cargo_weight?: number
  price_amount?: number
  price_currency?: Currency
  vehicle_type?: VehicleType
  vehicle_subtype?: VehicleSubType
  customer_org_name?: string
  customer_org_inn?: string
  customer_org_country?: string
  customer_member_id?: string
}

// Filter types
export type OwnershipFilter = 'all' | 'my_org' | 'my'

export type Country = 'RU' | 'KZ' | 'BY'

export const countryLabels: Record<Country, string> = {
  RU: 'Россия',
  KZ: 'Казахстан',
  BY: 'Беларусь',
}

export const ownershipOptions: { value: OwnershipFilter; label: string }[] = [
  { value: 'all', label: 'Все заявки' },
  { value: 'my_org', label: 'Моей организации' },
  { value: 'my', label: 'Мои' },
]

// Filter types (includes 'all' for "show all")
export type CountryFilter = Country | 'all'
export type FreightRequestStatusFilter = FreightRequestStatus | 'all'

export const countryOptions: { value: CountryFilter; label: string }[] = [
  { value: 'all', label: 'Все страны' },
  { value: 'RU', label: 'Россия' },
  { value: 'KZ', label: 'Казахстан' },
  { value: 'BY', label: 'Беларусь' },
]

export const statusOptions: { value: FreightRequestStatusFilter; label: string }[] = [
  { value: 'all', label: 'Все статусы' },
  { value: 'published', label: 'Опубликованы' },
  { value: 'selected', label: 'Выбран перевозчик' },
  { value: 'confirmed', label: 'Подтверждены' },
  { value: 'partially_completed', label: 'Частично завершены' },
  { value: 'completed', label: 'Завершены' },
  { value: 'cancelled', label: 'Отменены' },
  { value: 'cancelled_after_confirmed', label: 'Отменены после подтверждения' },
  { value: 'expired', label: 'Истекли' },
]

// Labels для UI

// Лейблы типов транспорта
export const vehicleTypeLabels: Record<VehicleType, string> = {
  van: 'Фургон',
  flatbed: 'Платформа',
  tanker: 'Цистерна',
  dump_truck: 'Самосвал',
  specialized_truck: 'Спецтранспорт',
  light_truck: 'Легкий грузовик',
  medium_truck: 'Средний грузовик',
  heavy_truck: 'Тяжелый грузовик',
}

// Лейблы подтипов транспорта
export const vehicleSubTypeLabels: Record<VehicleSubType, string> = {
  // Van
  dry_van: 'Сухой фургон',
  insulated: 'Изотермический',
  refrigerator: 'Рефрижератор',
  curtain_side: 'Тентованный',
  box_truck: 'Цельнометаллический',
  furniture_van: 'Мебельный фургон',
  // Flatbed
  standard_flatbed: 'Стандартная платформа',
  drop_deck: 'Низкорамная платформа',
  lowboy: 'Низкорамник',
  extendable: 'Раздвижная платформа',
  conestoga: 'Конестога',
  // Tanker
  liquid_tanker: 'Жидкостная цистерна',
  gas_tanker: 'Газовая цистерна',
  chemical_tanker: 'Химическая цистерна',
  food_tanker: 'Пищевая цистерна',
  bitumen_tanker: 'Битумовоз',
  // Dump truck
  rear_dump: 'Задняя разгрузка',
  side_dump: 'Боковая разгрузка',
  bottom_dump: 'Донная разгрузка',
  // Specialized
  car_carrier: 'Автовоз',
  timber_truck: 'Лесовоз',
  grain_truck: 'Зерновоз',
  livestock_carrier: 'Скотовоз',
  concrete_mixer: 'Бетоносмеситель',
  container_chassis: 'Контейнеровоз',
  tow_truck: 'Эвакуатор',
  crane_truck: 'Кран-манипулятор',
  // Light truck
  city_van: 'Городской фургон',
  pickup: 'Пикап',
  minivan_cargo: 'Грузовой минивэн',
  // Medium truck
  medium_box: 'Среднетоннажный фургон',
  medium_flatbed: 'Среднетоннажная платформа',
  // Heavy truck
  semi_trailer: 'Полуприцеп',
  road_train: 'Автопоезд',
  mega_trailer: 'Мега-трейлер',
}

// Маппинг: какие подтипы доступны для каждого типа
export const vehicleTypeSubTypes: Record<VehicleType, VehicleSubType[]> = {
  van: ['dry_van', 'insulated', 'refrigerator', 'curtain_side', 'box_truck', 'furniture_van'],
  flatbed: ['standard_flatbed', 'drop_deck', 'lowboy', 'extendable', 'conestoga'],
  tanker: ['liquid_tanker', 'gas_tanker', 'chemical_tanker', 'food_tanker', 'bitumen_tanker'],
  dump_truck: ['rear_dump', 'side_dump', 'bottom_dump'],
  specialized_truck: ['car_carrier', 'timber_truck', 'grain_truck', 'livestock_carrier', 'concrete_mixer', 'container_chassis', 'tow_truck', 'crane_truck'],
  light_truck: ['city_van', 'pickup', 'minivan_cargo'],
  medium_truck: ['medium_box', 'medium_flatbed'],
  heavy_truck: ['semi_trailer', 'road_train', 'mega_trailer'],
}

// Обратный маппинг: подтип → тип транспорта
export const subTypeToVehicleType: Record<VehicleSubType, VehicleType> = {
  // Van
  dry_van: 'van',
  insulated: 'van',
  refrigerator: 'van',
  curtain_side: 'van',
  box_truck: 'van',
  furniture_van: 'van',
  // Flatbed
  standard_flatbed: 'flatbed',
  drop_deck: 'flatbed',
  lowboy: 'flatbed',
  extendable: 'flatbed',
  conestoga: 'flatbed',
  // Tanker
  liquid_tanker: 'tanker',
  gas_tanker: 'tanker',
  chemical_tanker: 'tanker',
  food_tanker: 'tanker',
  bitumen_tanker: 'tanker',
  // Dump truck
  rear_dump: 'dump_truck',
  side_dump: 'dump_truck',
  bottom_dump: 'dump_truck',
  // Specialized truck
  car_carrier: 'specialized_truck',
  timber_truck: 'specialized_truck',
  grain_truck: 'specialized_truck',
  livestock_carrier: 'specialized_truck',
  concrete_mixer: 'specialized_truck',
  container_chassis: 'specialized_truck',
  tow_truck: 'specialized_truck',
  crane_truck: 'specialized_truck',
  // Light truck
  city_van: 'light_truck',
  pickup: 'light_truck',
  minivan_cargo: 'light_truck',
  // Medium truck
  medium_box: 'medium_truck',
  medium_flatbed: 'medium_truck',
  // Heavy truck
  semi_trailer: 'heavy_truck',
  road_train: 'heavy_truck',
  mega_trailer: 'heavy_truck',
}

// Получить тип транспорта по подтипу
export function getVehicleTypeForSubType(subtype: VehicleSubType): VehicleType {
  return subTypeToVehicleType[subtype]
}

// Проверить совместимость подтипа с типом
export function isSubTypeCompatible(vehicleType: VehicleType, subtype: VehicleSubType): boolean {
  return vehicleTypeSubTypes[vehicleType].includes(subtype)
}

export const loadingTypeLabels: Record<LoadingType, string> = {
  rear: 'Задняя',
  side: 'Боковая',
  top: 'Верхняя',
  full_untarp: 'Полная растентовка',
}

export const currencyLabels: Record<Currency, string> = {
  RUB: '₽',
  KZT: '₸',
  BYN: 'Br',
  EUR: '€',
  USD: '$',
}

export const currencyNames: Record<Currency, string> = {
  RUB: 'Российский рубль',
  KZT: 'Казахстанский тенге',
  BYN: 'Белорусский рубль',
  EUR: 'Евро',
  USD: 'Доллар США',
}

export const vatTypeLabels: Record<VatType, string> = {
  included: 'С НДС',
  excluded: 'Без НДС',
  none: 'НДС не применим',
}

export const paymentMethodLabels: Record<PaymentMethod, string> = {
  bank_transfer: 'Безналичный расчёт',
  cash: 'Наличные',
  card: 'Картой',
}

export const paymentTermsLabels: Record<PaymentTerms, string> = {
  prepaid: 'Предоплата',
  on_loading: 'При погрузке',
  on_unloading: 'При выгрузке',
  deferred: 'Отсрочка',
}

export const adrClassLabels: Record<ADRClass, string> = {
  none: 'Не требуется',
  class1: 'Класс 1 — Взрывчатые вещества',
  class2: 'Класс 2 — Газы',
  class3: 'Класс 3 — Легковоспламеняющиеся жидкости',
  class4: 'Класс 4 — Легковоспламеняющиеся твёрдые вещества',
  class5: 'Класс 5 — Окисляющие вещества',
  class6: 'Класс 6 — Токсичные вещества',
  class7: 'Класс 7 — Радиоактивные материалы',
  class8: 'Класс 8 — Коррозионные вещества',
  class9: 'Класс 9 — Прочие опасные вещества',
}

export const freightRequestStatusLabels: Record<FreightRequestStatus, string> = {
  published: 'Опубликована',
  selected: 'Выбран перевозчик',
  confirmed: 'Подтверждена',
  partially_completed: 'Частично завершена',
  completed: 'Завершена',
  cancelled: 'Отменена',
  cancelled_after_confirmed: 'Отменена после подтверждения',
  expired: 'Истекла',
}

// Options для селектов
export const vehicleTypeOptions = Object.entries(vehicleTypeLabels).map(([value, label]) => ({
  value: value as VehicleType,
  label,
}))

// Функция для получения подтипов по типу
export function getVehicleSubTypeOptions(vehicleType: VehicleType) {
  const subtypes = vehicleTypeSubTypes[vehicleType] || []
  return subtypes.map((value) => ({
    value,
    label: vehicleSubTypeLabels[value],
  }))
}

// Все подтипы для начального состояния (без фильтрации по типу)
export const allVehicleSubTypeOptions = Object.entries(vehicleSubTypeLabels).map(([value, label]) => ({
  value: value as VehicleSubType,
  label,
}))

export const loadingTypeOptions = Object.entries(loadingTypeLabels).map(([value, label]) => ({
  value: value as LoadingType,
  label,
}))

export const currencyOptions = Object.entries(currencyLabels).map(([value, label]) => ({
  value: value as Currency,
  label: `${label} (${value})`,
}))

export const vatTypeOptions = Object.entries(vatTypeLabels).map(([value, label]) => ({
  value: value as VatType,
  label,
}))

export const paymentMethodOptions = Object.entries(paymentMethodLabels).map(([value, label]) => ({
  value: value as PaymentMethod,
  label,
}))

export const paymentTermsOptions = Object.entries(paymentTermsLabels).map(([value, label]) => ({
  value: value as PaymentTerms,
  label,
}))

export const adrClassOptions = Object.entries(adrClassLabels).map(([value, label]) => ({
  value: value as ADRClass,
  label,
}))

// Offer types
export type OfferStatus =
  | 'pending'
  | 'selected'
  | 'confirmed'
  | 'rejected'
  | 'withdrawn'
  | 'declined'

export interface Offer {
  id: string
  carrier_org_id: string
  carrier_org_name?: string
  carrier_member_id?: string
  carrier_member_name?: string
  price: Money
  comment?: string
  vat_type: VatType
  payment_method: PaymentMethod
  freight_version: number
  status: OfferStatus
  created_at: string
}

export interface MakeOfferRequest {
  price: Money
  comment?: string
  vat_type: VatType
  payment_method: PaymentMethod
}

export interface MakeOfferResponse {
  offer_id: string
}

export const offerStatusLabels: Record<OfferStatus, string> = {
  pending: 'Ожидает',
  selected: 'Выбран',
  confirmed: 'Подтверждён',
  rejected: 'Отклонён',
  withdrawn: 'Отозван',
  declined: 'Отказ',
}

export const offerStatusColors: Record<OfferStatus, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  selected: 'bg-blue-100 text-blue-800',
  confirmed: 'bg-green-100 text-green-800',
  rejected: 'bg-red-100 text-red-800',
  withdrawn: 'bg-gray-100 text-gray-800',
  declined: 'bg-orange-100 text-orange-800',
}

export type OfferStatusFilter = OfferStatus | 'all'

export const offerStatusOptions: { value: OfferStatusFilter, label: string }[] = [
  { value: 'all', label: 'Все статусы' },
  { value: 'pending', label: 'Ожидает' },
  { value: 'selected', label: 'Выбран' },
  { value: 'confirmed', label: 'Подтверждён' },
  { value: 'rejected', label: 'Отклонён' },
  { value: 'withdrawn', label: 'Отозван' },
  { value: 'declined', label: 'Отказ' },
]
