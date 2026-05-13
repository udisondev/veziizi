import { api } from './client'
import type { Vehicle, CreateVehicleRequest, UpdateVehicleRequest } from '@/types/vehicle'

export const vehiclesApi = {
  async list(organizationId: string): Promise<Vehicle[]> {
    const result = await api.get<{ items: Vehicle[] }>(`/organizations/${organizationId}/vehicles`)
    return result?.items ?? []
  },

  async get(vehicleId: string): Promise<Vehicle> {
    return await api.get<Vehicle>(`/vehicles/${vehicleId}`)
  },

  async create(organizationId: string, data: CreateVehicleRequest): Promise<void> {
    await api.post(`/organizations/${organizationId}/vehicles`, data)
  },

  async update(organizationId: string, vehicleId: string, data: UpdateVehicleRequest): Promise<void> {
    await api.patch(`/organizations/${organizationId}/vehicles/${vehicleId}`, data)
  },

  async remove(organizationId: string, vehicleId: string): Promise<void> {
    await api.delete(`/organizations/${organizationId}/vehicles/${vehicleId}`)
  },
}
