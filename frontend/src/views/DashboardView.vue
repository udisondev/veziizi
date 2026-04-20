<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { usePermissions } from '@/composables/usePermissions'
import OrgStatusBanner from '@/components/ui/OrgStatusBanner.vue'
import PermissionGuard from '@/components/ui/PermissionGuard.vue'

const auth = useAuthStore()
const { canManageMembers, canCreateFreightRequest } = usePermissions()

const cardClass = 'block rounded-xl border bg-card text-card-foreground shadow-md p-6 cursor-pointer transition-all duration-200 hover:scale-[1.02] hover:shadow-xl'
</script>

<template>
  <div class="min-h-screen bg-background">
    <OrgStatusBanner />

    <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
      <h1 class="text-3xl font-bold text-foreground mb-8">
        Добро пожаловать, {{ auth.name }}!
      </h1>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <!-- Freight Requests -->
        <router-link to="/" :class="cardClass">
          <h3 class="text-lg font-medium text-foreground">Заявки</h3>
          <p class="mt-1 text-sm text-muted-foreground">
            Просмотр и управление заявками на перевозку
          </p>
        </router-link>

        <!-- Create Request -->
        <PermissionGuard :condition="canCreateFreightRequest">
          <router-link to="/freight-requests/new" :class="[cardClass, 'border-primary/30 bg-accent']">
            <h3 class="text-lg font-medium text-accent-foreground">+ Новая заявка</h3>
            <p class="mt-1 text-sm text-accent-foreground/70">
              Создать новую заявку на перевозку
            </p>
          </router-link>
        </PermissionGuard>

        <!-- My Offers -->
        <router-link to="/my-offers" :class="cardClass">
          <h3 class="text-lg font-medium text-foreground">Предложения</h3>
          <p class="mt-1 text-sm text-muted-foreground">
            Предложения вашей организации
          </p>
        </router-link>

        <!-- Members (owner/admin only) -->
        <PermissionGuard :condition="canManageMembers">
          <router-link to="/members" :class="cardClass">
            <h3 class="text-lg font-medium text-foreground">Сотрудники</h3>
            <p class="mt-1 text-sm text-muted-foreground">
              Управление сотрудниками организации
            </p>
          </router-link>
        </PermissionGuard>

        <!-- Invitations (owner/admin only) -->
        <PermissionGuard :condition="canManageMembers">
          <router-link to="/invitations" :class="cardClass">
            <h3 class="text-lg font-medium text-foreground">Приглашения</h3>
            <p class="mt-1 text-sm text-muted-foreground">
              Управление приглашениями
            </p>
          </router-link>
        </PermissionGuard>
      </div>
    </div>
  </div>
</template>
