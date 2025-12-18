// Enums (синхронизированы с backend)
export type CargoType =
  | 'general'
  | 'bulk'
  | 'liquid'
  | 'refrigerated'
  | 'dangerous'
  | 'oversized'
  | 'container'

export type BodyType =
  | 'tent'
  | 'refrigerator'
  | 'isothermal'
  | 'container'
  | 'openbed'
  | 'lowbed'
  | 'jumbo'
  | 'tank'
  | 'tipper'

export type LoadingType = 'rear' | 'side' | 'top'

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
  | 'cancelled'
  | 'expired'

// Interfaces
export interface Coordinates {
  latitude: number
  longitude: number
}

export interface RoutePoint {
  is_loading: boolean
  is_unloading: boolean
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
  type: CargoType
  adr_class?: ADRClass
  quantity?: number
}

export interface Temperature {
  min: number
  max: number
}

export interface VehicleRequirements {
  body_types: BodyType[]
  loading_types?: LoadingType[]
  capacity?: number
  volume?: number
  length?: number
  width?: number
  height?: number
  requires_adr?: boolean
  temperature?: Temperature
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

export interface FreightRequest {
  id: string
  request_number: number
  customer_org_id: string
  customer_member_id: string
  route: Route
  cargo: CargoInfo
  vehicle_requirements: VehicleRequirements
  payment: Payment
  comment?: string
  status: FreightRequestStatus
  freight_version: number
  expires_at: string
  created_at: string
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
  cargo_type?: CargoType
  cargo_weight?: number
  price_amount?: number
  price_currency?: Currency
  body_types?: BodyType[]
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

export const countryOptions: { value: Country | ''; label: string }[] = [
  { value: '', label: 'Все страны' },
  { value: 'RU', label: 'Россия' },
  { value: 'KZ', label: 'Казахстан' },
  { value: 'BY', label: 'Беларусь' },
]

export const statusOptions: { value: FreightRequestStatus | ''; label: string }[] = [
  { value: '', label: 'Все статусы' },
  { value: 'published', label: 'Опубликованы' },
  { value: 'selected', label: 'Выбран перевозчик' },
  { value: 'confirmed', label: 'Подтверждены' },
  { value: 'cancelled', label: 'Отменены' },
  { value: 'expired', label: 'Истекли' },
]

// Labels для UI
export const cargoTypeLabels: Record<CargoType, string> = {
  general: 'Общий груз',
  bulk: 'Насыпной',
  liquid: 'Наливной',
  refrigerated: 'Рефрижераторный',
  dangerous: 'Опасный',
  oversized: 'Негабаритный',
  container: 'Контейнерный',
}

export const bodyTypeLabels: Record<BodyType, string> = {
  tent: 'Тент',
  refrigerator: 'Рефрижератор',
  isothermal: 'Изотерм',
  container: 'Контейнеровоз',
  openbed: 'Открытая площадка',
  lowbed: 'Низкорамник',
  jumbo: 'Джамбо',
  tank: 'Цистерна',
  tipper: 'Самосвал',
}

export const loadingTypeLabels: Record<LoadingType, string> = {
  rear: 'Задняя',
  side: 'Боковая',
  top: 'Верхняя',
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
  cancelled: 'Отменена',
  expired: 'Истекла',
}

// Options для селектов
export const cargoTypeOptions = Object.entries(cargoTypeLabels).map(([value, label]) => ({
  value: value as CargoType,
  label,
}))

export const bodyTypeOptions = Object.entries(bodyTypeLabels).map(([value, label]) => ({
  value: value as BodyType,
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

export const offerStatusOptions = [
  { value: '' as const, label: 'Все статусы' },
  { value: 'pending' as const, label: 'Ожидает' },
  { value: 'selected' as const, label: 'Выбран' },
  { value: 'confirmed' as const, label: 'Подтверждён' },
  { value: 'rejected' as const, label: 'Отклонён' },
  { value: 'withdrawn' as const, label: 'Отозван' },
  { value: 'declined' as const, label: 'Отказ' },
]
