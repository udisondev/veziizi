/**
 * Централизованные маппинги статусов для StatusBadge и других компонентов
 * Все status maps определены здесь для избежания дублирования
 */

import type {
  FreightRequestStatus,
  OfferStatus,
} from '@/types/freightRequest'
import type { MemberStatus } from '@/types/member'
import type { InvitationStatus } from '@/types/invitation'
import type { OrganizationStatus } from '@/types/api'

// ============================================================================
// Тип для variant в StatusBadge
// ============================================================================

export type StatusVariant = 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' | 'outline'

export interface StatusMapEntry {
  label: string
  variant: StatusVariant
}

// ============================================================================
// FreightRequest Status Map
// ============================================================================

export const freightRequestStatusMap: Record<FreightRequestStatus, StatusMapEntry> = {
  published: { label: 'Опубликована', variant: 'success' },
  selected: { label: 'Выбран исполнитель', variant: 'warning' },
  confirmed: { label: 'Подтверждена', variant: 'info' },
  partially_completed: { label: 'Частично завершена', variant: 'info' },
  completed: { label: 'Завершена', variant: 'success' },
  cancelled: { label: 'Отменена', variant: 'destructive' },
  cancelled_after_confirmed: { label: 'Отменена после подтверждения', variant: 'destructive' },
  expired: { label: 'Истекла', variant: 'secondary' },
}

// ============================================================================
// Offer Status Map
// ============================================================================

export const offerStatusMap: Record<OfferStatus, StatusMapEntry> = {
  pending: { label: 'Ожидает', variant: 'secondary' },
  selected: { label: 'Выбран', variant: 'info' },
  confirmed: { label: 'Подтверждён', variant: 'success' },
  rejected: { label: 'Отклонён', variant: 'destructive' },
  withdrawn: { label: 'Отозван', variant: 'secondary' },
  declined: { label: 'Отказ', variant: 'destructive' },
}

// ============================================================================
// Member Status Map
// ============================================================================

export const memberStatusMap: Record<MemberStatus, StatusMapEntry> = {
  active: { label: 'Активен', variant: 'success' },
  blocked: { label: 'Заблокирован', variant: 'destructive' },
}

// Расширенный map с inactive (для UI)
export const memberStatusMapExtended: Record<string, StatusMapEntry> = {
  active: { label: 'Активен', variant: 'success' },
  inactive: { label: 'Неактивен', variant: 'secondary' },
  blocked: { label: 'Заблокирован', variant: 'destructive' },
}

// ============================================================================
// Invitation Status Map
// ============================================================================

export const invitationStatusMap: Record<InvitationStatus, StatusMapEntry> = {
  pending: { label: 'Ожидает', variant: 'warning' },
  accepted: { label: 'Принято', variant: 'success' },
  expired: { label: 'Истекло', variant: 'secondary' },
  cancelled: { label: 'Отменено', variant: 'destructive' },
}

// ============================================================================
// Organization Status Map
// ============================================================================

export const organizationStatusMap: Record<OrganizationStatus, StatusMapEntry> = {
  pending: { label: 'На проверке', variant: 'warning' },
  active: { label: 'Активна', variant: 'success' },
  rejected: { label: 'Отклонена', variant: 'destructive' },
  suspended: { label: 'Приостановлена', variant: 'destructive' },
}

// ============================================================================
// Review Status Map (для admin)
// ============================================================================

export const reviewStatusMap: Record<string, StatusMapEntry> = {
  pending_fraud_check: { label: 'Проверка на фрод', variant: 'warning' },
  pending_moderation: { label: 'На модерации', variant: 'warning' },
  pending_activation: { label: 'Ожидает активации', variant: 'info' },
  active: { label: 'Активен', variant: 'success' },
  rejected: { label: 'Отклонён', variant: 'destructive' },
  deactivated: { label: 'Деактивирован', variant: 'secondary' },
}

// ============================================================================
// Universal Status Map (объединённый для StatusBadge default)
// ============================================================================

export const universalStatusMap: Record<string, StatusMapEntry> = {
  // FreightRequest
  ...freightRequestStatusMap,
  // Offer (перезаписывает некоторые)
  ...offerStatusMap,
  // Member
  ...memberStatusMapExtended,
  // Invitation
  ...invitationStatusMap,
  // Organization
  ...organizationStatusMap,
  // Review
  ...reviewStatusMap,
}

// ============================================================================
// Helper функции
// ============================================================================

/**
 * Получает информацию о статусе из map
 * Если статус не найден, возвращает fallback
 */
export function getStatusInfo(
  status: string,
  statusMap: Record<string, StatusMapEntry> = universalStatusMap
): StatusMapEntry {
  return statusMap[status] ?? { label: status, variant: 'default' }
}
