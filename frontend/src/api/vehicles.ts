// TODO: подключить к реальному API когда бэкенд будет готов
import type { Vehicle, CreateVehicleRequest, UpdateVehicleRequest } from '@/types/vehicle'

export const vehiclesApi = {
  async list(_organizationId: string): Promise<Vehicle[]> {
    // TODO: GET /organizations/:id/vehicles
    return []
  },

  async create(_organizationId: string, _data: CreateVehicleRequest): Promise<Vehicle> {
    // TODO: POST /organizations/:id/vehicles
    throw new Error('Not implemented')
  },

  async update(_organizationId: string, _vehicleId: string, _data: UpdateVehicleRequest): Promise<Vehicle> {
    // TODO: PUT /organizations/:id/vehicles/:vehicleId
    throw new Error('Not implemented')
  },

  async remove(_organizationId: string, _vehicleId: string): Promise<void> {
    // TODO: DELETE /organizations/:id/vehicles/:vehicleId
    throw new Error('Not implemented')
  },
}
