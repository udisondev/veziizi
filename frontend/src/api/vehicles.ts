import { api } from './client'
import type { Vehicle, VehicleListItem, VehicleListParams, CreateVehicleRequest, UpdateVehicleRequest } from '@/types/vehicle'
import type { CursorPaginatedResponse } from './freightRequests'

export const vehiclesApi = {
  async listAll(params?: VehicleListParams): Promise<CursorPaginatedResponse<VehicleListItem>> {
    const p = new URLSearchParams()
    if (params?.vehicle_subtypes) p.set('vehicle_subtypes', params.vehicle_subtypes)
    if (params?.min_capacity !== undefined) p.set('min_capacity', params.min_capacity.toString())
    if (params?.max_capacity !== undefined) p.set('max_capacity', params.max_capacity.toString())
    if (params?.min_volume !== undefined) p.set('min_volume', params.min_volume.toString())
    if (params?.max_volume !== undefined) p.set('max_volume', params.max_volume.toString())
    if (params?.loading_types) p.set('loading_types', params.loading_types)
    if (params?.requires_adr) p.set('requires_adr', 'true')
    if (params?.thermograph) p.set('thermograph', 'true')
    if (params?.org_name) p.set('org_name', params.org_name)
    if (params?.limit) p.set('limit', params.limit.toString())
    if (params?.cursor) p.set('cursor', params.cursor)
    const query = p.toString()
    const result = await api.get<CursorPaginatedResponse<VehicleListItem> | null>(`/vehicles${query ? `?${query}` : ''}`)
    return result ?? { items: [], has_more: false }
  },

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
