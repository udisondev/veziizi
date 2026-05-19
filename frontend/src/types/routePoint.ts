export type RoutePointRole = 'any' | 'origin' | 'destination'

export interface RoutePointData {
  id: string
  countryId?: number
  countryName?: string
  cityId?: number
  cityName?: string
  order: number
  role?: RoutePointRole
}
