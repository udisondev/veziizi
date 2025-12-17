<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { vMaska } from 'maska/vue'
import { invitationsApi } from '@/api/invitations'
import type { InvitationListItem, InvitationStatus, InvitationRole } from '@/types/invitation'

// Маска телефона
const phoneMask = '+7 (###) ###-##-##'
const phonePlaceholder = '+7 (999) 999-99-99'

const auth = useAuthStore()

const items = ref<InvitationListItem[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

// Filters (applied state)
const statusFilter = ref<InvitationStatus | ''>('')

// Temp filters for modal
const tempStatus = ref<InvitationStatus | ''>('')
const showFilterModal = ref(false)

const statusOptions: { value: InvitationStatus | '', label: string }[] = [
  { value: '', label: 'Все статусы' },
  { value: 'pending', label: 'Ожидают' },
  { value: 'accepted', label: 'Приняты' },
  { value: 'expired', label: 'Истекли' },
  { value: 'cancelled', label: 'Отменены' },
]

// Computed
const hasActiveFilters = computed(() => statusFilter.value !== '')

// Form state
const showForm = ref(false)
const isSubmitting = ref(false)
const formError = ref<string | null>(null)
const createdToken = ref<string | null>(null)

const form = ref({
  email: '',
  role: 'employee' as InvitationRole,
  name: '',
  phone: '',
})

const roleOptions: { value: InvitationRole, label: string }[] = [
  { value: 'employee', label: 'Сотрудник' },
  { value: 'administrator', label: 'Администратор' },
]

async function loadItems() {
  if (!auth.organizationId) return

  isLoading.value = true
  error.value = null

  try {
    const status = statusFilter.value || undefined
    const response = await invitationsApi.list(auth.organizationId, status)
    items.value = response.items ?? []
  } catch (e) {
    error.value = 'Не удалось загрузить приглашения'
    console.error(e)
  } finally {
    isLoading.value = false
  }
}

async function createInvitation() {
  if (!auth.organizationId) return

  isSubmitting.value = true
  formError.value = null
  createdToken.value = null

  try {
    const response = await invitationsApi.create(auth.organizationId, {
      email: form.value.email,
      role: form.value.role,
      name: form.value.name || undefined,
      phone: form.value.phone || undefined,
    })

    createdToken.value = response.token

    // Reset form
    form.value = { email: '', role: 'employee', name: '', phone: '' }

    // Reload list
    await loadItems()
  } catch (e: any) {
    formError.value = e?.message || 'Не удалось создать приглашение'
    console.error(e)
  } finally {
    isSubmitting.value = false
  }
}

function closeForm() {
  showForm.value = false
  createdToken.value = null
  formError.value = null
}

// Cancel invitation
const cancellingId = ref<string | null>(null)
const showCancelModal = ref(false)
const cancellingInvitation = ref<InvitationListItem | null>(null)
const cancelError = ref<string | null>(null)

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
    // Optimistic update — сразу обновляем локальное состояние
    const item = items.value.find(i => i.id === cancellingInvitation.value!.id)
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

// Filter modal functions
function openFilterModal() {
  tempStatus.value = statusFilter.value
  showFilterModal.value = true
}

function applyFilters() {
  statusFilter.value = tempStatus.value
  showFilterModal.value = false
  loadItems()
}

function clearFilters() {
  tempStatus.value = ''
}

function resetAllFilters() {
  statusFilter.value = ''
  loadItems()
}

function closeFilterModal() {
  showFilterModal.value = false
}

function getStatusLabel(status: InvitationStatus): string {
  switch (status) {
    case 'pending': return 'Ожидает'
    case 'accepted': return 'Принято'
    case 'expired': return 'Истекло'
    case 'cancelled': return 'Отменено'
    default: return status
  }
}

function getStatusColor(status: InvitationStatus): string {
  switch (status) {
    case 'pending': return 'bg-yellow-100 text-yellow-800'
    case 'accepted': return 'bg-green-100 text-green-800'
    case 'expired': return 'bg-gray-100 text-gray-800'
    case 'cancelled': return 'bg-red-100 text-red-800'
    default: return 'bg-gray-100 text-gray-800'
  }
}

function getRoleLabel(role: string): string {
  switch (role) {
    case 'employee': return 'Сотрудник'
    case 'administrator': return 'Администратор'
    default: return role
  }
}

function formatDate(dateStr: string): string {
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

onMounted(() => {
  loadItems()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100 p-6">
    <div class="max-w-6xl mx-auto">
      <!-- Header -->
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6">
        <h1 class="text-xl sm:text-2xl font-bold text-gray-900 mb-4 sm:mb-0">Приглашения</h1>

        <div class="flex gap-2">
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

          <button
            @click="showForm = true"
            class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            + Создать
          </button>
        </div>
      </div>

      <!-- Active filters indicator -->
      <div v-if="hasActiveFilters" class="bg-blue-50 border border-blue-200 rounded-lg p-3 mb-6 flex items-center justify-between">
        <div class="text-sm text-blue-700">
          Статус: {{ statusOptions.find(o => o.value === statusFilter)?.label }}
        </div>
        <button
          @click="resetAllFilters"
          class="text-blue-600 hover:text-blue-800 text-sm underline whitespace-nowrap ml-2"
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
      </div>

      <!-- Empty state -->
      <div v-else-if="items.length === 0" class="bg-white rounded-lg shadow p-12 text-center">
        <p class="text-gray-500">
          {{ hasActiveFilters ? 'Нет приглашений по заданным фильтрам' : 'Приглашений пока нет' }}
        </p>
      </div>

      <!-- Invitations List -->
      <template v-else>
        <!-- Mobile Cards (visible below sm) -->
        <div class="sm:hidden space-y-3">
          <div
            v-for="item in items"
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
                  {{ getRoleLabel(item.role) }}
                </span>
                <span :class="[getStatusColor(item.status), 'px-2 py-0.5 text-xs font-medium rounded-full']">
                  {{ getStatusLabel(item.status) }}
                </span>
              </div>
            </div>
            <div class="flex items-center justify-between">
              <div class="text-xs text-gray-400">
                Истекает: {{ formatDate(item.expires_at) }}
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
                <tr v-for="item in items" :key="item.id" class="hover:bg-gray-50">
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {{ item.email }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ getRoleLabel(item.role) }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ item.name || '—' }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ item.phone || '—' }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <span
                      :class="[getStatusColor(item.status), 'px-2 py-1 text-xs font-medium rounded-full']"
                    >
                      {{ getStatusLabel(item.status) }}
                    </span>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ formatDate(item.expires_at) }}
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

      <!-- Create invitation modal -->
      <div v-if="showForm" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md">
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
              @click="closeForm"
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
                v-model="form.email"
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
                v-model="form.role"
                class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option v-for="opt in roleOptions" :key="opt.value" :value="opt.value">
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
                v-model="form.name"
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
                v-model="form.phone"
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
                @click="closeForm"
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

      <!-- Filter Modal -->
      <div v-if="showFilterModal" class="fixed inset-0 bg-black/25 flex items-center justify-center p-4 z-50" @click="closeFilterModal">
        <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
          <h2 class="text-xl font-bold mb-4">Фильтры</h2>

          <div class="space-y-4">
            <!-- Status -->
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

      <!-- Cancel Confirmation Modal -->
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
