<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { computed } from 'vue'

const authStore = useAuthStore()

// Показываем баннер только если пользователь залогинен И заблокирован
const show = computed(() => authStore.isAuthenticated && authStore.isBlocked)
</script>

<template>
  <Transition name="slide-down">
    <div
      v-if="show"
      class="fixed top-0 left-0 right-0 z-[100] bg-red-600 text-white shadow-lg"
    >
      <div class="container mx-auto px-4 py-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-4">
            <!-- Icon -->
            <div
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-red-700"
            >
              <svg
                class="h-6 w-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>

            <!-- Message -->
            <div class="flex-1">
              <h3 class="text-lg font-semibold">Ваш аккаунт заблокирован</h3>
              <p class="text-sm text-red-100">
                Доступ к функциям платформы ограничен. Пожалуйста, свяжитесь с
                администратором вашей организации или с поддержкой для
                получения дополнительной информации.
              </p>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-3">
            <button
              @click="authStore.logout"
              class="rounded-lg bg-red-700 px-4 py-2 text-sm font-medium transition-colors hover:bg-red-800"
            >
              Выйти
            </button>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.slide-down-enter-active {
  transition: all 0.3s ease-out;
}

.slide-down-leave-active {
  transition: all 0.2s ease-in;
}

.slide-down-enter-from {
  transform: translateY(-100%);
  opacity: 0;
}

.slide-down-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}
</style>
