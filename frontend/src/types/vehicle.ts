import type { VehicleType, VehicleSubType, LoadingType } from './freightRequest'

export type { VehicleType, VehicleSubType, LoadingType }

export interface Vehicle {
  id: string
  organization_id: string
  vehicle_type: VehicleType
  vehicle_subtype: VehicleSubType
  registration_number: string
  year?: number
  capacity?: number       // грузоподъёмность, кг
  volume?: number         // объём кузова, м³
  length?: number
  width?: number
  height?: number
  loading_types?: LoadingType[]
  created_at: string
}

export interface CreateVehicleRequest {
  vehicle_type: VehicleType
  vehicle_subtype: VehicleSubType
  registration_number: string
  year?: number
  capacity?: number
  volume?: number
  length?: number
  width?: number
  height?: number
  loading_types?: LoadingType[]
}

export interface UpdateVehicleRequest extends Partial<CreateVehicleRequest> {}
