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

export interface OrganizationStats {
  total_freight_requests: number
  active_freight_requests: number
  completed_deals: number
  total_offers_made: number
  successful_offers: number
}

export interface DashboardStats {
  as_customer_published: number
  as_customer_selected: number
  as_customer_confirmed: number
  as_carrier_confirmed: number
  as_carrier_partially_completed: number
  pending_offers_count: number
  pending_offers_today: number
  market_published_today: number
}

export interface PendingOfferItem {
  id: string
  freight_request_id: string
  request_number: number
  carrier_org_id: string
  carrier_org_name: string
  origin_address: string
  destination_address: string
  price_amount?: number | null
  price_currency?: string | null
  created_at: string
  offers_count: number
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

// Email Template types
export interface VariableSpec {
  type: string
  required: boolean
  description?: string
}

export interface EmailTemplate {
  id: string
  slug: string
  name: string
  subject: string
  body_html: string
  body_text: string
  category: 'transactional' | 'marketing'
  variables_schema: Record<string, VariableSpec>
  is_system: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface EmailTemplatesListResponse {
  templates: EmailTemplate[]
  total: number
}

export interface EmailTemplateListFilter {
  category?: 'transactional' | 'marketing'
  is_active?: boolean
  is_system?: boolean
  search?: string
  limit?: number
  offset?: number
}

export interface CreateEmailTemplateRequest {
  slug: string
  name: string
  subject: string
  body_html: string
  body_text: string
  category: 'transactional' | 'marketing'
  variables_schema?: Record<string, VariableSpec>
}

export interface UpdateEmailTemplateRequest {
  name?: string
  subject?: string
  body_html?: string
  body_text?: string
  category?: 'transactional' | 'marketing'
  variables_schema?: Record<string, VariableSpec>
  is_active?: boolean
}

export interface PreviewEmailTemplateRequest {
  template_id?: string
  subject: string
  body_html: string
  body_text: string
  variables: Record<string, string>
}

export interface PreviewEmailTemplateResponse {
  subject: string
  body_html: string
  body_text: string
}
