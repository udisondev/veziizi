<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { membersApi, type MemberProfile } from '@/api/members'
import {
  roleLabels,
  roleColors,
  statusLabels,
  statusColors,
} from '@/types/member'

// UI Components
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

// Shared Components
import { DetailPageHeader } from '@/components/shared'

// Icons
import { MoreVertical } from 'lucide-vue-next'

const route = useRoute()
const auth = useAuthStore()

const member = ref<MemberProfile | null>(null)
const isLoading = ref(true)
const error = ref('')

// Block modal
const showBlockModal = ref(false)
const blockReason = ref('')
const blockLoading = ref(false)
const blockError = ref('')

// Unblock modal
const showUnblockModal = ref(false)
const unblockLoading = ref(false)
const unblockError = ref('')

// Edit modal
const showEditModal = ref(false)
const editName = ref('')
const editEmail = ref('')
const editPhone = ref('')
const editLoading = ref(false)
const editError = ref('')

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

const canEdit = computed(() => {
  if (!member.value || !auth.organizationId) return false
  // User must be from the same organization
  if (member.value.organization_id !== auth.organizationId) return false

  // Owner can only be edited by themselves
  if (member.value.role === 'owner') {
    return member.value.id === auth.memberId
  }

  // For non-owner members: manager can edit anyone, or member can edit themselves
  if (auth.role === 'owner' || auth.role === 'administrator') {
    return true
  }

  // Regular employee can only edit themselves
  return member.value.id === auth.memberId
})

const hasAnyAction = computed(() => canBlock.value || canUnblock.value || canEdit.value)

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

onMounted(() => {
  loadData()
})


watch(() => route.params.id, () => {
  loadData()
})

// Block actions
function openBlockModal() {
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
  } catch (e: unknown) {
    blockError.value = e instanceof Error ? e.message : 'Не удалось заблокировать сотрудника'
  } finally {
    blockLoading.value = false
  }
}

// Unblock actions
function openUnblockModal() {
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
  } catch (e: unknown) {
    unblockError.value = e instanceof Error ? e.message : 'Не удалось разблокировать сотрудника'
  } finally {
    unblockLoading.value = false
  }
}

// Edit actions
function openEditModal() {
  if (!member.value) return
  editName.value = member.value.name
  editEmail.value = member.value.email
  editPhone.value = member.value.phone || ''
  editError.value = ''
  showEditModal.value = true
}

function closeEditModal() {
  showEditModal.value = false
}

async function confirmEdit() {
  if (!member.value) return

  if (!editName.value.trim()) {
    editError.value = 'Укажите ФИО'
    return
  }
  if (!editEmail.value.trim()) {
    editError.value = 'Укажите Email'
    return
  }
  if (!editPhone.value.trim()) {
    editError.value = 'Укажите телефон'
    return
  }

  editLoading.value = true
  editError.value = ''

  try {
    await membersApi.updateInfo(
      member.value.organization_id,
      member.value.id,
      editName.value.trim(),
      editEmail.value.trim(),
      editPhone.value.trim()
    )
    // Optimistic update - сразу обновляем локальное состояние
    member.value.name = editName.value.trim()
    member.value.email = editEmail.value.trim()
    member.value.phone = editPhone.value.trim()
    closeEditModal()
  } catch (e: unknown) {
    editError.value = e instanceof Error ? e.message : 'Не удалось обновить данные сотрудника'
  } finally {
    editLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <DetailPageHeader back-to="/" back-label="Назад" use-history>
      <template #actions>
        <DropdownMenu v-if="member && hasAnyAction">
          <DropdownMenuTrigger as-child>
            <Button data-tutorial="member-actions" variant="ghost" size="icon">
              <MoreVertical class="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              v-if="canEdit"
              data-tutorial="edit-member-btn"
              @click="openEditModal"
            >
              Редактировать
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="canBlock"
              data-tutorial="block-member-btn"
              class="text-destructive focus:text-destructive"
              @click="openBlockModal"
            >
              Заблокировать
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="canUnblock"
              data-tutorial="unblock-member-btn"
              class="text-success focus:text-success"
              @click="openUnblockModal"
            >
              Разблокировать
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </template>
    </DetailPageHeader>

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

    <!-- Edit Modal -->
    <div v-if="showEditModal" class="fixed inset-0 bg-black/25 flex items-center justify-center z-50 p-4" @click="closeEditModal">
      <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" @click.stop>
        <h2 class="text-xl font-bold mb-4 text-gray-900">Редактирование данных сотрудника</h2>

        <div v-if="editError" class="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm mb-4">
          {{ editError }}
        </div>

        <div class="space-y-4 mb-6">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              ФИО <span class="text-red-500">*</span>
            </label>
            <input
              v-model="editName"
              type="text"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Иванов Иван Иванович"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Email <span class="text-red-500">*</span>
            </label>
            <input
              v-model="editEmail"
              type="email"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="example@mail.ru"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Телефон <span class="text-red-500">*</span>
            </label>
            <input
              v-model="editPhone"
              type="tel"
              class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="+7 (999) 123-45-67"
            />
          </div>
        </div>

        <div class="flex gap-3">
          <button
            @click="closeEditModal"
            :disabled="editLoading"
            class="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 disabled:opacity-50"
          >
            Отмена
          </button>
          <button
            @click="confirmEdit"
            :disabled="editLoading || !editName.trim() || !editEmail.trim() || !editPhone.trim()"
            class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {{ editLoading ? 'Сохранение...' : 'Сохранить' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
