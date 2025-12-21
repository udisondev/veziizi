<script setup lang="ts">
import { ref, computed, onMounted, watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { membersApi, type MemberProfile } from '@/api/members'
import {
  roleLabels,
  roleColors,
  statusLabels,
  statusColors,
} from '@/types/member'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const member = ref<MemberProfile | null>(null)
const isLoading = ref(true)
const error = ref('')

// Menu state
const showMenu = ref(false)

// Block modal
const showBlockModal = ref(false)
const blockReason = ref('')
const blockLoading = ref(false)
const blockError = ref('')

// Unblock modal
const showUnblockModal = ref(false)
const unblockLoading = ref(false)
const unblockError = ref('')

// Permissions
const canManage = computed(() => {
  if (!member.value || !auth.organizationId) return false
  // User must be from the same organization
  if (member.value.organization_id !== auth.organizationId) return false
  // User must be owner or administrator
  return auth.role === 'owner' || auth.role === 'administrator'
})

const canBlock = computed(() => {
  if (!canManage.value || !member.value) return false
  // Cannot block owner
  if (member.value.role === 'owner') return false
  // Cannot block self
  if (member.value.id === auth.memberId) return false
  // Must be active
  return member.value.status === 'active'
})

const canUnblock = computed(() => {
  if (!canManage.value || !member.value) return false
  // Must be blocked
  return member.value.status === 'blocked'
})

const hasAnyAction = computed(() => canBlock.value || canUnblock.value)

async function loadData() {
  isLoading.value = true
  error.value = ''
  try {
    const id = route.params.id as string
    member.value = await membersApi.getProfile(id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

// Menu toggle
function toggleMenu() {
  showMenu.value = !showMenu.value
}

function closeMenu() {
  showMenu.value = false
}

// Close menu on outside click
function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('.menu-container')) {
    closeMenu()
  }
}

onMounted(() => {
  loadData()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

watch(() => route.params.id, () => {
  loadData()
})

// Block actions
function openBlockModal() {
  closeMenu()
  blockReason.value = ''
  blockError.value = ''
  showBlockModal.value = true
}

function closeBlockModal() {
  showBlockModal.value = false
}

async function confirmBlock() {
  if (!member.value || !blockReason.value.trim()) {
    blockError.value = 'Укажите причину блокировки'
    return
  }

  blockLoading.value = true
  blockError.value = ''

  try {
    await membersApi.block(member.value.organization_id, member.value.id, blockReason.value.trim())
    // Optimistic update - сразу обновляем локальное состояние
    member.value.status = 'blocked'
    closeBlockModal()
  } catch (e: any) {
    blockError.value = e?.message || 'Не удалось заблокировать сотрудника'
  } finally {
    blockLoading.value = false
  }
}

// Unblock actions
function openUnblockModal() {
  closeMenu()
  unblockError.value = ''
  showUnblockModal.value = true
}

function closeUnblockModal() {
  showUnblockModal.value = false
}

async function confirmUnblock() {
  if (!member.value) return

  unblockLoading.value = true
  unblockError.value = ''

  try {
    await membersApi.unblock(member.value.organization_id, member.value.id)
    // Optimistic update - сразу обновляем локальное состояние
    member.value.status = 'active'
    closeUnblockModal()
  } catch (e: any) {
    unblockError.value = e?.message || 'Не удалось разблокировать сотрудника'
  } finally {
    unblockLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-white shadow">
      <div class="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
        <button
          @click="router.back()"
          class="text-blue-600 hover:text-blue-800 text-sm"
        >
          &larr; Назад
        </button>

        <!-- Actions menu (three dots) -->
        <div v-if="member && hasAnyAction" class="relative menu-container">
          <button
            @click.stop="toggleMenu"
            class="p-2 hover:bg-gray-100 rounded-lg transition-colors"
            title="Действия"
          >
            <svg class="w-5 h-5 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
              <circle cx="12" cy="5" r="2" />
              <circle cx="12" cy="12" r="2" />
              <circle cx="12" cy="19" r="2" />
            </svg>
          </button>

          <!-- Dropdown menu -->
          <div
            v-if="showMenu"
            class="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-10"
          >
            <button
              v-if="canBlock"
              @click="openBlockModal"
              class="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50 transition-colors"
            >
              Заблокировать
            </button>
            <button
              v-if="canUnblock"
              @click="openUnblockModal"
              class="w-full px-4 py-2 text-left text-sm text-green-600 hover:bg-green-50 transition-colors"
            >
              Разблокировать
            </button>
          </div>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-4xl mx-auto px-4 py-6">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
        <div class="text-gray-500 mt-2">Загрузка...</div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
        {{ error }}
        <button @click="loadData" class="ml-4 text-red-600 underline">Повторить</button>
      </div>

      <!-- Content -->
      <div v-else-if="member" class="space-y-6">
        <!-- Header Card -->
        <div class="bg-white rounded-lg shadow p-6">
          <div class="flex items-start justify-between">
            <div>
              <h1 class="text-2xl font-bold text-gray-900 break-words">{{ member.name }}</h1>
              <div class="flex items-center gap-2 mt-2">
                <span :class="[roleColors[member.role], 'px-2 py-0.5 text-xs font-medium rounded-full']">
                  {{ roleLabels[member.role] }}
                </span>
                <span :class="[statusColors[member.status], 'px-2 py-0.5 text-xs font-medium rounded-full']">
                  {{ statusLabels[member.status] }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Details Card -->
        <div class="bg-white rounded-lg shadow p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Информация</h2>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <dt class="text-sm text-gray-500">ФИО</dt>
              <dd class="text-gray-900 font-medium break-words">{{ member.name }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Email</dt>
              <dd class="text-gray-900">
                <a :href="`mailto:${member.email}`" class="text-blue-600 hover:text-blue-800">
                  {{ member.email }}
                </a>
              </dd>
            </div>
            <div v-if="member.phone">
              <dt class="text-sm text-gray-500">Телефон</dt>
              <dd class="text-gray-900">
                <a :href="`tel:${member.phone}`" class="text-blue-600 hover:text-blue-800">
                  {{ member.phone }}
                </a>
              </dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Организация</dt>
              <dd>
                <router-link
                  :to="{ name: 'organization-profile', params: { id: member.organization_id } }"
                  class="text-blue-600 hover:text-blue-800 hover:underline break-words"
                >
                  {{ member.organization_name }}
                </router-link>
              </dd>
            </div>
            <div>
              <dt class="text-sm text-gray-500">Дата регистрации</dt>
              <dd class="text-gray-900">{{ formatDate(member.created_at) }}</dd>
            </div>
          </dl>
        </div>
      </div>
    </main>

    <!-- Block Modal -->
    <div v-if="showBlockModal" class="fixed inset-0 bg-black/25 flex items-center justify-center z-50 p-4" @click="closeBlockModal">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
        <h2 class="text-xl font-bold mb-4 text-red-600">Заблокировать сотрудника?</h2>

        <div v-if="blockError" class="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm mb-4">
          {{ blockError }}
        </div>

        <div class="bg-yellow-50 border border-yellow-200 rounded-lg p-3 mb-4">
          <p class="text-sm text-yellow-800">
            Заблокированный сотрудник не сможет войти в систему и выполнять действия от имени организации.
          </p>
        </div>

        <div class="mb-4">
          <p class="text-sm text-gray-600">
            Сотрудник: <strong>{{ member?.name }}</strong>
          </p>
          <p class="text-sm text-gray-600">
            Email: {{ member?.email }}
          </p>
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-1">
            Причина блокировки <span class="text-red-500">*</span>
          </label>
          <textarea
            v-model="blockReason"
            rows="3"
            class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-red-500"
            placeholder="Укажите причину блокировки..."
          ></textarea>
        </div>

        <div class="flex gap-3">
          <button
            @click="closeBlockModal"
            :disabled="blockLoading"
            class="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 disabled:opacity-50"
          >
            Отмена
          </button>
          <button
            @click="confirmBlock"
            :disabled="blockLoading || !blockReason.trim()"
            class="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
          >
            {{ blockLoading ? 'Блокировка...' : 'Заблокировать' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Unblock Modal -->
    <div v-if="showUnblockModal" class="fixed inset-0 bg-black/25 flex items-center justify-center z-50 p-4" @click="closeUnblockModal">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
        <h2 class="text-xl font-bold mb-4 text-green-600">Разблокировать сотрудника?</h2>

        <div v-if="unblockError" class="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm mb-4">
          {{ unblockError }}
        </div>

        <div class="mb-4">
          <p class="text-sm text-gray-600">
            Сотрудник: <strong>{{ member?.name }}</strong>
          </p>
          <p class="text-sm text-gray-600">
            Email: {{ member?.email }}
          </p>
        </div>

        <p class="text-sm text-gray-600 mb-4">
          После разблокировки сотрудник сможет снова войти в систему и выполнять действия.
        </p>

        <div class="flex gap-3">
          <button
            @click="closeUnblockModal"
            :disabled="unblockLoading"
            class="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 disabled:opacity-50"
          >
            Отмена
          </button>
          <button
            @click="confirmUnblock"
            :disabled="unblockLoading"
            class="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
          >
            {{ unblockLoading ? 'Разблокировка...' : 'Разблокировать' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
