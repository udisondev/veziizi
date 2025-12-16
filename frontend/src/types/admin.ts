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
  members: OrganizationMember[]
  created_at: string
}

export interface RejectRequest {
  reason: string
}

// Organization rating types
export interface OrganizationRating {
  total_reviews: number
  average_rating: number
}

export interface OrganizationReview {
  id: string
  order_id: string
  reviewer_org_id: string
  reviewer_org_name: string
  rating: number
  comment: string
  created_at: string
}

export interface OrganizationReviewsResponse {
  items: OrganizationReview[]
  total: number
}
