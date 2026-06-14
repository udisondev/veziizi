import type { VehicleType, VehicleSubType, LoadingType } from './freightRequest'

export type { VehicleType, VehicleSubType, LoadingType }

export type VehicleStatus = 'unconfirmed' | 'pending' | 'verified' | 'rejected' | 'archived'

export interface VehicleTemperature {
  min: number
  max: number
}

export interface Vehicle {
  id: string
  org_id: string
  vehicle_type: VehicleType
  vehicle_subtype: VehicleSubType
  registration_number: string
  brand?: string
  model?: string
  capacity?: number
  volume?: number
  length?: number
  width?: number
  height?: number
  loading_types?: string[]
  requires_adr: boolean
  temperature?: VehicleTemperature
  thermograph: boolean
  status: VehicleStatus
  rejection_reason?: string
  created_at: string
  updated_at: string
}

export interface CreateVehicleRequest {
  vehicle_type: VehicleType
  vehicle_subtype: VehicleSubType
  registration_number: string
  brand?: string
  model?: string
  capacity?: number
  volume?: number
  length?: number
  width?: number
  height?: number
  loading_types?: LoadingType[]
  requires_adr?: boolean
  temperature?: VehicleTemperature
  thermograph?: boolean
}

export interface UpdateVehicleRequest extends Partial<CreateVehicleRequest> {}

export interface VehicleListItem {
  id: string
  org_id: string
  org_name?: string
  vehicle_type: VehicleType
  vehicle_subtype: VehicleSubType
  registration_number: string
  brand?: string
  model?: string
  capacity?: number
  volume?: number
  loading_types?: LoadingType[]
  requires_adr: boolean
  temperature?: VehicleTemperature
  thermograph: boolean
  // Публичный флаг подтверждения — статус модерации наружу не отдаётся
  verified: boolean
}

export interface VehicleListParams {
  vehicle_subtypes?: string  // comma-separated
  min_capacity?: number
  max_capacity?: number
  min_volume?: number
  max_volume?: number
  loading_types?: string  // comma-separated
  requires_adr?: boolean
  thermograph?: boolean
  org_name?: string
  org_id?: string
  exclude_org_id?: string
  limit?: number
  cursor?: string
}
