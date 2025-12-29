import type {
  VehicleType,
  VehicleSubType,
  PaymentMethod,
  PaymentTerms,
  VatType,
} from './freightRequest'

// Критерий точки маршрута
export interface RoutePointCriteria {
  country_id: number
  country_name?: string
  city_id?: number
  city_name?: string
  order: number
}

// Подписка на заявки
export interface FreightSubscription {
  id: string
  member_id: string
  name: string

  // Числовые диапазоны
  min_weight?: number
  max_weight?: number
  min_price?: number
  max_price?: number
  min_volume?: number
  max_volume?: number

  // ENUM массивы
  vehicle_types?: VehicleType[]
  vehicle_subtypes?: VehicleSubType[]
  payment_methods?: PaymentMethod[]
  payment_terms?: PaymentTerms[]
  vat_types?: VatType[]

  // Маршрут
  route_points?: RoutePointCriteria[]

  is_active: boolean
  created_at: string
  updated_at: string
}

// Запрос на создание подписки
export interface FreightSubscriptionCreate {
  name: string
  min_weight?: number
  max_weight?: number
  min_price?: number
  max_price?: number
  min_volume?: number
  max_volume?: number
  vehicle_types?: VehicleType[]
  vehicle_subtypes?: VehicleSubType[]
  payment_methods?: PaymentMethod[]
  payment_terms?: PaymentTerms[]
  vat_types?: VatType[]
  route_points?: RoutePointCriteriaCreate[]
  is_active: boolean
}

// Точка маршрута для создания (без названий)
export interface RoutePointCriteriaCreate {
  country_id: number
  city_id?: number
  order: number
}

// Запрос на обновление подписки
export type FreightSubscriptionUpdate = FreightSubscriptionCreate

// Лимит подписок на пользователя
export const MAX_SUBSCRIPTIONS_PER_MEMBER = 10
