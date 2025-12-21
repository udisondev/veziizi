<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'

// Shared Components
import { PageHeader } from '@/components/shared'

// Icons
import { User, Mail, Phone, Shield, Building2, LogOut } from 'lucide-vue-next'

const auth = useAuthStore()
const router = useRouter()

const userInitial = computed(() => {
  return auth.name?.charAt(0).toUpperCase() || '?'
})

const roleLabels: Record<string, string> = {
  owner: 'Владелец',
  administrator: 'Администратор',
  employee: 'Сотрудник',
}

const roleBadgeVariant = computed(() => {
  if (auth.role === 'owner') return 'default'
  if (auth.role === 'administrator') return 'secondary'
  return 'outline'
})

const roleLabel = computed(() => {
  return auth.role ? (roleLabels[auth.role] || auth.role) : 'Неизвестно'
})

async function logout() {
  await auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <PageHeader title="Профиль" class="mb-6" />

    <div class="grid gap-6 md:grid-cols-2">
      <!-- User Info Card -->
      <Card>
        <CardHeader>
          <div class="flex items-center gap-4">
            <Avatar class="h-16 w-16">
              <AvatarFallback class="bg-primary/10 text-primary text-2xl">
                {{ userInitial }}
              </AvatarFallback>
            </Avatar>
            <div>
              <CardTitle>{{ auth.name }}</CardTitle>
              <CardDescription>{{ auth.email }}</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <Separator />

          <div class="space-y-4">
            <div class="flex items-center gap-3">
              <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <User class="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p class="text-sm text-muted-foreground">Имя</p>
                <p class="font-medium">{{ auth.name }}</p>
              </div>
            </div>

            <div class="flex items-center gap-3">
              <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <Mail class="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p class="text-sm text-muted-foreground">Email</p>
                <p class="font-medium">{{ auth.email }}</p>
              </div>
            </div>

            <div v-if="auth.phone" class="flex items-center gap-3">
              <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <Phone class="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p class="text-sm text-muted-foreground">Телефон</p>
                <p class="font-medium">{{ auth.phone }}</p>
              </div>
            </div>

            <div class="flex items-center gap-3">
              <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <Shield class="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p class="text-sm text-muted-foreground">Роль</p>
                <Badge :variant="roleBadgeVariant">
                  {{ roleLabel }}
                </Badge>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Organization Card -->
      <Card>
        <CardHeader>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Building2 class="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle class="text-lg">Организация</CardTitle>
              <CardDescription>Ваша организация</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div v-if="auth.organization" class="space-y-4">
            <div>
              <p class="text-sm text-muted-foreground">Название</p>
              <p class="font-medium">{{ auth.organization.name }}</p>
            </div>
          </div>
          <p v-else class="text-muted-foreground">
            Организация не найдена
          </p>
        </CardContent>
      </Card>
    </div>

    <!-- Logout Section -->
    <Card class="mt-6">
      <CardContent class="flex items-center justify-between py-4">
        <div>
          <p class="font-medium">Выход из системы</p>
          <p class="text-sm text-muted-foreground">
            Завершить текущую сессию
          </p>
        </div>
        <Button variant="destructive" @click="logout">
          <LogOut class="mr-2 h-4 w-4" />
          Выйти
        </Button>
      </CardContent>
    </Card>
  </div>
</template>
