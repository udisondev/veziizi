<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { PendingOrganization } from '@/types/admin'

const router = useRouter()
const admin = useAdminStore()

const organizations = ref<PendingOrganization[]>([])
const isLoading = ref(true)
const error = ref('')

const countryNames: Record<string, string> = {
  RU: 'Россия',
  KZ: 'Казахстан',
  BY: 'Беларусь',
}

onMounted(async () => {
  await loadOrganizations()
})

async function loadOrganizations() {
  isLoading.value = true
  error.value = ''
  try {
    organizations.value = await adminApi.getOrganizations()
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

async function handleLogout() {
  await admin.logout()
  router.push('/admin/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-900">
    <!-- Header -->
    <header class="bg-gray-800 shadow">
      <div class="max-w-7xl mx-auto px-4 py-4 flex justify-between items-center">
        <h1 class="text-xl font-bold text-white">Панель администратора</h1>
        <div class="flex items-center gap-4">
          <span class="text-gray-400 text-sm">{{ admin.email }}</span>
          <button
            @click="handleLogout"
            class="text-gray-400 hover:text-white text-sm"
          >
            Выйти
          </button>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-7xl mx-auto px-4 py-8">
      <div class="flex justify-between items-center mb-6">
        <h2 class="text-2xl font-bold text-white">Организации на модерации</h2>
        <button
          @click="loadOrganizations"
          :disabled="isLoading"
          class="px-4 py-2 text-sm bg-gray-700 text-white rounded hover:bg-gray-600 disabled:opacity-50"
        >
          Обновить
        </button>
      </div>

      <!-- Error -->
      <div v-if="error" class="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded mb-6">
        {{ error }}
      </div>

      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="text-gray-400">Загрузка...</div>
      </div>

      <!-- Empty -->
      <div v-else-if="organizations.length === 0" class="text-center py-12">
        <div class="text-gray-400 text-lg">Нет организаций на модерации</div>
        <p class="text-gray-500 mt-2">Все заявки обработаны</p>
      </div>

      <!-- List -->
      <div v-else class="bg-gray-800 rounded-lg overflow-hidden">
        <table class="min-w-full divide-y divide-gray-700">
          <thead class="bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                Организация
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                ИНН
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                Страна
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                Email
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                Дата
              </th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-300 uppercase tracking-wider">
                Действия
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-700">
            <tr v-for="org in organizations" :key="org.id" class="hover:bg-gray-700/50">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-white">{{ org.name }}</div>
                <div class="text-sm text-gray-400">{{ org.legal_name }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">
                {{ org.inn }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">
                {{ countryNames[org.country] }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">
                {{ org.email }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-400">
                {{ formatDate(org.created_at) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm">
                <router-link
                  :to="`/admin/organizations/${org.id}`"
                  class="text-indigo-400 hover:text-indigo-300"
                >
                  Подробнее
                </router-link>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </main>
  </div>
</template>
