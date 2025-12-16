// Роли для приглашений (employee или administrator, owner нельзя назначить)
export type InvitationRole = 'employee' | 'administrator'

// Статусы приглашений
export type InvitationStatus = 'pending' | 'accepted' | 'expired'

// Запрос на создание приглашения
export interface CreateInvitationRequest {
  email: string
  role: InvitationRole
  name?: string  // предзаполненное ФИО (опционально)
  phone?: string // предзаполненный телефон (опционально)
}

// Ответ на создание приглашения
export interface CreateInvitationResponse {
  invitation_id: string
  token: string // для ручного тестирования (пока нет отправки email)
}

// Запрос на принятие приглашения
export interface AcceptInvitationRequest {
  password: string
  name?: string  // опционально, если предзаполнено в приглашении
  phone?: string // опционально, если предзаполнено в приглашении
}

// Ответ на принятие приглашения
export interface AcceptInvitationResponse {
  organization_id: string
  member_id: string
}

// Данные приглашения для формы принятия
export interface InvitationDetails {
  id: string
  organization_id: string
  organization_name: string
  email: string
  role: string
  name?: string
  phone?: string
  status: InvitationStatus
  expires_at: string
}

// Элемент списка приглашений
export interface InvitationListItem {
  id: string
  email: string
  role: string
  name?: string
  phone?: string
  status: InvitationStatus
  expires_at: string
  created_at: string
}

// Ответ со списком приглашений
export interface InvitationListResponse {
  items: InvitationListItem[]
}
