<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { adminApi } from '@/api/admin'
import type { OrganizationDetail } from '@/types/admin'

const route = useRoute()
const router = useRouter()

const organization = ref<OrganizationDetail | null>(null)
const isLoading = ref(true)
const error = ref('')
const actionLoading = ref(false)
const showRejectModal = ref(false)
const rejectReason = ref('')

const countryNames: Record<string, string> = {
  RU: 'Россия',
  KZ: 'Казахстан',
  BY: 'Беларусь',
}

const statusNames: Record<string, string> = {
  pending: 'На модерации',
  active: 'Активна',
  suspended: 'Приостановлена',
  rejected: 'Отклонена',
}

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-500',
  active: 'bg-green-500',
  suspended: 'bg-orange-500',
  rejected: 'bg-red-500',
}

const roleNames: Record<string, string> = {
  owner: 'Владелец',
  administrator: 'Администратор',
  employee: 'Сотрудник',
}

onMounted(async () => {
  await loadOrganization()
})

async function loadOrganization() {
  isLoading.value = true
  error.value = ''
  try {
    const id = route.params.id as string
    organization.value = await adminApi.getOrganization(id)
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
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function handleApprove() {
  if (!organization.value) return
  actionLoading.value = true
  try {
    await adminApi.approveOrganization(organization.value.id)
    router.push('/admin')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
  }
}

async function handleReject() {
  if (!organization.value || !rejectReason.value.trim()) return
  actionLoading.value = true
  try {
    await adminApi.rejectOrganization(organization.value.id, { reason: rejectReason.value })
    router.push('/admin')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка'
  } finally {
    actionLoading.value = false
    showRejectModal.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-gray-900">
    <!-- Header -->
    <header class="bg-gray-800 shadow">
      <div class="max-w-7xl mx-auto px-4 py-4">
        <router-link to="/admin" class="text-indigo-400 hover:text-indigo-300 text-sm">
          &larr; Назад к списку
        </router-link>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-4xl mx-auto px-4 py-8">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="text-gray-400">Загрузка...</div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded">
        {{ error }}
      </div>

      <!-- Organization Details -->
      <div v-else-if="organization" class="space-y-6">
        <!-- Header -->
        <div class="flex justify-between items-start">
          <div>
            <h1 class="text-2xl font-bold text-white break-words">{{ organization.name }}</h1>
            <p class="text-gray-400 break-words">{{ organization.legal_name }}</p>
          </div>
          <span :class="[statusColors[organization.status], 'px-3 py-1 rounded-full text-sm text-white']">
            {{ statusNames[organization.status] }}
          </span>
        </div>

        <!-- Info Card -->
        <div class="bg-gray-800 rounded-lg p-6 space-y-4">
          <h2 class="text-lg font-medium text-white mb-4">Информация об организации</h2>
          <dl class="grid grid-cols-2 gap-4">
            <div>
              <dt class="text-sm text-gray-400">ИНН</dt>
              <dd class="text-white">{{ organization.inn }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-400">Страна</dt>
              <dd class="text-white">{{ countryNames[organization.country] }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-400">Телефон</dt>
              <dd class="text-white">{{ organization.phone }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-400">Email</dt>
              <dd class="text-white">{{ organization.email }}</dd>
            </div>
            <div class="col-span-2">
              <dt class="text-sm text-gray-400">Адрес</dt>
              <dd class="text-white break-words">{{ organization.address }}</dd>
            </div>
            <div>
              <dt class="text-sm text-gray-400">Дата регистрации</dt>
              <dd class="text-white">{{ formatDate(organization.created_at) }}</dd>
            </div>
          </dl>
        </div>

        <!-- Members -->
        <div class="bg-gray-800 rounded-lg p-6">
          <h2 class="text-lg font-medium text-white mb-4">Сотрудники</h2>
          <div class="overflow-hidden">
            <table class="min-w-full divide-y divide-gray-700">
              <thead>
                <tr>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase">Имя</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase">Email</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase">Телефон</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase">Роль</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-700">
                <tr v-for="member in organization.members" :key="member.id">
                  <td class="px-4 py-3 text-sm text-white">{{ member.name }}</td>
                  <td class="px-4 py-3 text-sm text-gray-300">{{ member.email }}</td>
                  <td class="px-4 py-3 text-sm text-gray-300">{{ member.phone }}</td>
                  <td class="px-4 py-3 text-sm text-gray-300">{{ roleNames[member.role] }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Actions -->
        <div v-if="organization.status === 'pending'" class="flex gap-4">
          <button
            @click="handleApprove"
            :disabled="actionLoading"
            class="flex-1 py-3 px-4 bg-green-600 hover:bg-green-700 text-white font-medium rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Обработка...' : 'Одобрить' }}
          </button>
          <button
            @click="showRejectModal = true"
            :disabled="actionLoading"
            class="flex-1 py-3 px-4 bg-red-600 hover:bg-red-700 text-white font-medium rounded-lg disabled:opacity-50"
          >
            Отклонить
          </button>
        </div>
      </div>
    </main>

    <!-- Reject Modal -->
    <div v-if="showRejectModal" class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div class="bg-gray-800 rounded-lg p-6 max-w-md w-full">
        <h3 class="text-lg font-medium text-white mb-4">Отклонить организацию</h3>
        <div class="mb-4">
          <label class="block text-sm text-gray-400 mb-2">Причина отклонения</label>
          <textarea
            v-model="rejectReason"
            rows="3"
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            placeholder="Укажите причину..."
          ></textarea>
        </div>
        <div class="flex gap-3">
          <button
            @click="showRejectModal = false"
            class="flex-1 py-2 px-4 bg-gray-700 hover:bg-gray-600 text-white rounded-lg"
          >
            Отмена
          </button>
          <button
            @click="handleReject"
            :disabled="!rejectReason.trim() || actionLoading"
            class="flex-1 py-2 px-4 bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
          >
            {{ actionLoading ? 'Отклонение...' : 'Отклонить' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
