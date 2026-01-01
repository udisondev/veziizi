<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { PendingOrganization } from '@/types/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

// Shared Components
import { ErrorBanner } from '@/components/shared'

// Icons
import {
  Building2,
  RefreshCcw,
  LogOut,
  ClipboardList,
  Star,
  AlertTriangle,
  ExternalLink,
  Headphones,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const admin = useAdminStore()

const organizations = ref<PendingOrganization[]>([])
const isLoading = ref(true)
const error = ref('')

const countryNames: Record<string, string> = {
  RU: 'Россия',
  KZ: 'Казахстан',
  BY: 'Беларусь',
}

const navItems = [
  { to: '/admin/organizations', label: 'Организации', icon: Building2 },
  { to: '/admin/reviews', label: 'Отзывы', icon: Star },
  { to: '/admin/fraudsters', label: 'Накрутчики', icon: AlertTriangle },
  { to: '/admin/support', label: 'Поддержка', icon: Headphones },
]

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

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}
</script>

<template>
  <div class="min-h-screen bg-slate-900">
    <!-- Header -->
    <header class="bg-slate-800 border-b border-slate-700 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-14">
          <!-- Left: Logo + Nav -->
          <div class="flex items-center gap-6">
            <h1 class="text-lg font-semibold text-white">Admin Panel</h1>
            <nav class="hidden md:flex items-center gap-1">
              <router-link
                v-for="item in navItems"
                :key="item.to"
                :to="item.to"
                :class="[
                  'px-3 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition-colors',
                  isActive(item.to)
                    ? 'bg-indigo-500/20 text-indigo-400'
                    : 'text-slate-400 hover:text-white hover:bg-slate-700'
                ]"
              >
                <component :is="item.icon" class="h-4 w-4" />
                {{ item.label }}
              </router-link>
            </nav>
          </div>

          <!-- Right: User info + Logout -->
          <div class="flex items-center gap-4">
            <span class="text-sm text-slate-400 hidden sm:block">{{ admin.email }}</span>
            <Button
              variant="ghost"
              size="sm"
              class="text-slate-400 hover:text-white hover:bg-slate-700"
              @click="handleLogout"
            >
              <LogOut class="h-4 w-4 mr-2" />
              Выйти
            </Button>
          </div>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h2 class="text-2xl font-bold text-white">Организации на модерации</h2>
          <p class="text-sm text-slate-400 mt-1">
            Заявки на регистрацию организаций
          </p>
        </div>
        <Button
          variant="outline"
          class="border-slate-600 text-slate-300 hover:bg-slate-700 hover:text-white"
          :disabled="isLoading"
          @click="loadOrganizations"
        >
          <RefreshCcw class="h-4 w-4 mr-2" :class="{ 'animate-spin': isLoading }" />
          Обновить
        </Button>
      </div>

      <!-- Error -->
      <ErrorBanner
        v-if="error"
        :message="error"
        @retry="loadOrganizations"
        class="mb-6"
      />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Empty -->
      <Card v-else-if="organizations.length === 0" class="bg-slate-800 border-slate-700">
        <CardContent class="py-12 text-center">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-700 mb-4">
            <ClipboardList class="h-8 w-8 text-slate-400" />
          </div>
          <h3 class="text-lg font-medium text-white mb-2">Нет организаций на модерации</h3>
          <p class="text-slate-400">Все заявки обработаны</p>
        </CardContent>
      </Card>

      <!-- Table -->
      <Card v-else class="bg-slate-800 border-slate-700">
        <Table>
          <TableHeader>
            <TableRow class="border-slate-700 hover:bg-transparent">
              <TableHead class="text-slate-300">Организация</TableHead>
              <TableHead class="text-slate-300">ИНН</TableHead>
              <TableHead class="text-slate-300">Страна</TableHead>
              <TableHead class="text-slate-300">Email</TableHead>
              <TableHead class="text-slate-300">Дата</TableHead>
              <TableHead class="text-slate-300 text-right">Действия</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow
              v-for="org in organizations"
              :key="org.id"
              class="border-slate-700 hover:bg-slate-700/50"
            >
              <TableCell>
                <div class="text-sm font-medium text-white">{{ org.name }}</div>
                <div class="text-sm text-slate-400">{{ org.legal_name }}</div>
              </TableCell>
              <TableCell class="text-slate-300 font-mono">
                {{ org.inn }}
              </TableCell>
              <TableCell class="text-slate-300">
                {{ countryNames[org.country] }}
              </TableCell>
              <TableCell class="text-slate-300">
                {{ org.email }}
              </TableCell>
              <TableCell class="text-slate-400">
                {{ formatDate(org.created_at) }}
              </TableCell>
              <TableCell class="text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10"
                  as-child
                >
                  <router-link :to="`/admin/organizations/${org.id}`">
                    Подробнее
                    <ExternalLink class="h-3.5 w-3.5 ml-1" />
                  </router-link>
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </Card>
    </main>
  </div>
</template>
