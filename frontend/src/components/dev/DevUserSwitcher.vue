<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { devApi, type DevUser } from '@/api/dev'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const isOpen = ref(false)
const isLoading = ref(false)
const isSwitching = ref(false)
const isDeleting = ref<string | null>(null)
const search = ref('')
const users = ref<DevUser[]>([])
const error = ref<string | null>(null)

let searchTimeout: ReturnType<typeof setTimeout> | null = null

watch(search, (value) => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    loadUsers(value)
  }, 300)
})

async function loadUsers(searchQuery = '') {
  isLoading.value = true
  error.value = null
  try {
    users.value = await devApi.listUsers(searchQuery)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

async function switchUser(user: DevUser) {
  isSwitching.value = true
  try {
    await devApi.switchUser(user.id)
    await auth.fetchMe()
    isOpen.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка переключения'
  } finally {
    isSwitching.value = false
  }
}

async function deleteUser(user: DevUser, event: Event) {
  event.stopPropagation()
  if (!confirm(`Удалить пользователя ${user.name} (${user.email})?`)) {
    return
  }
  isDeleting.value = user.id
  error.value = null
  try {
    await devApi.deleteUser(user.id)
    users.value = users.value.filter(u => u.id !== user.id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка удаления'
  } finally {
    isDeleting.value = null
  }
}

function toggle() {
  isOpen.value = !isOpen.value
  if (isOpen.value && users.value.length === 0) {
    loadUsers()
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.ctrlKey && e.shiftKey && e.key === 'U') {
    e.preventDefault()
    toggle()
  }
  if (e.key === 'Escape' && isOpen.value) {
    isOpen.value = false
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

const currentUserId = computed(() => auth.memberId)

const groupedUsers = computed(() => {
  const groups: Record<string, { orgName: string; orgStatus: string; users: DevUser[] }> = {}
  for (const user of users.value) {
    if (!groups[user.organization_id]) {
      groups[user.organization_id] = {
        orgName: user.organization_name,
        orgStatus: user.organization_status,
        users: [],
      }
    }
    groups[user.organization_id].users.push(user)
  }
  return Object.values(groups)
})

function getStatusBadgeClass(status: string) {
  switch (status) {
    case 'active':
      return 'bg-green-100 text-green-700'
    case 'pending':
      return 'bg-yellow-100 text-yellow-700'
    case 'rejected':
      return 'bg-red-100 text-red-700'
    case 'suspended':
      return 'bg-gray-100 text-gray-700'
    default:
      return 'bg-gray-100 text-gray-600'
  }
}

function getRoleBadgeClass(role: string) {
  switch (role) {
    case 'owner':
      return 'bg-purple-100 text-purple-700'
    case 'administrator':
      return 'bg-blue-100 text-blue-700'
    case 'employee':
      return 'bg-gray-100 text-gray-600'
    default:
      return 'bg-gray-100 text-gray-600'
  }
}
</script>

<template>
  <!-- Trigger button -->
  <div class="fixed right-0 top-1/2 -translate-y-1/2 z-[9999]">
    <button
      @click="toggle"
      class="bg-orange-500 hover:bg-orange-600 text-white px-2 py-4 rounded-l-lg shadow-lg transition-all"
      :class="{ 'translate-x-0': !isOpen, '-translate-x-2': isOpen }"
      title="Dev User Switcher (Ctrl+Shift+U)"
    >
      <span class="writing-mode-vertical text-xs font-medium">DEV</span>
    </button>
  </div>

  <!-- Slide-out panel -->
  <Transition name="slide">
    <div
      v-if="isOpen"
      class="fixed right-0 top-0 h-full w-80 bg-white shadow-2xl z-[9998] flex flex-col"
    >
      <!-- Header -->
      <div class="bg-orange-500 text-white p-4 flex items-center justify-between">
        <h2 class="font-semibold">Dev User Switcher</h2>
        <button @click="isOpen = false" class="hover:bg-orange-600 rounded p-1">
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Search -->
      <div class="p-3 border-b">
        <input
          v-model="search"
          type="text"
          placeholder="Поиск по email или имени..."
          class="w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
        />
      </div>

      <!-- Content -->
      <div class="flex-1 overflow-y-auto">
        <!-- Loading -->
        <div v-if="isLoading" class="p-4 text-center text-gray-500">
          Загрузка...
        </div>

        <!-- Error -->
        <div v-else-if="error" class="p-4 text-center text-red-500">
          {{ error }}
        </div>

        <!-- Users list -->
        <div v-else class="p-2">
          <div v-for="group in groupedUsers" :key="group.orgName" class="mb-4">
            <!-- Organization header -->
            <div class="px-2 py-1 mb-1 flex items-center gap-2">
              <span class="font-medium text-sm text-gray-700 truncate">{{ group.orgName }}</span>
              <span
                :class="getStatusBadgeClass(group.orgStatus)"
                class="text-xs px-1.5 py-0.5 rounded shrink-0"
              >
                {{ group.orgStatus }}
              </span>
            </div>

            <!-- Users in organization -->
            <div class="space-y-1">
              <div
                v-for="user in group.users"
                :key="user.id"
                :class="[
                  'rounded-md text-sm transition-colors',
                  user.id === currentUserId
                    ? 'bg-orange-100 border border-orange-300'
                    : 'hover:bg-gray-100 border border-transparent',
                ]"
              >
                <button
                  @click="switchUser(user)"
                  :disabled="isSwitching || user.id === currentUserId"
                  :class="[
                    'w-full text-left px-3 py-2',
                    isSwitching ? 'opacity-50 cursor-not-allowed' : ''
                  ]"
                >
                  <div class="flex items-center justify-between">
                    <div class="truncate">
                      <div class="font-medium text-gray-900 truncate">{{ user.name }}</div>
                      <div class="text-xs text-gray-500 truncate">{{ user.email }}</div>
                    </div>
                    <span
                      :class="getRoleBadgeClass(user.role)"
                      class="text-xs px-1.5 py-0.5 rounded ml-2 shrink-0"
                    >
                      {{ user.role }}
                    </span>
                  </div>
                  <div v-if="user.id === currentUserId" class="text-xs text-orange-600 mt-1">
                    Текущий пользователь
                  </div>
                </button>
                <!-- Delete button (not for owner) -->
                <button
                  v-if="user.role !== 'owner'"
                  @click="deleteUser(user, $event)"
                  :disabled="isDeleting === user.id"
                  class="w-full px-3 py-1.5 text-xs text-red-500 hover:bg-red-50 border-t border-gray-100 transition-colors flex items-center justify-center gap-1"
                  :class="{ 'opacity-50 cursor-not-allowed': isDeleting === user.id }"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Удалить
                </button>
              </div>
            </div>
          </div>

          <!-- Empty state -->
          <div v-if="!isLoading && users.length === 0" class="p-4 text-center text-gray-500">
            Пользователи не найдены
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="p-2 border-t text-center text-xs text-gray-400">
        Ctrl+Shift+U для открытия/закрытия
      </div>
    </div>
  </Transition>

  <!-- Backdrop -->
  <Transition name="fade">
    <div
      v-if="isOpen"
      @click="isOpen = false"
      class="fixed inset-0 bg-black/20 z-[9997]"
    />
  </Transition>
</template>

<style scoped>
.writing-mode-vertical {
  writing-mode: vertical-rl;
  text-orientation: mixed;
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
