<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()

async function logout() {
  await auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-100 p-6">
    <div class="max-w-2xl mx-auto bg-white rounded-lg shadow p-6">
      <h1 class="text-2xl font-bold mb-6">Профиль</h1>

      <dl class="space-y-4">
        <div>
          <dt class="text-sm text-gray-500">Имя</dt>
          <dd class="text-lg">{{ auth.name }}</dd>
        </div>
        <div>
          <dt class="text-sm text-gray-500">Email</dt>
          <dd class="text-lg">{{ auth.email }}</dd>
        </div>
        <div v-if="auth.phone">
          <dt class="text-sm text-gray-500">Телефон</dt>
          <dd class="text-lg">{{ auth.phone }}</dd>
        </div>
        <div>
          <dt class="text-sm text-gray-500">Роль</dt>
          <dd class="text-lg">{{ auth.role }}</dd>
        </div>
        <div>
          <dt class="text-sm text-gray-500">Организация</dt>
          <dd class="text-lg">{{ auth.organization?.name }}</dd>
        </div>
      </dl>

      <button
        @click="logout"
        class="mt-8 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
      >
        Выйти
      </button>
    </div>
  </div>
</template>
