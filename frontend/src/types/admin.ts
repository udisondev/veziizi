export interface AdminLoginRequest {
  email: string
  password: string
}

export interface AdminLoginResponse {
  admin_id: string
  email: string
  name: string
}

export interface PendingOrganization {
  id: string
  name: string
  inn: string
  legal_name: string
  country: 'RU' | 'KZ' | 'BY'
  email: string
  created_at: string
}

export interface CarrierProfile {
  description?: string
  vehicle_types?: string[]
  regions?: string[]
  has_adr: boolean
  has_refrigerator: boolean
}

export interface OrganizationMember {
  id: string
  email: string
  name: string
  phone: string
  role: 'owner' | 'administrator' | 'employee'
  status: 'active' | 'blocked'
  created_at: string
}

export interface OrganizationDetail {
  id: string
  name: string
  inn: string
  legal_name: string
  country: 'RU' | 'KZ' | 'BY'
  phone: string
  email: string
  address: string
  status: 'pending' | 'active' | 'suspended' | 'rejected'
  is_carrier: boolean
  carrier_profile: CarrierProfile | null
  members: OrganizationMember[]
  created_at: string
}

export interface RejectRequest {
  reason: string
}
