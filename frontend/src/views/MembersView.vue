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
import type { MemberListItem, MemberRole, MemberStatus } from '@/types/member'
import type { InvitationListItem, InvitationStatus, InvitationRole } from '@/types/invitation'
import {
  roleLabels,
  roleColors,
  statusLabels,
  statusColors,
  roleOptions,
  statusOptions,
} from '@/types/member'
import EventHistory from '@/components/EventHistory.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { canManageInvitations, canViewHistory } = usePermissions()

// Tabs
type TabType = 'members' | 'invitations' | 'history'
const currentTab = ref<TabType>('members')

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
const searchQuery = ref('')
const roleFilter = ref<MemberRole | ''>('')
const statusFilter = ref<MemberStatus | ''>('')
const showFilterModal = ref(false)

// Temp filters for modal
const tempSearch = ref('')
const tempRole = ref<MemberRole | ''>('')
const tempStatus = ref<MemberStatus | ''>('')

// Invitations data
const invitations = ref<InvitationListItem[]>([])
const isLoadingInvitations = ref(false)

// Invitations filters
const invitationsStatusFilter = ref<InvitationStatus | ''>('')
const tempInvitationsStatus = ref<InvitationStatus | ''>('')

const invitationStatusOptions: { value: InvitationStatus | '', label: string }[] = [
  { value: '', label: 'Все статусы' },
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

  if (roleFilter.value) {
    result = result.filter((m) => m.role === roleFilter.value)
  }

  if (statusFilter.value) {
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
  () => searchQuery.value.trim() !== '' || roleFilter.value !== '' || statusFilter.value !== ''
)

const hasActiveInvitationsFilters = computed(
  () => invitationsStatusFilter.value !== ''
)

const hasActiveFilters = computed(() => {
  if (currentTab.value === 'members') {
    return hasActiveMembersFilters.value
  }
  return hasActiveInvitationsFilters.value
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
    const status = invitationsStatusFilter.value || undefined
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

// Invitation helpers
function getInvitationStatusLabel(status: InvitationStatus): string {
  switch (status) {
    case 'pending': return 'Ожидает'
    case 'accepted': return 'Принято'
    case 'expired': return 'Истекло'
    case 'cancelled': return 'Отменено'
    default: return status
  }
}

function getInvitationStatusColor(status: InvitationStatus): string {
  switch (status) {
    case 'pending': return 'bg-yellow-100 text-yellow-800'
    case 'accepted': return 'bg-green-100 text-green-800'
    case 'expired': return 'bg-gray-100 text-gray-800'
    case 'cancelled': return 'bg-red-100 text-red-800'
    default: return 'bg-gray-100 text-gray-800'
  }
}

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

function getInvitationUrl(token: string): string {
  return `${window.location.origin}/invitations/${token}`
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}

// Filter modal
function openFilterModal() {
  if (currentTab.value === 'members') {
    tempSearch.value = searchQuery.value
    tempRole.value = roleFilter.value
    tempStatus.value = statusFilter.value
  } else {
    tempInvitationsStatus.value = invitationsStatusFilter.value
  }
  showFilterModal.value = true
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
  showFilterModal.value = false
}

function clearFilters() {
  if (currentTab.value === 'members') {
    tempSearch.value = ''
    tempRole.value = ''
    tempStatus.value = ''
  } else {
    tempInvitationsStatus.value = ''
  }
}

function resetAllFilters() {
  if (currentTab.value === 'members') {
    searchQuery.value = ''
    roleFilter.value = ''
    statusFilter.value = ''
  } else {
    invitationsStatusFilter.value = ''
    loadInvitations()
  }
}

function closeFilterModal() {
  showFilterModal.value = false
}

// Helpers
function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
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
  <div class="min-h-screen bg-gray-100 p-4 sm:p-6">
    <div class="max-w-6xl mx-auto">
      <!-- Header -->
      <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4 mb-6">
        <div>
          <h1 class="text-xl sm:text-2xl font-bold">
            {{ isSelectionMode ? 'Выберите ответственного' : 'Штат' }}
          </h1>
          <p v-if="isSelectionMode" class="text-sm text-gray-500 mt-1">
            Нажмите на сотрудника для назначения
          </p>
        </div>
        <div class="flex gap-2">
          <button
            v-if="isSelectionMode"
            @click="cancelSelection"
            class="px-3 py-2 text-sm bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors"
          >
            Отмена
          </button>
          <button
            v-if="canManageInvitations && !isSelectionMode"
            @click="showInvitationForm = true"
            class="px-3 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors whitespace-nowrap"
          >
            + Пригласить
          </button>
          <button
            @click="openFilterModal"
            class="px-3 py-2 text-sm bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors flex items-center gap-2"
          >
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
              />
            </svg>
            Фильтры
            <span
              v-if="hasActiveFilters"
              class="w-2 h-2 bg-blue-600 rounded-full"
            ></span>
          </button>
        </div>
      </div>

      <!-- Tab switcher -->
      <div
        v-if="(canManageInvitations || canViewHistory) && !isSelectionMode"
        class="bg-white rounded-lg p-3 mb-6 flex gap-6"
      >
        <label class="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            name="tab"
            value="members"
            v-model="currentTab"
            class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
          />
          <span class="text-sm font-medium text-gray-700">Сотрудники</span>
        </label>
        <label v-if="canManageInvitations" class="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            name="tab"
            value="invitations"
            v-model="currentTab"
            class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
          />
          <span class="text-sm font-medium text-gray-700">Приглашения</span>
        </label>
        <label v-if="canViewHistory" class="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            name="tab"
            value="history"
            v-model="currentTab"
            class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
          />
          <span class="text-sm font-medium text-gray-700">История</span>
        </label>
      </div>

      <!-- Active filters indicator -->
      <div v-if="hasActiveFilters" class="bg-blue-50 border border-blue-200 rounded-lg p-3 mb-6 flex items-center justify-between">
        <div class="text-sm text-blue-700">
          <template v-if="currentTab === 'members'">
            <span v-if="searchQuery">Поиск: "{{ searchQuery }}"</span>
            <span v-if="searchQuery && (roleFilter || statusFilter)">, </span>
            <span v-if="roleFilter">Роль: {{ roleLabels[roleFilter] }}</span>
            <span v-if="roleFilter && statusFilter">, </span>
            <span v-if="statusFilter">Статус: {{ statusLabels[statusFilter] }}</span>
          </template>
          <template v-else>
            <span>Статус: {{ invitationStatusOptions.find(o => o.value === invitationsStatusFilter)?.label }}</span>
          </template>
        </div>
        <button
          @click="resetAllFilters"
          class="text-blue-600 hover:text-blue-800 text-sm underline"
        >
          Сбросить
        </button>
      </div>

      <!-- Members Tab Content -->
      <template v-if="currentTab === 'members'">
        <!-- Loading -->
        <div v-if="isLoading" class="text-center py-12">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
          <p class="mt-2 text-gray-600">Загрузка...</p>
        </div>

        <!-- Error -->
        <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {{ error }}
          <button @click="loadMembers" class="ml-2 underline">Повторить</button>
        </div>

        <!-- Empty state -->
        <div v-else-if="filteredMembers.length === 0" class="bg-white rounded-lg shadow p-12 text-center">
          <p class="text-gray-500">
            {{ hasActiveFilters ? 'Нет сотрудников по заданным фильтрам' : 'Сотрудников пока нет' }}
          </p>
        </div>

        <!-- Members List -->
        <template v-else>
        <!-- Mobile Cards (visible below sm) -->
        <div class="sm:hidden space-y-3">
          <div
            v-for="member in filteredMembers"
            :key="member.id"
            :class="[
              'bg-white rounded-lg shadow p-4 cursor-pointer',
              isSelectionMode && member.status === 'active' ? 'active:bg-blue-50' : '',
              isSelectionMode && member.status !== 'active' ? 'opacity-50 cursor-not-allowed' : 'hover:bg-gray-50',
              selectLoading ? 'pointer-events-none' : ''
            ]"
            @click="isSelectionMode ? selectMember(member) : goToMember(member)"
          >
            <div class="flex items-start justify-between mb-2">
              <div>
                <div class="font-medium text-gray-900">
                  {{ member.name }}
                  <span v-if="member.id === auth.memberId" class="text-xs text-gray-400">(вы)</span>
                </div>
                <div class="text-sm text-gray-500">{{ member.email }}</div>
                <div v-if="member.phone" class="text-sm text-gray-500">{{ member.phone }}</div>
              </div>
              <div class="flex flex-col items-end gap-1">
                <span :class="[roleColors[member.role], 'px-2 py-0.5 text-xs font-medium rounded-full']">
                  {{ roleLabels[member.role] }}
                </span>
                <span :class="[statusColors[member.status], 'px-2 py-0.5 text-xs font-medium rounded-full']">
                  {{ statusLabels[member.status] }}
                </span>
              </div>
            </div>
            <div class="text-xs text-gray-400">
              Добавлен {{ formatDate(member.created_at) }}
            </div>
          </div>
        </div>

        <!-- Desktop Table (visible on sm and above) -->
        <div class="hidden sm:block bg-white rounded-lg shadow overflow-hidden">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200">
              <thead class="bg-gray-50">
                <tr>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Дата
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    ФИО
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Телефон
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Email
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Роль
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Статус
                  </th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-gray-200">
                <tr
                  v-for="member in filteredMembers"
                  :key="member.id"
                  :class="[
                    'transition-colors cursor-pointer',
                    isSelectionMode && member.status === 'active' ? 'hover:bg-blue-50' : 'hover:bg-gray-50',
                    isSelectionMode && member.status !== 'active' ? 'opacity-50 cursor-not-allowed' : '',
                    selectLoading ? 'pointer-events-none' : ''
                  ]"
                  @click="isSelectionMode ? selectMember(member) : goToMember(member)"
                >
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ formatDate(member.created_at) }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <div class="text-sm font-medium text-gray-900">
                      {{ member.name }}
                    </div>
                    <div v-if="member.id === auth.memberId" class="text-xs text-gray-400">
                      (это вы)
                    </div>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ member.phone || '—' }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ member.email }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <span :class="[roleColors[member.role], 'px-2 py-1 text-xs font-medium rounded-full']">
                      {{ roleLabels[member.role] }}
                    </span>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <span :class="[statusColors[member.status], 'px-2 py-1 text-xs font-medium rounded-full']">
                      {{ statusLabels[member.status] }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        </template>
      </template>

      <!-- Invitations Tab Content -->
      <template v-else>
        <!-- Loading -->
        <div v-if="isLoadingInvitations" class="text-center py-12">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
          <p class="mt-2 text-gray-600">Загрузка...</p>
        </div>

        <!-- Error -->
        <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {{ error }}
          <button @click="loadInvitations" class="ml-2 underline">Повторить</button>
        </div>

        <!-- Empty state -->
        <div v-else-if="invitations.length === 0" class="bg-white rounded-lg shadow p-12 text-center">
          <p class="text-gray-500">
            {{ hasActiveFilters ? 'Нет приглашений по заданным фильтрам' : 'Приглашений пока нет' }}
          </p>
        </div>

        <!-- Invitations List -->
        <template v-else>
          <!-- Mobile Cards (visible below sm) -->
          <div class="sm:hidden space-y-3">
            <div
              v-for="item in invitations"
              :key="item.id"
              class="bg-white rounded-lg shadow p-4"
            >
              <div class="flex items-start justify-between mb-2">
                <div>
                  <div class="font-medium text-gray-900">{{ item.email }}</div>
                  <div v-if="item.name" class="text-sm text-gray-500">{{ item.name }}</div>
                  <div v-if="item.phone" class="text-sm text-gray-500">{{ item.phone }}</div>
                </div>
                <div class="flex flex-col items-end gap-1">
                  <span class="px-2 py-0.5 text-xs font-medium rounded-full bg-gray-100 text-gray-800">
                    {{ getInvitationRoleLabel(item.role) }}
                  </span>
                  <span :class="[getInvitationStatusColor(item.status), 'px-2 py-0.5 text-xs font-medium rounded-full']">
                    {{ getInvitationStatusLabel(item.status) }}
                  </span>
                </div>
              </div>
              <div class="flex items-center justify-between">
                <div class="text-xs text-gray-400">
                  Истекает: {{ formatDateTime(item.expires_at) }}
                </div>
                <button
                  v-if="item.status === 'pending'"
                  @click="openCancelModal(item)"
                  class="text-xs text-red-600 hover:text-red-800"
                >
                  Отменить
                </button>
              </div>
            </div>
          </div>

          <!-- Desktop Table (visible on sm and above) -->
          <div class="hidden sm:block bg-white rounded-lg shadow overflow-hidden">
            <div class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-200">
                <thead class="bg-gray-50">
                  <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Email
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Роль
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      ФИО
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Телефон
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Статус
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Истекает
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Действия
                    </th>
                  </tr>
                </thead>
                <tbody class="bg-white divide-y divide-gray-200">
                  <tr v-for="item in invitations" :key="item.id" class="hover:bg-gray-50">
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {{ item.email }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {{ getInvitationRoleLabel(item.role) }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {{ item.name || '—' }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {{ item.phone || '—' }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                      <span
                        :class="[getInvitationStatusColor(item.status), 'px-2 py-1 text-xs font-medium rounded-full']"
                      >
                        {{ getInvitationStatusLabel(item.status) }}
                      </span>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {{ formatDateTime(item.expires_at) }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm">
                      <button
                        v-if="item.status === 'pending'"
                        @click="openCancelModal(item)"
                        class="text-red-600 hover:text-red-800"
                      >
                        Отменить
                      </button>
                      <span v-else class="text-gray-400">—</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>
      </template>

      <!-- History Tab Content -->
      <template v-if="currentTab === 'history'">
        <div class="bg-white rounded-lg shadow p-6">
          <EventHistory :load-fn="loadOrganizationHistory" />
        </div>
      </template>

      <!-- Filter Modal -->
      <div v-if="showFilterModal" class="fixed inset-0 bg-black/25 flex items-center justify-center p-4 z-50" @click="closeFilterModal">
        <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
          <h2 class="text-xl font-bold mb-4">Фильтры</h2>

          <div class="space-y-4">
            <!-- Members filters -->
            <template v-if="currentTab === 'members'">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Поиск
                </label>
                <input
                  v-model="tempSearch"
                  type="text"
                  class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="ФИО, email или телефон"
                />
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Роль
                </label>
                <select
                  v-model="tempRole"
                  class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option v-for="opt in roleOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Статус
                </label>
                <select
                  v-model="tempStatus"
                  class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
              </div>
            </template>

            <!-- Invitations filters -->
            <template v-else>
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Статус
                </label>
                <select
                  v-model="tempInvitationsStatus"
                  class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option v-for="opt in invitationStatusOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
              </div>
            </template>
          </div>

          <div class="flex flex-col gap-2 mt-6">
            <button
              @click="applyFilters"
              class="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Применить
            </button>
            <div class="flex gap-2">
              <button
                @click="clearFilters"
                class="flex-1 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
              >
                Очистить
              </button>
              <button
                @click="closeFilterModal"
                class="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
              >
                Отмена
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Create Invitation Modal -->
      <div v-if="showInvitationForm" class="fixed inset-0 bg-black/25 flex items-center justify-center p-4 z-50" @click="closeInvitationForm">
        <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
          <h2 class="text-xl font-bold mb-4">Новое приглашение</h2>

          <!-- Success state with token -->
          <div v-if="createdToken" class="space-y-4">
            <div class="bg-green-50 border border-green-200 rounded-lg p-4">
              <p class="text-green-700 font-medium mb-2">Приглашение создано!</p>
              <p class="text-sm text-green-600 mb-2">Ссылка для приглашения:</p>
              <div class="flex items-center gap-2">
                <input
                  type="text"
                  :value="getInvitationUrl(createdToken)"
                  readonly
                  class="flex-1 px-3 py-2 text-sm border rounded bg-gray-50"
                />
                <button
                  @click="copyToClipboard(getInvitationUrl(createdToken))"
                  class="px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm"
                >
                  Копировать
                </button>
              </div>
            </div>
            <button
              @click="closeInvitationForm"
              class="w-full px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
            >
              Закрыть
            </button>
          </div>

          <!-- Form -->
          <form v-else @submit.prevent="createInvitation" class="space-y-4">
            <div v-if="formError" class="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm">
              {{ formError }}
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                Email <span class="text-red-500">*</span>
              </label>
              <input
                v-model="invitationForm.email"
                type="email"
                required
                class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="user@example.com"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                Роль <span class="text-red-500">*</span>
              </label>
              <select
                v-model="invitationForm.role"
                class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option v-for="opt in invitationRoleOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                ФИО
                <span class="text-gray-400 font-normal">(опционально)</span>
              </label>
              <input
                v-model="invitationForm.name"
                type="text"
                class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Иванов Иван Иванович"
              />
              <p class="mt-1 text-xs text-gray-500">
                Если заполнить, приглашённый не сможет изменить
              </p>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                Телефон
                <span class="text-gray-400 font-normal">(опционально)</span>
              </label>
              <input
                v-model="invitationForm.phone"
                v-maska
                :data-maska="phoneMask"
                type="tel"
                inputmode="tel"
                class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                :placeholder="phonePlaceholder"
              />
              <p class="mt-1 text-xs text-gray-500">
                Если заполнить, приглашённый не сможет изменить
              </p>
            </div>

            <div class="flex gap-3 pt-2">
              <button
                type="button"
                @click="closeInvitationForm"
                class="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
              >
                Отмена
              </button>
              <button
                type="submit"
                :disabled="isSubmitting"
                class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                {{ isSubmitting ? 'Создание...' : 'Создать' }}
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Cancel Invitation Modal -->
      <div
        v-if="showCancelModal"
        class="fixed inset-0 bg-black/25 flex items-center justify-center z-50 p-4"
        @click.self="closeCancelModal"
      >
        <div class="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
          <h3 class="text-lg font-medium text-gray-900 mb-2">
            Отменить приглашение?
          </h3>
          <p class="text-sm text-gray-500 mb-4">
            Вы уверены, что хотите отменить приглашение для <strong>{{ cancellingInvitation?.email }}</strong>?
            Пользователь не сможет принять это приглашение.
          </p>

          <div v-if="cancelError" class="bg-red-50 border border-red-200 rounded p-3 mb-4 text-sm text-red-700">
            {{ cancelError }}
          </div>

          <div class="flex justify-end gap-3">
            <button
              @click="closeCancelModal"
              :disabled="cancellingId !== null"
              class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md disabled:opacity-50"
            >
              Отмена
            </button>
            <button
              @click="confirmCancel"
              :disabled="cancellingId !== null"
              class="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md disabled:opacity-50"
            >
              <span v-if="cancellingId" class="flex items-center gap-2">
                <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                Отмена...
              </span>
              <span v-else>Отменить приглашение</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
