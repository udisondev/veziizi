export type MemberRole = 'owner' | 'administrator' | 'employee'
export type MemberStatus = 'active' | 'blocked'

export interface MemberListItem {
  id: string
  email: string
  name: string
  phone: string
  role: MemberRole
  status: MemberStatus
  created_at: string
}

export const roleLabels: Record<MemberRole, string> = {
  owner: 'Владелец',
  administrator: 'Администратор',
  employee: 'Сотрудник',
}

export const roleColors: Record<MemberRole, string> = {
  owner: 'bg-purple-100 text-purple-800',
  administrator: 'bg-blue-100 text-blue-800',
  employee: 'bg-gray-100 text-gray-800',
}

export const statusLabels: Record<MemberStatus, string> = {
  active: 'Активен',
  blocked: 'Заблокирован',
}

export const statusColors: Record<MemberStatus, string> = {
  active: 'bg-green-100 text-green-800',
  blocked: 'bg-red-100 text-red-800',
}

export const roleOptions: { value: MemberRole | '', label: string }[] = [
  { value: '', label: 'Все роли' },
  { value: 'owner', label: 'Владелец' },
  { value: 'administrator', label: 'Администратор' },
  { value: 'employee', label: 'Сотрудник' },
]

export const statusOptions: { value: MemberStatus | '', label: string }[] = [
  { value: '', label: 'Все статусы' },
  { value: 'active', label: 'Активен' },
  { value: 'blocked', label: 'Заблокирован' },
]

// Роли для изменения (без owner)
export const editableRoleOptions: { value: MemberRole, label: string }[] = [
  { value: 'administrator', label: 'Администратор' },
  { value: 'employee', label: 'Сотрудник' },
]
