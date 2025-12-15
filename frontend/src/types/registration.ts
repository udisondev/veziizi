export type Country = 'RU' | 'KZ' | 'BY'

export interface OrganizationData {
  name: string
  inn: string
  legal_name: string
  country: Country
  phone: string
  email: string
  address: string
}

export interface OwnerData {
  owner_name: string
  owner_email: string
  owner_phone: string
  owner_password: string
  confirm_password: string
}

export interface RegisterRequest {
  name: string
  inn: string
  legal_name: string
  country: Country
  phone: string
  email: string
  address: string
  owner_email: string
  owner_password: string
  owner_name: string
  owner_phone: string
}

export interface RegisterResponse {
  organization_id: string
  member_id: string
}
