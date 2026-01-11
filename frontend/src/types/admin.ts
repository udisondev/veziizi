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
  status: string
  activation_date?: string
  created_at: string
}

export interface OrganizationReviewsResponse {
  items: OrganizationReview[]
  total: number
}

// Review moderation types
export interface FraudSignal {
  type: string
  severity: 'low' | 'medium' | 'high'
  description: string
  score_impact: number
}

export interface PendingReview {
  id: string
  order_id: string
  reviewer_org_id: string
  reviewed_org_id: string
  rating: number
  comment: string
  order_amount: number
  order_currency: string
  raw_weight: number
  fraud_score: number
  fraud_signals: FraudSignal[]
  activation_date?: string
  created_at: string
  analyzed_at?: string
}

export interface PendingReviewsResponse {
  reviews: PendingReview[]
  total: number
}

export interface ApproveReviewRequest {
  final_weight: number
  note?: string
}

export interface RejectReviewRequest {
  reason: string
}

// Fraudster types
export interface Fraudster {
  org_id: string
  org_name: string
  is_confirmed: boolean
  is_suspected: boolean
  marked_at: string
  marked_by: string
  reason: string
  total_reviews_left: number
  deactivated_reviews: number
  reputation_score: number
}

export interface FraudstersResponse {
  fraudsters: Fraudster[]
  total: number
}

export interface MarkFraudsterRequest {
  is_confirmed: boolean
  reason: string
}

export interface UnmarkFraudsterRequest {
  reason: string
}
