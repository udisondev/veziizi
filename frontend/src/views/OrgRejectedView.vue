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
  <div class="min-h-screen flex items-center justify-center bg-red-50">
    <div class="max-w-md w-full text-center space-y-6 p-8">
      <div class="text-6xl">❌</div>
      <h1 class="text-2xl font-bold text-gray-900">Заявка отклонена</h1>
      <p class="text-gray-600">
        К сожалению, заявка организации <strong>{{ auth.organization?.name }}</strong> была отклонена.
        Пожалуйста, свяжитесь с поддержкой для уточнения причины.
      </p>
      <button
        @click="logout"
        class="text-gray-500 hover:text-gray-700 underline"
      >
        Выйти
      </button>
    </div>
  </div>
</template>
