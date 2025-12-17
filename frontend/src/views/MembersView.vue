<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { membersApi } from '@/api/members'
import { freightRequestsApi } from '@/api/freightRequests'
import type { MemberListItem, MemberRole, MemberStatus } from '@/types/member'
import {
  roleLabels,
  roleColors,
  statusLabels,
  statusColors,
  roleOptions,
  statusOptions,
} from '@/types/member'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

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

const hasActiveFilters = computed(
  () => searchQuery.value.trim() !== '' || roleFilter.value !== '' || statusFilter.value !== ''
)

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

// Filter modal
function openFilterModal() {
  tempSearch.value = searchQuery.value
  tempRole.value = roleFilter.value
  tempStatus.value = statusFilter.value
  showFilterModal.value = true
}

function applyFilters() {
  searchQuery.value = tempSearch.value
  roleFilter.value = tempRole.value
  statusFilter.value = tempStatus.value
  showFilterModal.value = false
}

function clearFilters() {
  tempSearch.value = ''
  tempRole.value = ''
  tempStatus.value = ''
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
})
</script>

<template>
  <div class="min-h-screen bg-gray-100 p-4 sm:p-6">
    <div class="max-w-6xl mx-auto">
      <!-- Header -->
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-2xl font-bold">
            {{ isSelectionMode ? 'Выберите ответственного' : 'Сотрудники' }}
          </h1>
          <p v-if="isSelectionMode" class="text-sm text-gray-500 mt-1">
            Нажмите на сотрудника для назначения
          </p>
        </div>
        <div class="flex gap-2">
          <button
            v-if="isSelectionMode"
            @click="cancelSelection"
            class="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors"
          >
            Отмена
          </button>
          <button
            @click="openFilterModal"
            class="px-4 py-2 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors flex items-center gap-2"
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

      <!-- Active filters indicator -->
      <div v-if="hasActiveFilters" class="bg-blue-50 border border-blue-200 rounded-lg p-3 mb-6 flex items-center justify-between">
        <div class="text-sm text-blue-700">
          <span v-if="searchQuery">Поиск: "{{ searchQuery }}"</span>
          <span v-if="searchQuery && (roleFilter || statusFilter)">, </span>
          <span v-if="roleFilter">Роль: {{ roleLabels[roleFilter] }}</span>
          <span v-if="roleFilter && statusFilter">, </span>
          <span v-if="statusFilter">Статус: {{ statusLabels[statusFilter] }}</span>
        </div>
        <button
          @click="searchQuery = ''; roleFilter = ''; statusFilter = ''"
          class="text-blue-600 hover:text-blue-800 text-sm underline"
        >
          Сбросить
        </button>
      </div>

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

      <!-- Filter Modal -->
      <div v-if="showFilterModal" class="fixed inset-0 bg-black/25 flex items-center justify-center p-4 z-50" @click="closeFilterModal">
        <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
          <h2 class="text-xl font-bold mb-4">Фильтры</h2>

          <div class="space-y-4">
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
    </div>
  </div>
</template>
