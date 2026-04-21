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
  <div class="min-h-screen flex items-center justify-center bg-destructive/5">
    <div class="max-w-md w-full text-center space-y-6 p-8">
      <div class="text-6xl">❌</div>
      <h1 class="text-2xl font-bold text-foreground">Заявка отклонена</h1>
      <p class="text-muted-foreground">
        К сожалению, заявка организации <strong>{{ auth.organization?.name }}</strong> была отклонена.
        Пожалуйста, свяжитесь с поддержкой для уточнения причины.
      </p>
      <button
        @click="logout"
        class="text-muted-foreground hover:text-foreground underline"
      >
        Выйти
      </button>
    </div>
  </div>
</template>
