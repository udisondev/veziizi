<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const { canManageMembers } = usePermissions()

const isMenuOpen = ref(false)

const menuItems = computed(() => {
  const items = [
    { to: '/', label: 'Заявки', icon: '📦' },
    { to: '/orders', label: 'Заказы', icon: '📋' },
    { to: '/my-offers', label: 'Мои офферы', icon: '💰' },
    { to: '/members', label: 'Сотрудники', icon: '👥' },
  ]

  if (canManageMembers.value) {
    items.push(
      { to: '/invitations', label: 'Приглашения', icon: '✉️' },
      { to: '/organization/settings', label: 'Настройки', icon: '⚙️' },
    )
  }

  items.push({ to: '/profile', label: 'Профиль', icon: '👤' })

  return items
})

function toggleMenu() {
  isMenuOpen.value = !isMenuOpen.value
}

function closeMenu() {
  isMenuOpen.value = false
}

function navigate(to: string) {
  router.push(to)
  closeMenu()
}

async function logout() {
  await auth.logout()
  router.push('/login')
}
</script>

<template>
  <header class="bg-white shadow-sm border-b border-gray-200 sticky top-0 z-50">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-14">
        <!-- Left: Menu button + Title -->
        <div class="flex items-center gap-3">
          <!-- Menu button -->
          <div class="relative">
            <button
              @click="toggleMenu"
              class="p-2 rounded-md text-gray-600 hover:text-gray-900 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>

            <!-- Dropdown menu -->
            <div
              v-if="isMenuOpen"
              class="absolute left-0 mt-2 w-56 max-w-[calc(100vw-2rem)] rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 z-50"
            >
              <div class="py-1">
                <button
                  v-for="item in menuItems"
                  :key="item.to"
                  @click="navigate(item.to)"
                  :class="[
                    'w-full text-left px-4 py-2 text-sm flex items-center gap-3',
                    route.path === item.to
                      ? 'bg-blue-50 text-blue-700'
                      : 'text-gray-700 hover:bg-gray-100'
                  ]"
                >
                  <span>{{ item.icon }}</span>
                  <span>{{ item.label }}</span>
                </button>

                <div class="border-t border-gray-100 my-1"></div>

                <button
                  @click="logout"
                  class="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 flex items-center gap-3"
                >
                  <span>🚪</span>
                  <span>Выйти</span>
                </button>
              </div>
            </div>
          </div>

          <!-- Logo/Title -->
          <router-link to="/" class="text-lg font-semibold text-gray-900">
            Veziizi
          </router-link>
        </div>

        <!-- Right: User info -->
        <div class="flex items-center gap-4">
          <span class="text-sm text-gray-600 hidden sm:block">
            {{ auth.organization?.name }}
          </span>
          <div class="h-8 w-8 rounded-full bg-blue-100 flex items-center justify-center text-blue-600 font-medium text-sm">
            {{ auth.name?.charAt(0).toUpperCase() }}
          </div>
        </div>
      </div>
    </div>

    <!-- Overlay to close menu -->
    <div
      v-if="isMenuOpen"
      @click="closeMenu"
      class="fixed inset-0 z-[-1]"
    ></div>
  </header>
</template>
