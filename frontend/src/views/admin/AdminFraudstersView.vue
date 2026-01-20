<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { Fraudster } from '@/types/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// Shared Components
import { ErrorBanner } from '@/components/shared'

// Icons
import {
  Building2,
  RefreshCcw,
  LogOut,
  Star,
  AlertTriangle,
  ShieldOff,
  Headphones,
  Mail,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const admin = useAdminStore()

const fraudsters = ref<Fraudster[]>([])
const total = ref(0)
const isLoading = ref(true)
const error = ref('')

// Modal state
const showUnmarkModal = ref(false)
const selectedFraudster = ref<Fraudster | null>(null)
const unmarkReason = ref('')
const isSubmitting = ref(false)

const navItems = [
  { to: '/admin/organizations', label: 'Организации', icon: Building2 },
  { to: '/admin/reviews', label: 'Отзывы', icon: Star },
  { to: '/admin/fraudsters', label: 'Накрутчики', icon: AlertTriangle },
  { to: '/admin/support', label: 'Поддержка', icon: Headphones },
  { to: '/admin/email-templates', label: 'Email шаблоны', icon: Mail },
]

onMounted(async () => {
  await loadFraudsters()
})

async function loadFraudsters() {
  isLoading.value = true
  error.value = ''
  try {
    const response = await adminApi.getFraudsters()
    fraudsters.value = response.fraudsters ?? []
    total.value = response.total
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

function openUnmarkModal(fraudster: Fraudster) {
  selectedFraudster.value = fraudster
  unmarkReason.value = ''
  showUnmarkModal.value = true
}

function closeModal() {
  showUnmarkModal.value = false
  selectedFraudster.value = null
}

async function submitUnmark() {
  if (!selectedFraudster.value || !unmarkReason.value.trim()) return
  isSubmitting.value = true
  try {
    await adminApi.unmarkFraudster(selectedFraudster.value.org_id, {
      reason: unmarkReason.value.trim(),
    })
    closeModal()
    await loadFraudsters()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка снятия метки'
  } finally {
    isSubmitting.value = false
  }
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
          <h2 class="text-2xl font-bold text-white">Накрутчики</h2>
          <p class="text-sm text-slate-400 mt-1">Всего: {{ total }}</p>
        </div>
        <Button
          variant="outline"
          class="border-slate-600 text-slate-300 hover:bg-slate-700 hover:text-white"
          :disabled="isLoading"
          @click="loadFraudsters"
        >
          <RefreshCcw class="h-4 w-4 mr-2" :class="{ 'animate-spin': isLoading }" />
          Обновить
        </Button>
      </div>

      <!-- Error -->
      <ErrorBanner
        v-if="error"
        :message="error"
        @retry="loadFraudsters"
        class="mb-6"
      />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Empty -->
      <Card v-else-if="fraudsters.length === 0" class="bg-slate-800 border-slate-700">
        <CardContent class="py-12 text-center">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-700 mb-4">
            <AlertTriangle class="h-8 w-8 text-slate-400" />
          </div>
          <h3 class="text-lg font-medium text-white mb-2">Нет отмеченных накрутчиков</h3>
          <p class="text-slate-400">Организации с подозрительной активностью пока не обнаружены</p>
        </CardContent>
      </Card>

      <!-- List -->
      <div v-else class="space-y-4">
        <Card
          v-for="fraudster in fraudsters"
          :key="fraudster.org_id"
          class="bg-slate-800 border-slate-700"
        >
          <CardContent class="p-6">
            <div class="flex flex-col lg:flex-row lg:justify-between lg:items-start gap-4 mb-4">
              <div>
                <!-- Name + Status -->
                <div class="flex items-center gap-3 mb-2">
                  <span class="text-lg font-medium text-white">{{ fraudster.org_name }}</span>
                  <Badge
                    :variant="fraudster.is_confirmed ? 'destructive' : 'warning'"
                  >
                    {{ fraudster.is_confirmed ? 'Подтверждённый' : 'Подозреваемый' }}
                  </Badge>
                </div>

                <!-- Reason -->
                <p v-if="fraudster.reason" class="text-slate-300 mb-3">
                  <span class="text-slate-500">Причина:</span> {{ fraudster.reason }}
                </p>

                <!-- Stats -->
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-slate-400">
                  <span>Оставлено отзывов: {{ fraudster.total_reviews_left }}</span>
                  <span>Деактивировано: {{ fraudster.deactivated_reviews }}</span>
                  <span>Репутация: {{ (fraudster.reputation_score * 100).toFixed(0) }}%</span>
                </div>

                <!-- Date -->
                <div class="text-sm text-slate-500 mt-2">
                  Отмечен: {{ formatDate(fraudster.marked_at) }}
                </div>
              </div>

              <!-- Actions -->
              <div class="shrink-0">
                <Button
                  size="sm"
                  class="bg-green-600 hover:bg-green-500 text-white"
                  @click="openUnmarkModal(fraudster)"
                >
                  <ShieldOff class="h-4 w-4 mr-1" />
                  Снять метку
                </Button>
              </div>
            </div>

            <!-- ID -->
            <div class="mt-4 pt-4 border-t border-slate-700 text-xs text-slate-600 font-mono">
              ID: {{ fraudster.org_id }}
            </div>
          </CardContent>
        </Card>
      </div>
    </main>

    <!-- Unmark Modal -->
    <Dialog v-model:open="showUnmarkModal">
      <DialogContent class="bg-slate-800 border-slate-700 text-white sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-white">Снять метку накрутчика</DialogTitle>
          <DialogDescription class="text-slate-400">
            Организация: {{ selectedFraudster?.org_name }}
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label class="text-slate-200">Причина снятия метки</Label>
          <Textarea
            v-model="unmarkReason"
            rows="3"
            class="bg-slate-700 border-slate-600 text-white resize-none"
            placeholder="Укажите причину..."
          />
        </div>

        <DialogFooter>
          <Button
            variant="ghost"
            class="text-slate-400 hover:text-white"
            :disabled="isSubmitting"
            @click="closeModal"
          >
            Отмена
          </Button>
          <Button
            class="bg-green-600 hover:bg-green-500 text-white"
            :disabled="isSubmitting || !unmarkReason.trim()"
            @click="submitUnmark"
          >
            {{ isSubmitting ? 'Сохранение...' : 'Снять метку' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
