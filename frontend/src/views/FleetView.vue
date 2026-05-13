<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { vehiclesApi } from '@/api/vehicles'
import type { Vehicle, CreateVehicleRequest, UpdateVehicleRequest } from '@/types/vehicle'
import {
  vehicleTypeLabels, vehicleTypeOptions,
  getVehicleSubTypeOptions, vehicleSubTypeLabels,
  loadingTypeOptions, loadingTypeLabels,
  type VehicleType, type VehicleSubType, type LoadingType,
} from '@/types/freightRequest'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
import { SelectField } from '@/components/ui/select-field'
import { TabsSlider, type TabSliderItem } from '@/components/ui/tabs'
import {
  Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter,
} from '@/components/ui/sheet'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter,
} from '@/components/ui/dialog'
import { LoadingSpinner, EmptyState } from '@/components/shared'
import { Truck, Plus, Pencil, Trash2, Weight, Box, List } from 'lucide-vue-next'

type TabValue = 'list' | 'add'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

// --- Табы ---
const activeTab = ref<TabValue>((route.query.tab as TabValue) || 'list')

const tabs: TabSliderItem[] = [
  { value: 'list', label: 'Список', icon: List },
  { value: 'add', label: 'Добавить', icon: Plus },
]

watch(activeTab, (tab) => {
  router.replace({ query: { ...route.query, tab } })
  if (tab === 'add') resetForm()
})

watch(
  () => route.query.tab as TabValue | undefined,
  (tab) => { if (tab && tab !== activeTab.value) activeTab.value = tab }
)

// --- Список ---
const vehicles = ref<Vehicle[]>([])
const isLoading = ref(false)
const loadError = ref<string | null>(null)

async function loadVehicles() {
  if (!auth.organizationId) return
  isLoading.value = true
  loadError.value = null
  try {
    vehicles.value = await vehiclesApi.list(auth.organizationId)
  } catch {
    loadError.value = 'Не удалось загрузить автопарк'
  } finally {
    isLoading.value = false
  }
}

loadVehicles()

// --- Форма добавления ---
const addForm = ref<CreateVehicleRequest>(emptyForm())
const addError = ref<string | null>(null)
const isSaving = ref(false)

function emptyForm(): CreateVehicleRequest {
  return {
    vehicle_type: '' as VehicleType,
    vehicle_subtype: '' as VehicleSubType,
    registration_number: '',
    year: undefined,
    capacity: undefined,
    volume: undefined,
    loading_types: [],
  }
}

function resetForm() {
  addForm.value = emptyForm()
  addError.value = null
}

const addSubtypeOptions = computed(() =>
  addForm.value.vehicle_type ? getVehicleSubTypeOptions(addForm.value.vehicle_type) : []
)

watch(() => addForm.value.vehicle_type, () => {
  addForm.value.vehicle_subtype = '' as VehicleSubType
})

function toggleAddLoadingType(type: LoadingType) {
  const cur = addForm.value.loading_types ?? []
  addForm.value.loading_types = cur.includes(type) ? cur.filter(t => t !== type) : [...cur, type]
}

async function saveAdd() {
  if (!auth.organizationId) return
  if (!addForm.value.vehicle_type || !addForm.value.vehicle_subtype || !addForm.value.registration_number.trim()) {
    addError.value = 'Заполните обязательные поля'
    return
  }
  isSaving.value = true
  addError.value = null
  try {
    await vehiclesApi.create(auth.organizationId, addForm.value)
    activeTab.value = 'list'
    await loadVehicles()
  } catch {
    addError.value = 'Не удалось сохранить. Попробуйте ещё раз'
  } finally {
    isSaving.value = false
  }
}

// --- Редактирование (Sheet в списке) ---
const editingVehicle = ref<Vehicle | null>(null)
const editForm = ref<UpdateVehicleRequest>({})
const isEditSheetOpen = ref(false)
const isEditSaving = ref(false)
const editError = ref<string | null>(null)

const editSubtypeOptions = computed(() =>
  (editForm.value.vehicle_type as VehicleType | undefined)
    ? getVehicleSubTypeOptions(editForm.value.vehicle_type as VehicleType)
    : []
)

watch(() => editForm.value.vehicle_type, () => {
  editForm.value.vehicle_subtype = '' as VehicleSubType
})

function openEdit(vehicle: Vehicle) {
  editingVehicle.value = vehicle
  editForm.value = {
    vehicle_type: vehicle.vehicle_type,
    vehicle_subtype: vehicle.vehicle_subtype,
    registration_number: vehicle.registration_number,
    year: vehicle.year,
    capacity: vehicle.capacity,
    volume: vehicle.volume,
    loading_types: vehicle.loading_types ?? [],
  }
  editError.value = null
  isEditSheetOpen.value = true
}

function toggleEditLoadingType(type: LoadingType) {
  const cur = editForm.value.loading_types ?? []
  editForm.value.loading_types = cur.includes(type) ? cur.filter(t => t !== type) : [...cur, type]
}

async function saveEdit() {
  if (!auth.organizationId || !editingVehicle.value) return
  if (!editForm.value.vehicle_type || !editForm.value.vehicle_subtype || !editForm.value.registration_number?.trim()) {
    editError.value = 'Заполните обязательные поля'
    return
  }
  isEditSaving.value = true
  editError.value = null
  try {
    await vehiclesApi.update(auth.organizationId, editingVehicle.value.id, editForm.value)
    isEditSheetOpen.value = false
    await loadVehicles()
  } catch {
    editError.value = 'Не удалось сохранить. Попробуйте ещё раз'
  } finally {
    isEditSaving.value = false
  }
}

// --- Удаление ---
const deletingVehicle = ref<Vehicle | null>(null)
const isDeleting = ref(false)

async function confirmDelete() {
  if (!deletingVehicle.value || !auth.organizationId) return
  isDeleting.value = true
  try {
    await vehiclesApi.remove(auth.organizationId, deletingVehicle.value.id)
    deletingVehicle.value = null
    await loadVehicles()
  } finally {
    isDeleting.value = false
  }
}

// --- Helpers ---
function formatCapacity(kg?: number) {
  if (!kg) return null
  return kg >= 1000 ? `${(kg / 1000).toLocaleString('ru-RU')} т` : `${kg} кг`
}
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <!-- Табы -->
    <div class="mb-6">
      <TabsSlider v-model="activeTab" :items="tabs" />
    </div>

    <!-- === Список === -->
    <template v-if="activeTab === 'list'">
      <LoadingSpinner v-if="isLoading" text="Загрузка автопарка..." />

      <div
        v-else-if="loadError"
        class="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive"
      >
        {{ loadError }}
      </div>

      <EmptyState
        v-else-if="vehicles.length === 0"
        :icon="Truck"
        title="Автопарк пуст"
        description="Добавьте первый автомобиль вашей организации"
      >
        <template #action>
          <Button @click="activeTab = 'add'">
            <Plus class="mr-2 h-4 w-4" />
            Добавить автомобиль
          </Button>
        </template>
      </EmptyState>

      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <Card v-for="vehicle in vehicles" :key="vehicle.id">
          <CardContent class="p-4">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <div class="font-semibold text-foreground">
                  {{ vehicleTypeLabels[vehicle.vehicle_type] }}
                </div>
                <div class="text-sm text-muted-foreground">
                  {{ vehicleSubTypeLabels[vehicle.vehicle_subtype] }}
                </div>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <Button variant="ghost" size="icon" class="h-8 w-8" @click="openEdit(vehicle)">
                  <Pencil class="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-destructive hover:text-destructive"
                  @click="deletingVehicle = vehicle"
                >
                  <Trash2 class="h-4 w-4" />
                </Button>
              </div>
            </div>

            <div class="mt-3 space-y-1.5">
              <div class="flex items-center gap-1.5 text-sm">
                <Truck class="h-4 w-4 text-muted-foreground shrink-0" />
                <span class="font-mono font-medium">{{ vehicle.registration_number }}</span>
                <span v-if="vehicle.year" class="text-muted-foreground">· {{ vehicle.year }}</span>
              </div>
              <div v-if="vehicle.capacity" class="flex items-center gap-1.5 text-sm text-muted-foreground">
                <Weight class="h-4 w-4 shrink-0" />
                {{ formatCapacity(vehicle.capacity) }}
              </div>
              <div v-if="vehicle.volume" class="flex items-center gap-1.5 text-sm text-muted-foreground">
                <Box class="h-4 w-4 shrink-0" />
                {{ vehicle.volume }} м³
              </div>
              <div v-if="vehicle.loading_types?.length" class="text-xs text-muted-foreground">
                {{ vehicle.loading_types.map(t => loadingTypeLabels[t]).join(', ') }}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>

    <!-- === Добавить === -->
    <template v-if="activeTab === 'add'">
      <div class="flex justify-center">
      <div class="w-full max-w-lg bg-card border border-border rounded-xl shadow-sm p-6 space-y-5">
        <div class="space-y-1.5">
          <Label>Тип транспорта <span class="text-destructive">*</span></Label>
          <SelectField
            v-model="addForm.vehicle_type"
            :options="vehicleTypeOptions"
            sheet-label="Тип транспорта"
          />
        </div>

        <div class="space-y-1.5">
          <Label>Тип кузова <span class="text-destructive">*</span></Label>
          <SelectField
            v-model="addForm.vehicle_subtype"
            :options="addSubtypeOptions"
            :disabled="!addForm.vehicle_type"
            sheet-label="Тип кузова"
          />
        </div>

        <div class="space-y-1.5">
          <Label for="reg-number">Гос. номер <span class="text-destructive">*</span></Label>
          <Input
            id="reg-number"
            v-model="addForm.registration_number"
            placeholder="А123БВ 77"
            class="font-mono uppercase"
          />
        </div>

        <div class="space-y-1.5">
          <Label for="year">Год выпуска</Label>
          <Input
            id="year"
            v-model.number="addForm.year"
            type="number"
            placeholder="2020"
            min="1980"
            :max="new Date().getFullYear()"
          />
        </div>

        <div class="space-y-1.5">
          <Label for="capacity">Грузоподъёмность, кг</Label>
          <Input
            id="capacity"
            v-model.number="addForm.capacity"
            type="number"
            placeholder="20000"
            min="0"
          />
        </div>

        <div class="space-y-1.5">
          <Label for="volume">Объём кузова, м³</Label>
          <Input
            id="volume"
            v-model.number="addForm.volume"
            type="number"
            placeholder="82"
            min="0"
          />
        </div>

        <div class="space-y-2">
          <Label>Типы погрузки</Label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in loadingTypeOptions"
              :key="option.value"
              type="button"
              :class="[
                'px-3 py-1.5 rounded-full text-sm border transition-colors',
                addForm.loading_types?.includes(option.value as LoadingType)
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'border-border text-muted-foreground hover:border-foreground hover:text-foreground',
              ]"
              @click="toggleAddLoadingType(option.value as LoadingType)"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <p v-if="addError" class="text-sm text-destructive">{{ addError }}</p>

        <div class="flex items-center gap-3 pt-2">
          <Button variant="outline" @click="activeTab = 'list'">Отмена</Button>
          <Button :disabled="isSaving" @click="saveAdd">
            {{ isSaving ? 'Сохранение...' : 'Добавить' }}
          </Button>
        </div>
      </div>
      </div>
    </template>

    <!-- Sheet редактирования -->
    <Sheet v-model:open="isEditSheetOpen">
      <SheetContent class="w-full sm:max-w-md overflow-y-auto">
        <SheetHeader class="mb-6">
          <SheetTitle>Редактировать автомобиль</SheetTitle>
        </SheetHeader>

        <div class="space-y-5">
          <div class="space-y-1.5">
            <Label>Тип транспорта <span class="text-destructive">*</span></Label>
            <SelectField
              v-model="editForm.vehicle_type"
              :options="vehicleTypeOptions"
              sheet-label="Тип транспорта"
            />
          </div>

          <div class="space-y-1.5">
            <Label>Тип кузова <span class="text-destructive">*</span></Label>
            <SelectField
              v-model="editForm.vehicle_subtype"
              :options="editSubtypeOptions"
              :disabled="!editForm.vehicle_type"
              sheet-label="Тип кузова"
            />
          </div>

          <div class="space-y-1.5">
            <Label>Гос. номер <span class="text-destructive">*</span></Label>
            <Input
              v-model="editForm.registration_number"
              placeholder="А123БВ 77"
              class="font-mono uppercase"
            />
          </div>

          <div class="space-y-1.5">
            <Label>Год выпуска</Label>
            <Input
              v-model.number="editForm.year"
              type="number"
              placeholder="2020"
              min="1980"
              :max="new Date().getFullYear()"
            />
          </div>

          <div class="space-y-1.5">
            <Label>Грузоподъёмность, кг</Label>
            <Input
              v-model.number="editForm.capacity"
              type="number"
              placeholder="20000"
              min="0"
            />
          </div>

          <div class="space-y-1.5">
            <Label>Объём кузова, м³</Label>
            <Input
              v-model.number="editForm.volume"
              type="number"
              placeholder="82"
              min="0"
            />
          </div>

          <div class="space-y-2">
            <Label>Типы погрузки</Label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="option in loadingTypeOptions"
                :key="option.value"
                type="button"
                :class="[
                  'px-3 py-1.5 rounded-full text-sm border transition-colors',
                  editForm.loading_types?.includes(option.value as LoadingType)
                    ? 'bg-primary text-primary-foreground border-primary'
                    : 'border-border text-muted-foreground hover:border-foreground hover:text-foreground',
                ]"
                @click="toggleEditLoadingType(option.value as LoadingType)"
              >
                {{ option.label }}
              </button>
            </div>
          </div>

          <p v-if="editError" class="text-sm text-destructive">{{ editError }}</p>
        </div>

        <SheetFooter class="mt-8 gap-3">
          <Button variant="outline" class="flex-1" @click="isEditSheetOpen = false">Отмена</Button>
          <Button class="flex-1" :disabled="isEditSaving" @click="saveEdit">
            {{ isEditSaving ? 'Сохранение...' : 'Сохранить' }}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>

    <!-- Диалог удаления -->
    <Dialog :open="!!deletingVehicle" @update:open="v => { if (!v) deletingVehicle = null }">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Удалить автомобиль?</DialogTitle>
          <DialogDescription>
            <template v-if="deletingVehicle">
              {{ vehicleTypeLabels[deletingVehicle.vehicle_type] }} · {{ deletingVehicle.registration_number }}
            </template>
            будет удалён из автопарка.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-3">
          <Button variant="outline" @click="deletingVehicle = null">Отмена</Button>
          <Button variant="destructive" :disabled="isDeleting" @click="confirmDelete">
            {{ isDeleting ? 'Удаление...' : 'Удалить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
