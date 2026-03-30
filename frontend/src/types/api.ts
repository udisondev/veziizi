export type MemberRole = 'owner' | 'administrator' | 'employee'

export type MemberStatus = 'active' | 'blocked'

export type OrganizationStatus = 'pending' | 'active' | 'rejected' | 'suspended'

export interface OrganizationBrief {
  name: string
  status: OrganizationStatus
}

export interface MeResponse {
  member_id: string
  organization_id: string
  role: MemberRole
  email: string
  name: string
  phone?: string
  telegram_id?: number
  status: MemberStatus
  organization?: OrganizationBrief
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  member_id: string
  organization_id: string
  email: string
  name: string
  role: MemberRole
}

export interface ApiError {
  error: string
}
