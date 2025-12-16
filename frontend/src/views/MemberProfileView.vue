<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { membersApi, type MemberProfile } from '@/api/members'

const route = useRoute()
const router = useRouter()

const member = ref<MemberProfile | null>(null)
const isLoading = ref(true)
const error = ref('')

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
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-white shadow">
      <div class="max-w-4xl mx-auto px-4 py-4">
        <button
          @click="router.back()"
          class="text-blue-600 hover:text-blue-800 text-sm"
        >
          &larr; Назад
        </button>
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
          <h1 class="text-2xl font-bold text-gray-900">{{ member.name }}</h1>
          <p class="text-gray-500 mt-1">Сотрудник</p>
        </div>

        <!-- Details Card -->
        <div class="bg-white rounded-lg shadow p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Информация</h2>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <dt class="text-sm text-gray-500">ФИО</dt>
              <dd class="text-gray-900 font-medium">{{ member.name }}</dd>
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
                  class="text-blue-600 hover:text-blue-800 hover:underline"
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
  </div>
</template>
