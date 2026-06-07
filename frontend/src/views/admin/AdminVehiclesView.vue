<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api/admin'
import type { PendingVehicle } from '@/types/admin'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  type VehicleType,
  type VehicleSubType,
} from '@/types/freightRequest'

// UI Components
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
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
  Check,
  X,
  Headphones,
  Mail,
  Truck,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const admin = useAdminStore()

const vehicles = ref<PendingVehicle[]>([])
const isLoading = ref(true)
const error = ref('')

// Modal state
const showRejectModal = ref(false)
const selectedVehicle = ref<PendingVehicle | null>(null)
const rejectReason = ref('')
const isSubmitting = ref(false)
const verifyingId = ref<string | null>(null)

const navItems = [
  { to: '/admin/organizations', label: 'Организации', icon: Building2 },
  { to: '/admin/vehicles', label: 'Транспорт', icon: Truck },
  { to: '/admin/reviews', label: 'Отзывы', icon: Star },
  { to: '/admin/fraudsters', label: 'Накрутчики', icon: AlertTriangle },
  { to: '/admin/support', label: 'Поддержка', icon: Headphones },
  { to: '/admin/email-templates', label: 'Email шаблоны', icon: Mail },
]

onMounted(async () => {
  await loadVehicles()
})

async function loadVehicles() {
  isLoading.value = true
  error.value = ''
  try {
    vehicles.value = await adminApi.getPendingVehicles()
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

function formatVehicle(vehicle: PendingVehicle): string {
  const type = vehicleTypeLabels[vehicle.vehicle_type as VehicleType] ?? vehicle.vehicle_type
  const subtype = vehicleSubTypeLabels[vehicle.vehicle_subtype as VehicleSubType] ?? vehicle.vehicle_subtype
  return `${type} · ${subtype}`
}

async function submitVerify(vehicle: PendingVehicle) {
  if (verifyingId.value) return
  verifyingId.value = vehicle.id
  try {
    await adminApi.verifyVehicle(vehicle.org_id, vehicle.id)
    await loadVehicles()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка подтверждения'
  } finally {
    verifyingId.value = null
  }
}

function openRejectModal(vehicle: PendingVehicle) {
  selectedVehicle.value = vehicle
  rejectReason.value = ''
  showRejectModal.value = true
}

function closeModals() {
  showRejectModal.value = false
  selectedVehicle.value = null
}

async function submitReject() {
  if (!selectedVehicle.value || !rejectReason.value.trim()) return
  isSubmitting.value = true
  try {
    await adminApi.rejectVehicle(selectedVehicle.value.org_id, selectedVehicle.value.id, {
      reason: rejectReason.value.trim(),
    })
    closeModals()
    await loadVehicles()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка отклонения'
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
          <h2 class="text-2xl font-bold text-white">Транспорт на модерации</h2>
          <p class="text-sm text-slate-400 mt-1">Всего: {{ vehicles.length }}</p>
        </div>
        <Button
          variant="outline"
          class="border-slate-600 text-slate-300 hover:bg-slate-700 hover:text-white"
          :disabled="isLoading"
          @click="loadVehicles"
        >
          <RefreshCcw class="h-4 w-4 mr-2" :class="{ 'animate-spin': isLoading }" />
          Обновить
        </Button>
      </div>

      <!-- Error -->
      <ErrorBanner
        v-if="error"
        :message="error"
        @retry="loadVehicles"
        class="mb-6"
      />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <!-- Empty -->
      <Card v-else-if="vehicles.length === 0" class="bg-slate-800 border-slate-700">
        <CardContent class="py-12 text-center">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-700 mb-4">
            <Truck class="h-8 w-8 text-slate-400" />
          </div>
          <h3 class="text-lg font-medium text-white mb-2">Нет транспорта на модерации</h3>
          <p class="text-slate-400">Все заявки обработаны</p>
        </CardContent>
      </Card>

      <!-- List -->
      <div v-else class="space-y-4">
        <Card
          v-for="vehicle in vehicles"
          :key="vehicle.id"
          class="bg-slate-800 border-slate-700"
        >
          <CardContent class="p-6">
            <div class="flex flex-col lg:flex-row lg:justify-between lg:items-start gap-4">
              <div>
                <!-- Тип + гос. номер -->
                <div class="flex items-center gap-3 mb-2 flex-wrap">
                  <Truck class="h-5 w-5 text-slate-400 shrink-0" />
                  <span class="text-white font-medium">{{ formatVehicle(vehicle) }}</span>
                  <span class="font-mono font-semibold text-indigo-400">{{ vehicle.registration_number }}</span>
                </div>

                <!-- Meta -->
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-slate-400">
                  <span v-if="vehicle.brand || vehicle.model">
                    {{ [vehicle.brand, vehicle.model].filter(Boolean).join(' ') }}
                  </span>
                  <span>Отправлен: {{ formatDate(vehicle.submitted_at) }}</span>
                </div>
              </div>

              <!-- Actions -->
              <div class="flex gap-2 shrink-0">
                <Button
                  size="sm"
                  class="bg-green-600 hover:bg-green-500 text-white"
                  :disabled="verifyingId === vehicle.id"
                  @click="submitVerify(vehicle)"
                >
                  <Check class="h-4 w-4 mr-1" />
                  {{ verifyingId === vehicle.id ? 'Подтверждение...' : 'Подтвердить' }}
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  @click="openRejectModal(vehicle)"
                >
                  <X class="h-4 w-4 mr-1" />
                  Отклонить
                </Button>
              </div>
            </div>

            <!-- IDs -->
            <div class="mt-4 pt-4 border-t border-slate-700 text-xs text-slate-600 font-mono">
              Vehicle: {{ vehicle.id.slice(0, 8) }}... · Org: {{ vehicle.org_id.slice(0, 8) }}...
            </div>
          </CardContent>
        </Card>
      </div>
    </main>

    <!-- Reject Modal -->
    <Dialog v-model:open="showRejectModal">
      <DialogContent class="bg-slate-800 border-slate-700 text-white sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-white">Отклонить транспорт</DialogTitle>
          <DialogDescription class="text-slate-400">
            <template v-if="selectedVehicle">
              {{ formatVehicle(selectedVehicle) }} · {{ selectedVehicle.registration_number }}
            </template>
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-2">
          <Label class="text-slate-200">Причина отклонения</Label>
          <Textarea
            v-model="rejectReason"
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
            @click="closeModals"
          >
            Отмена
          </Button>
          <Button
            variant="destructive"
            :disabled="isSubmitting || !rejectReason.trim()"
            @click="submitReject"
          >
            {{ isSubmitting ? 'Сохранение...' : 'Отклонить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
