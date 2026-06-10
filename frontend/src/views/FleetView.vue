<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { vehiclesApi } from '@/api/vehicles'
import { freightRequestsApi } from '@/api/freightRequests'
import type { Vehicle } from '@/types/vehicle'
import type { CarrierInviteItem } from '@/types/freightRequest'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
} from '@/types/freightRequest'
import { vehicleStatusMap } from '@/constants/statusMaps'

import { Button } from '@/components/ui/button'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter,
} from '@/components/ui/dialog'
import { LoadingSpinner, StatusBadge } from '@/components/shared'
import { useToast } from '@/components/ui/toast/use-toast'
import { logger } from '@/utils/logger'
import { Truck, Plus, Trash2, AlertCircle, ChevronRight, BadgeCheck, Mail } from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()
const { toast } = useToast()

const vehicles = ref<Vehicle[]>([])
const invitations = ref<CarrierInviteItem[]>([])
const isLoading = ref(false)
const loadError = ref<string | null>(null)

// Количество активных приглашений на каждое ТС
const inviteCountByVehicle = computed(() => {
  const map = new Map<string, number>()
  for (const inv of invitations.value) {
    if (inv.freight_status === 'published') {
      map.set(inv.vehicle_id, (map.get(inv.vehicle_id) ?? 0) + 1)
    }
  }
  return map
})

async function loadVehicles() {
  if (!auth.organizationId) return
  isLoading.value = true
  loadError.value = null
  try {
    const [vehiclesRes, invitationsRes] = await Promise.allSettled([
      vehiclesApi.list(auth.organizationId),
      freightRequestsApi.listCarrierInvitations({ limit: 100 }),
    ])
    if (vehiclesRes.status === 'fulfilled') {
      vehicles.value = vehiclesRes.value
    } else {
      loadError.value = 'Не удалось загрузить автопарк'
    }
    if (invitationsRes.status === 'fulfilled') {
      invitations.value = invitationsRes.value.items
    }
  } finally {
    isLoading.value = false
  }
}

loadVehicles()

const deletingVehicle = ref<Vehicle | null>(null)
const isDeleting = ref(false)

async function confirmDelete() {
  if (!deletingVehicle.value || !auth.organizationId) return
  isDeleting.value = true
  try {
    await vehiclesApi.remove(auth.organizationId, deletingVehicle.value.id)
    deletingVehicle.value = null
    await loadVehicles()
  } catch (e) {
    logger.error('Failed to remove vehicle', e)
    toast({ title: 'Не удалось убрать транспорт из автопарка', variant: 'destructive' })
  } finally {
    isDeleting.value = false
  }
}

const submittingId = ref<string | null>(null)

// Отправить ТС на модерацию (unconfirmed|rejected → pending)
async function submitVehicle(vehicle: Vehicle) {
  if (!auth.organizationId || submittingId.value) return
  submittingId.value = vehicle.id
  try {
    await vehiclesApi.submit(auth.organizationId, vehicle.id)
    await loadVehicles()
  } catch (e) {
    logger.error('Failed to submit vehicle for verification', e)
    toast({ title: 'Не удалось отправить транспорт на подтверждение', variant: 'destructive' })
  } finally {
    submittingId.value = null
  }
}

function canSubmit(vehicle: Vehicle) {
  return vehicle.status === 'unconfirmed' || vehicle.status === 'rejected'
}

function formatCapacity(val?: number) {
  if (!val) return null
  return `${val.toLocaleString('ru-RU')} т`
}
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <LoadingSpinner v-if="isLoading" text="Загрузка автопарка..." />

    <div
      v-else-if="loadError"
      class="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive"
    >
      {{ loadError }}
    </div>

    <div v-else class="rounded-lg border bg-card overflow-hidden">
      <!-- Хедер -->
      <div class="flex items-center justify-between px-5 py-4 border-b">
        <div class="flex items-center gap-2.5">
          <h2 class="text-sm font-semibold text-foreground">Автопарк</h2>
          <span
            v-if="vehicles.length > 0"
            class="text-xs font-semibold px-1.5 py-0.5 rounded-full bg-primary/10 text-primary"
          >{{ vehicles.length }}</span>
        </div>
        <Button size="sm" variant="outline" @click="router.push({ name: 'vehicle-add' })">
          <Plus class="h-4 w-4 mr-1.5" />
          Добавить
        </Button>
      </div>

      <!-- Пустое состояние -->
      <div v-if="vehicles.length === 0" class="px-5 py-10 text-center text-sm text-muted-foreground">
        Нет транспортных средств
      </div>

      <!-- Строки -->
      <div v-else>
        <div
          v-for="(vehicle, index) in vehicles"
          :key="vehicle.id"
          class="flex items-center gap-4 px-5 py-4 transition-colors"
          :class="[
            index < vehicles.length - 1 ? 'border-b' : '',
            vehicle.status !== 'archived' ? 'cursor-pointer hover:bg-muted/40' : 'opacity-60',
          ]"
          @click="vehicle.status !== 'archived' && router.push({ name: 'vehicle-edit', params: { id: vehicle.id } })"
        >
          <Truck class="h-6 w-6 text-muted-foreground shrink-0" />

          <div class="flex-1 min-w-0">
            <!-- Строка 1: гос. номер + статус -->
            <div class="flex items-center gap-2.5 flex-wrap">
              <span class="font-mono font-bold text-base">{{ vehicle.registration_number }}</span>
              <span v-if="vehicle.brand || vehicle.model" class="text-sm text-muted-foreground">
                {{ [vehicle.brand, vehicle.model].filter(Boolean).join(' ') }}
              </span>
              <StatusBadge :status="vehicle.status" :status-map="vehicleStatusMap" class="shrink-0" />
              <button
                v-if="inviteCountByVehicle.get(vehicle.id)"
                class="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
                @click.stop="router.push('/my-offers?tab=invitations')"
              >
                <Mail class="h-3 w-3" />
                {{ inviteCountByVehicle.get(vehicle.id) }}
              </button>
            </div>

            <!-- Строка 2: тип, кузов, характеристики -->
            <div class="flex items-center gap-1.5 text-sm text-muted-foreground mt-1 flex-wrap">
              <span>{{ vehicleTypeLabels[vehicle.vehicle_type] }}</span>
              <span>·</span>
              <span>{{ vehicleSubTypeLabels[vehicle.vehicle_subtype] }}</span>
              <template v-if="vehicle.capacity">
                <span>·</span>
                <span>{{ formatCapacity(vehicle.capacity) }}</span>
              </template>
              <template v-if="vehicle.volume">
                <span>·</span>
                <span>{{ vehicle.volume }} м³</span>
              </template>
              <template v-if="vehicle.requires_adr">
                <span>·</span>
                <span class="text-orange-600 dark:text-orange-400 font-medium">ADR</span>
              </template>
              <template v-if="vehicle.thermograph">
                <span>·</span>
                <span class="text-blue-600 dark:text-blue-400 font-medium">Термограф</span>
              </template>
            </div>

            <!-- Строка 3: причина отклонения -->
            <div
              v-if="vehicle.status === 'rejected' && vehicle.rejection_reason"
              class="flex items-center gap-1 text-sm text-destructive mt-1"
            >
              <AlertCircle class="h-3.5 w-3.5 shrink-0" />
              <span>{{ vehicle.rejection_reason }}</span>
            </div>
          </div>

          <Button
            v-if="canSubmit(vehicle)"
            size="sm"
            variant="outline"
            class="shrink-0"
            :disabled="submittingId === vehicle.id"
            @click.stop="submitVehicle(vehicle)"
          >
            <BadgeCheck class="h-4 w-4 mr-1.5" />
            {{ submittingId === vehicle.id ? 'Отправка...' : 'Подтвердить' }}
          </Button>

          <Button
            variant="ghost"
            size="icon"
            class="h-9 w-9 text-muted-foreground hover:text-destructive shrink-0"
            :disabled="vehicle.status === 'archived'"
            @click.stop="deletingVehicle = vehicle"
          >
            <Trash2 class="h-5 w-5" />
          </Button>

          <ChevronRight
            class="h-5 w-5 text-muted-foreground shrink-0"
            :class="vehicle.status === 'archived' ? 'invisible' : ''"
          />
        </div>
      </div>
    </div>

    <Dialog :open="!!deletingVehicle" @update:open="v => { if (!v) deletingVehicle = null }">
      <DialogContent>
        <DialogHeader class="space-y-4">
          <DialogTitle>Убрать из автопарка?</DialogTitle>
          <DialogDescription>
            <template v-if="deletingVehicle">
              {{ vehicleTypeLabels[deletingVehicle.vehicle_type] }} · {{ deletingVehicle.registration_number }}
            </template>
            будет удалён из вашего автопарка.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <Button variant="outline" @click="deletingVehicle = null">Отмена</Button>
          <Button variant="destructive" :disabled="isDeleting" @click="confirmDelete">
            {{ isDeleting ? 'Удаление...' : 'Убрать' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
