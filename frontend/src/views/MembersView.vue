<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import { vMaska } from 'maska/vue'
import { membersApi } from '@/api/members'
import { freightRequestsApi } from '@/api/freightRequests'
import { invitationsApi } from '@/api/invitations'
import { historyApi } from '@/api/history'
import type { MemberListItem, MemberRole, MemberRoleFilter, MemberStatus, MemberStatusFilter } from '@/types/member'
import type { InvitationListItem, InvitationStatus, InvitationRole } from '@/types/invitation'
import {
  roleLabels,
  statusLabels,
  roleOptions,
  statusOptions,
} from '@/types/member'
import EventHistory from '@/components/EventHistory.vue'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { Tabs, TabsContent } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

// Shared Components
import {
  PageHeader,
  StatusBadge,
  LoadingSpinner,
  EmptyState,
  ErrorBanner,
  ConfirmDialog,
  TabsDropdown,
  FilterSheet,
  type TabItem,
} from '@/components/shared'

// Icons
import {
  Users,
  Mail,
  Plus,
  Copy,
  Check,
  Clock,
  UserPlus,
  History,
  AlertCircle,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { canManageInvitations, canViewHistory } = usePermissions()

// Tabs
const currentTab = ref('members')

// History loader
function loadOrganizationHistory(limit: number, offset: number) {
  if (!auth.organizationId) {
    return Promise.resolve({ items: [], total: 0 })
  }
  return historyApi.getOrganizationHistory(auth.organizationId, { limit, offset })
}

// Selection mode (for reassigning freight request responsible)
const isSelectionMode = computed(() => route.query.selectFor === 'freightRequest')
const freightRequestId = computed(() => route.query.frId as string | undefined)
const selectLoading = ref(false)

// Data
const members = ref<MemberListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters
const showFilters = ref(false)
const searchQuery = ref('')
const roleFilter = ref<MemberRoleFilter>('all')
const statusFilter = ref<MemberStatusFilter>('all')

// Temp filters for sheet
const tempSearch = ref('')
const tempRole = ref<MemberRoleFilter>('all')
const tempStatus = ref<MemberStatusFilter>('all')

// Invitations data
const invitations = ref<InvitationListItem[]>([])
const isLoadingInvitations = ref(false)

// Invitations filters
type InvitationStatusFilter = InvitationStatus | 'all'
const invitationsStatusFilter = ref<InvitationStatusFilter>('all')
const tempInvitationsStatus = ref<InvitationStatusFilter>('all')

const invitationStatusOptions: { value: InvitationStatusFilter, label: string }[] = [
  { value: 'all', label: 'Все статусы' },
  { value: 'pending', label: 'Ожидают' },
  { value: 'accepted', label: 'Приняты' },
  { value: 'expired', label: 'Истекли' },
  { value: 'cancelled', label: 'Отменены' },
]

const invitationRoleOptions: { value: InvitationRole, label: string }[] = [
  { value: 'employee', label: 'Сотрудник' },
  { value: 'administrator', label: 'Администратор' },
]

// Invitation form
const showInvitationForm = ref(false)
const isSubmitting = ref(false)
const formError = ref<string | null>(null)
const createdToken = ref<string | null>(null)
const copied = ref(false)
const phoneMask = '+7 (###) ###-##-##'
const phonePlaceholder = '+7 (999) 999-99-99'

const invitationForm = ref({
  email: '',
  role: 'employee' as InvitationRole,
  name: '',
  phone: '',
})

// Cancel invitation
const cancellingId = ref<string | null>(null)
const showCancelModal = ref(false)
const cancellingInvitation = ref<InvitationListItem | null>(null)
const cancelError = ref<string | null>(null)

// Status maps for StatusBadge
const memberStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  active: { label: 'Активен', variant: 'success' },
  inactive: { label: 'Неактивен', variant: 'secondary' },
  blocked: { label: 'Заблокирован', variant: 'destructive' },
}

const invitationStatusMap: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'info' | 'secondary' }> = {
  pending: { label: 'Ожидает', variant: 'warning' },
  accepted: { label: 'Принято', variant: 'success' },
  expired: { label: 'Истекло', variant: 'secondary' },
  cancelled: { label: 'Отменено', variant: 'destructive' },
}

// Computed
const filteredMembers = computed(() => {
  let result = members.value

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(
      (m) =>
        m.name.toLowerCase().includes(q) ||
        m.email.toLowerCase().includes(q) ||
        (m.phone && m.phone.includes(q))
    )
  }

  if (roleFilter.value !== 'all') {
    result = result.filter((m) => m.role === roleFilter.value)
  }

  if (statusFilter.value !== 'all') {
    result = result.filter((m) => m.status === statusFilter.value)
  }

  // Sort: owner first, then by date
  return result.sort((a, b) => {
    if (a.role === 'owner') return -1
    if (b.role === 'owner') return 1
    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  })
})

const hasActiveMembersFilters = computed(
  () => searchQuery.value.trim() !== '' || roleFilter.value !== 'all' || statusFilter.value !== 'all'
)

const hasActiveInvitationsFilters = computed(
  () => invitationsStatusFilter.value !== 'all'
)

const hasActiveFilters = computed(() => {
  if (currentTab.value === 'members') {
    return hasActiveMembersFilters.value
  }
  return hasActiveInvitationsFilters.value
})

const activeFiltersCount = computed(() => {
  if (currentTab.value === 'members') {
    let count = 0
    if (searchQuery.value.trim()) count++
    if (roleFilter.value !== 'all') count++
    if (statusFilter.value !== 'all') count++
    return count
  }
  return invitationsStatusFilter.value !== 'all' ? 1 : 0
})

const tabItems = computed((): TabItem[] => {
  const items: TabItem[] = [
    { value: 'members', label: 'Сотрудники', icon: Users },
  ]
  if (canManageInvitations.value) {
    items.push({ value: 'invitations', label: 'Приглашения', icon: UserPlus })
  }
  if (canViewHistory.value) {
    items.push({ value: 'history', label: 'История', icon: History, separator: true })
  }
  return items
})

// Load data
async function loadMembers() {
  if (!auth.organizationId) return

  isLoading.value = true
  error.value = null

  try {
    members.value = await membersApi.listByOrganization(auth.organizationId)
  } catch (e) {
    error.value = 'Не удалось загрузить список сотрудников'
    console.error(e)
  } finally {
    isLoading.value = false
  }
}

async function loadInvitations() {
  if (!auth.organizationId) return

  isLoadingInvitations.value = true
  error.value = null

  try {
    const status = invitationsStatusFilter.value !== 'all' ? invitationsStatusFilter.value as InvitationStatus : undefined
    const response = await invitationsApi.list(auth.organizationId, status)
    invitations.value = response.items ?? []
  } catch (e) {
    error.value = 'Не удалось загрузить приглашения'
    console.error(e)
  } finally {
    isLoadingInvitations.value = false
  }
}

// Invitation CRUD
async function createInvitation() {
  if (!auth.organizationId) return

  isSubmitting.value = true
  formError.value = null
  createdToken.value = null

  try {
    const response = await invitationsApi.create(auth.organizationId, {
      email: invitationForm.value.email,
      role: invitationForm.value.role,
      name: invitationForm.value.name || undefined,
      phone: invitationForm.value.phone || undefined,
    })

    createdToken.value = response.token
    invitationForm.value = { email: '', role: 'employee', name: '', phone: '' }
    await loadInvitations()
  } catch (e: any) {
    formError.value = e?.message || 'Не удалось создать приглашение'
    console.error(e)
  } finally {
    isSubmitting.value = false
  }
}

function closeInvitationForm() {
  showInvitationForm.value = false
  createdToken.value = null
  formError.value = null
  copied.value = false
}

function openCancelModal(item: InvitationListItem) {
  cancellingInvitation.value = item
  cancelError.value = null
  showCancelModal.value = true
}

function closeCancelModal() {
  showCancelModal.value = false
  cancellingInvitation.value = null
  cancelError.value = null
}

async function confirmCancel() {
  if (!auth.organizationId || !cancellingInvitation.value) return

  cancellingId.value = cancellingInvitation.value.id
  cancelError.value = null

  try {
    await invitationsApi.cancel(auth.organizationId, cancellingInvitation.value.id)
    const item = invitations.value.find(i => i.id === cancellingInvitation.value!.id)
    if (item) {
      item.status = 'cancelled'
    }
    closeCancelModal()
  } catch (e: any) {
    console.error(e)
    cancelError.value = e?.message || 'Не удалось отменить приглашение'
  } finally {
    cancellingId.value = null
  }
}

// Helpers
function getInvitationRoleLabel(role: string): string {
  switch (role) {
    case 'employee': return 'Сотрудник'
    case 'administrator': return 'Администратор'
    default: return role
  }
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function getInvitationUrl(token: string): string {
  return `${window.location.origin}/invitations/${token}`
}

async function copyToClipboard(text: string) {
  await navigator.clipboard.writeText(text)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

// Filter functions
function openFilters() {
  if (currentTab.value === 'members') {
    tempSearch.value = searchQuery.value
    tempRole.value = roleFilter.value
    tempStatus.value = statusFilter.value
  } else {
    tempInvitationsStatus.value = invitationsStatusFilter.value
  }
  showFilters.value = true
}

function applyFilters() {
  if (currentTab.value === 'members') {
    searchQuery.value = tempSearch.value
    roleFilter.value = tempRole.value
    statusFilter.value = tempStatus.value
  } else {
    invitationsStatusFilter.value = tempInvitationsStatus.value
    loadInvitations()
  }
  showFilters.value = false
}

function resetFilters() {
  if (currentTab.value === 'members') {
    tempSearch.value = ''
    tempRole.value = 'all'
    tempStatus.value = 'all'
  } else {
    tempInvitationsStatus.value = 'all'
  }
}

function resetAllFilters() {
  if (currentTab.value === 'members') {
    searchQuery.value = ''
    roleFilter.value = 'all'
    statusFilter.value = 'all'
  } else {
    invitationsStatusFilter.value = 'all'
    loadInvitations()
  }
}

// Navigation to member detail
function goToMember(member: MemberListItem) {
  if (isSelectionMode.value) return
  router.push({ name: 'member-profile', params: { id: member.id } })
}

// Selection mode handlers
async function selectMember(member: MemberListItem) {
  if (!isSelectionMode.value || !freightRequestId.value) return
  if (member.status !== 'active') return

  selectLoading.value = true
  error.value = null

  try {
    await freightRequestsApi.reassign(freightRequestId.value, member.id)
    router.push(`/freight-requests/${freightRequestId.value}`)
  } catch (e: any) {
    error.value = e?.message || 'Не удалось назначить ответственного'
  } finally {
    selectLoading.value = false
  }
}

function cancelSelection() {
  if (freightRequestId.value) {
    router.push(`/freight-requests/${freightRequestId.value}`)
  } else {
    router.back()
  }
}

function getRoleBadgeVariant(role: MemberRole): 'default' | 'secondary' | 'outline' {
  if (role === 'owner') return 'default'
  if (role === 'administrator') return 'secondary'
  return 'outline'
}

onMounted(() => {
  loadMembers()
  if (canManageInvitations.value) {
    loadInvitations()
  }
})

watch(currentTab, (tab) => {
  if (tab === 'invitations' && invitations.value.length === 0 && canManageInvitations.value) {
    loadInvitations()
  }
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Header -->
    <PageHeader
      :title="isSelectionMode ? 'Выберите ответственного' : 'Штат'"
      class="mb-6"
    >
      <template #description v-if="isSelectionMode">
        Нажмите на сотрудника для назначения
      </template>
      <template #actions>
        <Button
          v-if="isSelectionMode"
          variant="outline"
          @click="cancelSelection"
        >
          Отмена
        </Button>
        <Button
          v-if="canManageInvitations && !isSelectionMode"
          @click="showInvitationForm = true"
        >
          <Plus class="mr-2 h-4 w-4" />
          Пригласить
        </Button>

        <!-- Filters Sheet -->
        <FilterSheet
          v-model:open="showFilters"
          :active-filters-count="activeFiltersCount"
          :description="currentTab === 'members' ? 'Фильтрация сотрудников' : 'Фильтрация приглашений'"
          @open="openFilters"
          @apply="applyFilters"
          @reset="resetFilters"
        >
          <!-- Members filters -->
          <template v-if="currentTab === 'members'">
            <div class="space-y-2">
              <Label>Поиск</Label>
              <Input
                v-model="tempSearch"
                placeholder="ФИО, email или телефон"
              />
            </div>

            <div class="space-y-2">
              <Label>Роль</Label>
              <Select v-model="tempRole">
                <SelectTrigger>
                  <SelectValue placeholder="Все роли" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="opt in roleOptions"
                    :key="opt.value"
                    :value="opt.value"
                  >
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <Label>Статус</Label>
              <Select v-model="tempStatus">
                <SelectTrigger>
                  <SelectValue placeholder="Все статусы" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="opt in statusOptions"
                    :key="opt.value"
                    :value="opt.value"
                  >
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </template>

          <!-- Invitations filters -->
          <template v-else>
            <div class="space-y-2">
              <Label>Статус</Label>
              <Select v-model="tempInvitationsStatus">
                <SelectTrigger>
                  <SelectValue placeholder="Все статусы" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="opt in invitationStatusOptions"
                    :key="opt.value"
                    :value="opt.value"
                  >
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </template>
        </FilterSheet>
      </template>
    </PageHeader>

    <!-- Active filters indicator -->
    <Card v-if="hasActiveFilters" class="mb-6 border-primary/20 bg-primary/5">
      <CardContent class="flex items-center justify-between py-3">
        <div class="text-sm text-primary flex flex-wrap gap-x-2 gap-y-1">
          <template v-if="currentTab === 'members'">
            <span v-if="searchQuery">Поиск: "{{ searchQuery }}"</span>
            <span v-if="searchQuery && (roleFilter !== 'all' || statusFilter !== 'all')">, </span>
            <span v-if="roleFilter !== 'all'">Роль: {{ roleLabels[roleFilter as MemberRole] }}</span>
            <span v-if="roleFilter !== 'all' && statusFilter !== 'all'">, </span>
            <span v-if="statusFilter !== 'all'">Статус: {{ statusLabels[statusFilter as MemberStatus] }}</span>
          </template>
          <template v-else>
            <span>Статус: {{ invitationStatusOptions.find(o => o.value === invitationsStatusFilter)?.label }}</span>
          </template>
        </div>
        <Button variant="ghost" size="sm" @click="resetAllFilters">
          Сбросить
        </Button>
      </CardContent>
    </Card>

    <!-- Tabs -->
    <Tabs v-if="(canManageInvitations || canViewHistory) && !isSelectionMode" v-model="currentTab" class="space-y-6">
      <!-- Tab selector dropdown -->
      <TabsDropdown v-model="currentTab" :items="tabItems" />

      <!-- Members Tab -->
      <TabsContent value="members">
        <LoadingSpinner v-if="isLoading" text="Загрузка сотрудников..." />

        <ErrorBanner
          v-else-if="error"
          :message="error"
          @retry="loadMembers"
        />

        <EmptyState
          v-else-if="filteredMembers.length === 0"
          :icon="Users"
          :title="hasActiveFilters ? 'Нет сотрудников по фильтрам' : 'Сотрудников пока нет'"
          :description="hasActiveFilters ? 'Попробуйте изменить параметры фильтрации' : 'Пригласите первого сотрудника'"
        />

        <template v-else>
          <!-- Mobile Cards -->
          <div class="sm:hidden space-y-3">
            <Card
              v-for="member in filteredMembers"
              :key="member.id"
              :class="cn(
                'cursor-pointer transition-shadow',
                isSelectionMode && member.status === 'active' && 'hover:shadow-md',
                isSelectionMode && member.status !== 'active' && 'opacity-50 cursor-not-allowed',
                !isSelectionMode && 'hover:shadow-md',
                selectLoading && 'pointer-events-none'
              )"
              @click="isSelectionMode ? selectMember(member) : goToMember(member)"
            >
              <CardContent class="p-4">
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0 flex-1">
                    <div class="font-medium text-foreground truncate">
                      {{ member.name }}
                      <span v-if="member.id === auth.memberId" class="text-xs text-muted-foreground">(вы)</span>
                    </div>
                    <div class="text-sm text-muted-foreground truncate">{{ member.email }}</div>
                    <div v-if="member.phone" class="text-sm text-muted-foreground">{{ member.phone }}</div>
                  </div>
                  <div class="flex flex-col items-end gap-1">
                    <Badge :variant="getRoleBadgeVariant(member.role)">
                      {{ roleLabels[member.role] }}
                    </Badge>
                    <StatusBadge :status="member.status" :status-map="memberStatusMap" />
                  </div>
                </div>
                <div class="mt-2 text-xs text-muted-foreground">
                  Добавлен {{ formatDate(member.created_at) }}
                </div>
              </CardContent>
            </Card>
          </div>

          <!-- Desktop Table -->
          <Card class="hidden sm:block">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Дата</TableHead>
                  <TableHead>ФИО</TableHead>
                  <TableHead>Телефон</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Роль</TableHead>
                  <TableHead>Статус</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="member in filteredMembers"
                  :key="member.id"
                  :class="cn(
                    'cursor-pointer',
                    isSelectionMode && member.status === 'active' && 'hover:bg-primary/5',
                    isSelectionMode && member.status !== 'active' && 'opacity-50 cursor-not-allowed',
                    selectLoading && 'pointer-events-none'
                  )"
                  @click="isSelectionMode ? selectMember(member) : goToMember(member)"
                >
                  <TableCell class="text-muted-foreground">
                    {{ formatDate(member.created_at) }}
                  </TableCell>
                  <TableCell>
                    <div class="font-medium">{{ member.name }}</div>
                    <div v-if="member.id === auth.memberId" class="text-xs text-muted-foreground">
                      (это вы)
                    </div>
                  </TableCell>
                  <TableCell class="text-muted-foreground">
                    {{ member.phone || '—' }}
                  </TableCell>
                  <TableCell class="text-muted-foreground">
                    {{ member.email }}
                  </TableCell>
                  <TableCell>
                    <Badge :variant="getRoleBadgeVariant(member.role)">
                      {{ roleLabels[member.role] }}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge :status="member.status" :status-map="memberStatusMap" />
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </Card>
        </template>
      </TabsContent>

      <!-- Invitations Tab -->
      <TabsContent value="invitations">
        <LoadingSpinner v-if="isLoadingInvitations" text="Загрузка приглашений..." />

        <ErrorBanner
          v-else-if="error"
          :message="error"
          @retry="loadInvitations"
        />

        <EmptyState
          v-else-if="invitations.length === 0"
          :icon="Mail"
          :title="hasActiveFilters ? 'Нет приглашений по фильтрам' : 'Приглашений пока нет'"
          :description="hasActiveFilters ? 'Попробуйте изменить параметры фильтрации' : 'Создайте приглашение для нового сотрудника'"
        >
          <template #action>
            <Button @click="showInvitationForm = true">
              <Plus class="mr-2 h-4 w-4" />
              Создать приглашение
            </Button>
          </template>
        </EmptyState>

        <template v-else>
          <!-- Mobile Cards -->
          <div class="sm:hidden space-y-3">
            <Card v-for="item in invitations" :key="item.id">
              <CardContent class="p-4">
                <div class="flex items-start justify-between gap-2 mb-2">
                  <div class="min-w-0 flex-1">
                    <div class="font-medium text-foreground truncate">{{ item.email }}</div>
                    <div v-if="item.name" class="text-sm text-muted-foreground truncate">{{ item.name }}</div>
                    <div v-if="item.phone" class="text-sm text-muted-foreground">{{ item.phone }}</div>
                  </div>
                  <div class="flex flex-col items-end gap-1">
                    <Badge variant="outline">
                      {{ getInvitationRoleLabel(item.role) }}
                    </Badge>
                    <StatusBadge :status="item.status" :status-map="invitationStatusMap" />
                  </div>
                </div>
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock class="h-3 w-3" />
                    {{ formatDateTime(item.expires_at) }}
                  </div>
                  <Button
                    v-if="item.status === 'pending'"
                    variant="ghost"
                    size="sm"
                    class="text-destructive hover:text-destructive"
                    @click="openCancelModal(item)"
                  >
                    Отменить
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

          <!-- Desktop Table -->
          <Card class="hidden sm:block">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Роль</TableHead>
                  <TableHead>ФИО</TableHead>
                  <TableHead>Телефон</TableHead>
                  <TableHead>Статус</TableHead>
                  <TableHead>Истекает</TableHead>
                  <TableHead>Действия</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="item in invitations" :key="item.id">
                  <TableCell class="font-medium">{{ item.email }}</TableCell>
                  <TableCell>{{ getInvitationRoleLabel(item.role) }}</TableCell>
                  <TableCell class="text-muted-foreground">{{ item.name || '—' }}</TableCell>
                  <TableCell class="text-muted-foreground">{{ item.phone || '—' }}</TableCell>
                  <TableCell>
                    <StatusBadge :status="item.status" :status-map="invitationStatusMap" />
                  </TableCell>
                  <TableCell class="text-muted-foreground">
                    {{ formatDateTime(item.expires_at) }}
                  </TableCell>
                  <TableCell>
                    <Button
                      v-if="item.status === 'pending'"
                      variant="ghost"
                      size="sm"
                      class="text-destructive hover:text-destructive"
                      @click="openCancelModal(item)"
                    >
                      Отменить
                    </Button>
                    <span v-else class="text-muted-foreground">—</span>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </Card>
        </template>
      </TabsContent>

      <!-- History Tab -->
      <TabsContent value="history">
        <Card>
          <CardContent class="p-6">
            <EventHistory :load-fn="loadOrganizationHistory" />
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>

    <!-- Members list when no tabs (selection mode or no permissions) -->
    <template v-if="(!canManageInvitations && !canViewHistory) || isSelectionMode">
      <LoadingSpinner v-if="isLoading" text="Загрузка сотрудников..." />

      <ErrorBanner
        v-else-if="error"
        :message="error"
        @retry="loadMembers"
      />

      <EmptyState
        v-else-if="filteredMembers.length === 0"
        :icon="Users"
        :title="hasActiveFilters ? 'Нет сотрудников по фильтрам' : 'Сотрудников пока нет'"
        :description="hasActiveFilters ? 'Попробуйте изменить параметры фильтрации' : ''"
      />

      <div v-else class="space-y-3">
        <Card
          v-for="member in filteredMembers"
          :key="member.id"
          :class="cn(
            'cursor-pointer transition-shadow',
            isSelectionMode && member.status === 'active' && 'hover:shadow-md',
            isSelectionMode && member.status !== 'active' && 'opacity-50 cursor-not-allowed',
            !isSelectionMode && 'hover:shadow-md',
            selectLoading && 'pointer-events-none'
          )"
          @click="isSelectionMode ? selectMember(member) : goToMember(member)"
        >
          <CardContent class="p-4">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <div class="font-medium text-foreground truncate">
                  {{ member.name }}
                  <span v-if="member.id === auth.memberId" class="text-xs text-muted-foreground">(вы)</span>
                </div>
                <div class="text-sm text-muted-foreground truncate">{{ member.email }}</div>
              </div>
              <div class="flex flex-col items-end gap-1">
                <Badge :variant="getRoleBadgeVariant(member.role)">
                  {{ roleLabels[member.role] }}
                </Badge>
                <StatusBadge :status="member.status" :status-map="memberStatusMap" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>

    <!-- Create Invitation Dialog -->
    <Dialog v-model:open="showInvitationForm">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Новое приглашение</DialogTitle>
          <DialogDescription>
            Отправьте приглашение новому сотруднику
          </DialogDescription>
        </DialogHeader>

        <!-- Success state with token -->
        <div v-if="createdToken" class="space-y-4">
          <div class="rounded-lg border border-success/50 bg-success/10 p-4">
            <p class="text-success font-medium mb-2">Приглашение создано!</p>
            <p class="text-sm text-muted-foreground mb-2">Ссылка для приглашения:</p>
            <div class="flex items-center gap-2">
              <Input
                :model-value="getInvitationUrl(createdToken)"
                readonly
                class="flex-1 text-sm"
              />
              <Button
                size="icon"
                variant="outline"
                @click="copyToClipboard(getInvitationUrl(createdToken))"
              >
                <Check v-if="copied" class="h-4 w-4 text-success" />
                <Copy v-else class="h-4 w-4" />
              </Button>
            </div>
          </div>
          <Button variant="outline" class="w-full" @click="closeInvitationForm">
            Закрыть
          </Button>
        </div>

        <!-- Form -->
        <form v-else @submit.prevent="createInvitation" class="space-y-4">
          <div
            v-if="formError"
            class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
          >
            <AlertCircle class="h-4 w-4 shrink-0" />
            {{ formError }}
          </div>

          <div class="space-y-2">
            <Label for="inv-email">
              Email <span class="text-destructive">*</span>
            </Label>
            <Input
              id="inv-email"
              v-model="invitationForm.email"
              type="email"
              required
              placeholder="user@example.com"
            />
          </div>

          <div class="space-y-2">
            <Label for="inv-role">
              Роль <span class="text-destructive">*</span>
            </Label>
            <Select v-model="invitationForm.role">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in invitationRoleOptions"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="space-y-2">
            <Label for="inv-name">
              ФИО
              <span class="text-muted-foreground font-normal">(опционально)</span>
            </Label>
            <Input
              id="inv-name"
              v-model="invitationForm.name"
              placeholder="Иванов Иван Иванович"
            />
            <p class="text-xs text-muted-foreground">
              Если заполнить, приглашённый не сможет изменить
            </p>
          </div>

          <div class="space-y-2">
            <Label for="inv-phone">
              Телефон
              <span class="text-muted-foreground font-normal">(опционально)</span>
            </Label>
            <Input
              id="inv-phone"
              v-model="invitationForm.phone"
              v-maska
              :data-maska="phoneMask"
              type="tel"
              inputmode="tel"
              :placeholder="phonePlaceholder"
            />
            <p class="text-xs text-muted-foreground">
              Если заполнить, приглашённый не сможет изменить
            </p>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" @click="closeInvitationForm">
              Отмена
            </Button>
            <Button type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Создание...' : 'Создать' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Cancel Invitation Dialog -->
    <ConfirmDialog
      :open="showCancelModal"
      title="Отменить приглашение?"
      :description="`Вы уверены, что хотите отменить приглашение для ${cancellingInvitation?.email}? Пользователь не сможет принять это приглашение.`"
      confirm-text="Отменить приглашение"
      confirm-variant="destructive"
      :loading="cancellingId !== null"
      @confirm="confirmCancel"
      @cancel="closeCancelModal"
    />
  </div>
</template>
