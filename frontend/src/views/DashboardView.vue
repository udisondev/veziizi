<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import OrgStatusBanner from '@/components/ui/OrgStatusBanner.vue'
import PermissionGuard from '@/components/ui/PermissionGuard.vue'

const auth = useAuthStore()
const { canManageMembers, canCreateFreightRequest } = usePermissions()
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <OrgStatusBanner />

    <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
      <h1 class="text-3xl font-bold text-gray-900 mb-8">
        Добро пожаловать, {{ auth.name }}!
      </h1>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <!-- Freight Requests -->
        <router-link
          to="/"
          class="bg-white overflow-hidden shadow rounded-lg p-6 hover:shadow-md transition-shadow"
        >
          <h3 class="text-lg font-medium text-gray-900">Заявки</h3>
          <p class="mt-1 text-sm text-gray-500">
            Просмотр и управление заявками на перевозку
          </p>
        </router-link>

        <!-- Create Request -->
        <PermissionGuard :condition="canCreateFreightRequest">
          <router-link
            to="/freight-requests/new"
            class="bg-blue-50 overflow-hidden shadow rounded-lg p-6 hover:shadow-md transition-shadow border-2 border-blue-200"
          >
            <h3 class="text-lg font-medium text-blue-900">+ Новая заявка</h3>
            <p class="mt-1 text-sm text-blue-700">
              Создать новую заявку на перевозку
            </p>
          </router-link>
        </PermissionGuard>

        <!-- My Offers -->
        <router-link
          to="/my-offers"
          class="bg-white overflow-hidden shadow rounded-lg p-6 hover:shadow-md transition-shadow"
        >
          <h3 class="text-lg font-medium text-gray-900">Предложения</h3>
          <p class="mt-1 text-sm text-gray-500">
            Предложения вашей организации
          </p>
        </router-link>

        <!-- Members (owner/admin only) -->
        <PermissionGuard :condition="canManageMembers">
          <router-link
            to="/members"
            class="bg-white overflow-hidden shadow rounded-lg p-6 hover:shadow-md transition-shadow"
          >
            <h3 class="text-lg font-medium text-gray-900">Сотрудники</h3>
            <p class="mt-1 text-sm text-gray-500">
              Управление сотрудниками организации
            </p>
          </router-link>
        </PermissionGuard>

        <!-- Invitations (owner/admin only) -->
        <PermissionGuard :condition="canManageMembers">
          <router-link
            to="/invitations"
            class="bg-white overflow-hidden shadow rounded-lg p-6 hover:shadow-md transition-shadow"
          >
            <h3 class="text-lg font-medium text-gray-900">Приглашения</h3>
            <p class="mt-1 text-sm text-gray-500">
              Управление приглашениями
            </p>
          </router-link>
        </PermissionGuard>
      </div>
    </div>
  </div>
</template>
