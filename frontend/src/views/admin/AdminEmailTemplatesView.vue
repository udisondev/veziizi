<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { EmailTemplate } from '@/types/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'

// Shared Components
import { ErrorBanner } from '@/components/shared'

// Icons
import {
  Building2,
  RefreshCcw,
  LogOut,
  Star,
  AlertTriangle,
  Headphones,
  Mail,
  Plus,
  Search,
  Pencil,
  Trash2,
  FileText,
  Lock,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const admin = useAdminStore()

const templates = ref<EmailTemplate[]>([])
const total = ref(0)
const isLoading = ref(true)
const error = ref('')
const searchQuery = ref('')
const categoryFilter = ref<string>('all')
const statusFilter = ref<string>('all')

const navItems = [
  { to: '/admin/organizations', label: 'Организации', icon: Building2 },
  { to: '/admin/reviews', label: 'Отзывы', icon: Star },
  { to: '/admin/fraudsters', label: 'Накрутчики', icon: AlertTriangle },
  { to: '/admin/support', label: 'Поддержка', icon: Headphones },
  { to: '/admin/email-templates', label: 'Email шаблоны', icon: Mail },
]

onMounted(async () => {
  await loadTemplates()
})

async function loadTemplates() {
  isLoading.value = true
  error.value = ''
  try {
    const filter: {
      category?: 'transactional' | 'marketing'
      is_active?: boolean
      search?: string
    } = {}

    if (categoryFilter.value !== 'all') {
      filter.category = categoryFilter.value as 'transactional' | 'marketing'
    }
    if (statusFilter.value !== 'all') {
      filter.is_active = statusFilter.value === 'active'
    }
    if (searchQuery.value.trim()) {
      filter.search = searchQuery.value.trim()
    }

    const result = await adminApi.getEmailTemplates(filter)
    templates.value = result.templates
    total.value = result.total
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

function editTemplate(id: string) {
  router.push(`/admin/email-templates/${id}`)
}

function createTemplate() {
  router.push('/admin/email-templates/new')
}

async function deleteTemplate(template: EmailTemplate) {
  if (template.is_system) {
    alert('Системные шаблоны нельзя удалить')
    return
  }

  if (!confirm(`Удалить шаблон "${template.name}"?`)) {
    return
  }

  try {
    await adminApi.deleteEmailTemplate(template.id)
    await loadTemplates()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка удаления'
  }
}

const categoryLabels: Record<string, string> = {
  transactional: 'Транзакционный',
  marketing: 'Маркетинговый',
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
          <h2 class="text-2xl font-bold text-white">Email шаблоны</h2>
          <p class="text-sm text-slate-400 mt-1">
            Управление шаблонами email-уведомлений
          </p>
        </div>
        <div class="flex items-center gap-2">
          <Button
            variant="outline"
            class="border-slate-600 text-slate-300 hover:bg-slate-700 hover:text-white"
            :disabled="isLoading"
            @click="loadTemplates"
          >
            <RefreshCcw class="h-4 w-4 mr-2" :class="{ 'animate-spin': isLoading }" />
            Обновить
          </Button>
          <Button
            class="bg-indigo-600 hover:bg-indigo-700"
            @click="createTemplate"
          >
            <Plus class="h-4 w-4 mr-2" />
            Создать шаблон
          </Button>
        </div>
      </div>

      <!-- Filters -->
      <Card class="bg-slate-800 border-slate-700 mb-6">
        <CardContent class="pt-6">
          <div class="flex flex-col sm:flex-row gap-4">
            <div class="flex-1">
              <div class="relative">
                <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                <Input
                  v-model="searchQuery"
                  type="text"
                  placeholder="Поиск по названию или slug..."
                  class="pl-10 bg-slate-900 border-slate-600 text-white placeholder:text-slate-500"
                  @keyup.enter="loadTemplates"
                />
              </div>
            </div>
            <Select v-model="categoryFilter" @update:model-value="loadTemplates">
              <SelectTrigger class="w-[180px] bg-slate-900 border-slate-600 text-white">
                <SelectValue placeholder="Категория" />
              </SelectTrigger>
              <SelectContent class="bg-slate-800 border-slate-600">
                <SelectItem value="all" class="text-white hover:bg-slate-700">Все категории</SelectItem>
                <SelectItem value="transactional" class="text-white hover:bg-slate-700">Транзакционные</SelectItem>
                <SelectItem value="marketing" class="text-white hover:bg-slate-700">Маркетинговые</SelectItem>
              </SelectContent>
            </Select>
            <Select v-model="statusFilter" @update:model-value="loadTemplates">
              <SelectTrigger class="w-[150px] bg-slate-900 border-slate-600 text-white">
                <SelectValue placeholder="Статус" />
              </SelectTrigger>
              <SelectContent class="bg-slate-800 border-slate-600">
                <SelectItem value="all" class="text-white hover:bg-slate-700">Все</SelectItem>
                <SelectItem value="active" class="text-white hover:bg-slate-700">Активные</SelectItem>
                <SelectItem value="inactive" class="text-white hover:bg-slate-700">Неактивные</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      <!-- Error -->
      <ErrorBanner
        v-if="error"
        :message="error"
        @retry="loadTemplates"
        class="mb-6"
      />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Empty -->
      <Card v-else-if="templates.length === 0" class="bg-slate-800 border-slate-700">
        <CardContent class="py-12 text-center">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-700 mb-4">
            <FileText class="h-8 w-8 text-slate-400" />
          </div>
          <h3 class="text-lg font-medium text-white mb-2">Нет шаблонов</h3>
          <p class="text-slate-400 mb-4">Создайте первый email-шаблон</p>
          <Button class="bg-indigo-600 hover:bg-indigo-700" @click="createTemplate">
            <Plus class="h-4 w-4 mr-2" />
            Создать шаблон
          </Button>
        </CardContent>
      </Card>

      <!-- Table -->
      <Card v-else class="bg-slate-800 border-slate-700">
        <Table>
          <TableHeader>
            <TableRow class="border-slate-700 hover:bg-transparent">
              <TableHead class="text-slate-300">Название</TableHead>
              <TableHead class="text-slate-300">Slug</TableHead>
              <TableHead class="text-slate-300">Категория</TableHead>
              <TableHead class="text-slate-300">Статус</TableHead>
              <TableHead class="text-slate-300">Обновлён</TableHead>
              <TableHead class="text-slate-300 text-right">Действия</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow
              v-for="template in templates"
              :key="template.id"
              class="border-slate-700 hover:bg-slate-700/50"
            >
              <TableCell>
                <div class="flex items-center gap-2">
                  <Lock v-if="template.is_system" class="h-4 w-4 text-slate-500" title="Системный шаблон" />
                  <span class="text-sm font-medium text-white">{{ template.name }}</span>
                </div>
              </TableCell>
              <TableCell class="text-slate-300 font-mono text-sm">
                {{ template.slug }}
              </TableCell>
              <TableCell>
                <Badge
                  :class="template.category === 'transactional'
                    ? 'bg-blue-500/20 text-blue-400 border-blue-500/30'
                    : 'bg-purple-500/20 text-purple-400 border-purple-500/30'"
                >
                  {{ categoryLabels[template.category] }}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge
                  :class="template.is_active
                    ? 'bg-green-500/20 text-green-400 border-green-500/30'
                    : 'bg-slate-500/20 text-slate-400 border-slate-500/30'"
                >
                  {{ template.is_active ? 'Активен' : 'Неактивен' }}
                </Badge>
              </TableCell>
              <TableCell class="text-slate-400 text-sm">
                {{ formatDate(template.updated_at) }}
              </TableCell>
              <TableCell class="text-right">
                <div class="flex items-center justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10"
                    @click="editTemplate(template.id)"
                  >
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <Button
                    v-if="!template.is_system"
                    variant="ghost"
                    size="sm"
                    class="text-red-400 hover:text-red-300 hover:bg-red-500/10"
                    @click="deleteTemplate(template)"
                  >
                    <Trash2 class="h-4 w-4" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </Card>

      <!-- Total count -->
      <div v-if="templates.length > 0" class="mt-4 text-sm text-slate-400">
        Всего: {{ total }} {{ total === 1 ? 'шаблон' : 'шаблонов' }}
      </div>
    </main>
  </div>
</template>
