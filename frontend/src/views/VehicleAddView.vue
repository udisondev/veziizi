<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { vehiclesApi } from '@/api/vehicles'
import type { CreateVehicleRequest } from '@/types/vehicle'
import {
  vehicleTypeOptions,
  getVehicleSubTypeOptions, allVehicleSubTypeOptions,
  getVehicleTypeForSubType, isSubTypeCompatible,
  loadingTypeOptions,
  type VehicleType, type VehicleSubType, type LoadingType,
} from '@/types/freightRequest'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { SelectField } from '@/components/ui/select-field'
import { DetailPageHeader } from '@/components/shared'

interface VehicleFormState {
  vehicle_type: VehicleType | ''
  vehicle_subtype: VehicleSubType | ''
  registration_number: string
  brand: string
  model: string
  capacity: number | undefined
  volume: number | undefined
  length: number | undefined
  width: number | undefined
  height: number | undefined
  loading_types: LoadingType[]
  requires_adr: boolean
  thermograph: boolean
  has_temperature: boolean
  temp_min: number | undefined
  temp_max: number | undefined
}

const router = useRouter()
const auth = useAuthStore()

const isSaving = ref(false)
const formError = ref<string | null>(null)

const form = ref<VehicleFormState>({
  vehicle_type: '',
  vehicle_subtype: '',
  registration_number: '',
  brand: '',
  model: '',
  capacity: undefined,
  volume: undefined,
  length: undefined,
  width: undefined,
  height: undefined,
  loading_types: [],
  requires_adr: false,
  thermograph: false,
  has_temperature: false,
  temp_min: undefined,
  temp_max: undefined,
})

const temperatureSubTypes: VehicleSubType[] = ['insulated', 'refrigerator']

const subtypeOptions = computed(() =>
  form.value.vehicle_type
    ? getVehicleSubTypeOptions(form.value.vehicle_type)
    : allVehicleSubTypeOptions
)

const isTemperatureSubType = computed(() =>
  !!form.value.vehicle_subtype && temperatureSubTypes.includes(form.value.vehicle_subtype as VehicleSubType)
)

watch(isTemperatureSubType, (isTemp) => {
  if (!isTemp) {
    form.value.has_temperature = false
    form.value.thermograph = false
  }
})

function selectVehicleType(value: string | null | undefined) {
  if (!value) {
    form.value.vehicle_type = '' as VehicleType
    form.value.vehicle_subtype = '' as VehicleSubType
    return
  }
  const type = value as VehicleType
  const cur = form.value.vehicle_subtype as VehicleSubType | ''
  if (cur && !isSubTypeCompatible(type, cur)) {
    form.value.vehicle_subtype = '' as VehicleSubType
  }
  form.value.vehicle_type = type
}

function selectVehicleSubType(subtype: VehicleSubType) {
  form.value.vehicle_type = form.value.vehicle_type || getVehicleTypeForSubType(subtype)
  form.value.vehicle_subtype = subtype
}

function toggleLoadingType(type: LoadingType) {
  const cur = form.value.loading_types
  form.value.loading_types = cur.includes(type) ? cur.filter(t => t !== type) : [...cur, type]
}

function formToRequest(f: VehicleFormState): CreateVehicleRequest {
  return {
    vehicle_type: f.vehicle_type as VehicleType,
    vehicle_subtype: f.vehicle_subtype as VehicleSubType,
    registration_number: f.registration_number.trim().toUpperCase(),
    brand: f.brand.trim() || undefined,
    model: f.model.trim() || undefined,
    capacity: f.capacity,
    volume: f.volume,
    length: f.length,
    width: f.width,
    height: f.height,
    loading_types: f.loading_types,
    requires_adr: f.requires_adr || undefined,
    thermograph: f.thermograph || undefined,
    temperature: f.has_temperature && f.temp_min !== undefined && f.temp_max !== undefined
      ? { min: f.temp_min, max: f.temp_max }
      : undefined,
  }
}

async function handleSubmit() {
  if (!auth.organizationId) return
  if (!form.value.vehicle_type || !form.value.vehicle_subtype || !form.value.registration_number.trim()) {
    formError.value = 'Заполните обязательные поля'
    return
  }
  isSaving.value = true
  formError.value = null
  try {
    await vehiclesApi.create(auth.organizationId, formToRequest(form.value))
    router.push({ name: 'fleet' })
  } catch {
    formError.value = 'Не удалось сохранить. Попробуйте ещё раз'
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <div>
    <DetailPageHeader :back-to="{ name: 'fleet' }" back-label="К автопарку" />

    <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
      <div class="flex justify-center">
        <div class="w-full max-w-lg bg-card border border-border rounded-xl shadow-sm p-6 space-y-5">

          <form class="space-y-5" @submit.prevent="handleSubmit">
            <div class="space-y-1.5">
              <Label>Тип транспорта <span class="text-destructive">*</span></Label>
              <SelectField
                :model-value="form.vehicle_type"
                :options="vehicleTypeOptions"
                sheet-label="Тип транспорта"
                clearable
                clear-label="Не выбран"
                @update:model-value="selectVehicleType"
              />
            </div>

            <div class="space-y-1.5">
              <Label>Тип кузова <span class="text-destructive">*</span></Label>
              <SelectField
                :model-value="form.vehicle_subtype"
                :options="subtypeOptions"
                sheet-label="Тип кузова"
                @update:model-value="selectVehicleSubType($event as VehicleSubType)"
              />
            </div>

            <div class="space-y-1.5">
              <Label for="reg">Гос. номер <span class="text-destructive">*</span></Label>
              <Input id="reg" v-model="form.registration_number" placeholder="А123БВ 77" class="font-mono uppercase" />
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-1.5">
                <Label for="brand">Марка</Label>
                <Input id="brand" v-model="form.brand" placeholder="Volvo" />
              </div>
              <div class="space-y-1.5">
                <Label for="model">Модель</Label>
                <Input id="model" v-model="form.model" placeholder="FH16" />
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-1.5">
                <Label for="capacity">Грузоподъёмность, т</Label>
                <Input id="capacity" v-model.number="form.capacity" type="number" placeholder="20" min="0" step="0.1" />
              </div>
              <div class="space-y-1.5">
                <Label for="volume">Объём кузова, м³</Label>
                <Input id="volume" v-model.number="form.volume" type="number" placeholder="82" min="0" />
              </div>
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
                    form.loading_types.includes(option.value as LoadingType)
                      ? 'bg-primary text-primary-foreground border-primary'
                      : 'border-border text-muted-foreground hover:border-foreground hover:text-foreground',
                  ]"
                  @click="toggleLoadingType(option.value as LoadingType)"
                >{{ option.label }}</button>
              </div>
            </div>

            <template v-if="isTemperatureSubType">
              <div class="space-y-3">
                <label class="flex items-center gap-2.5 cursor-pointer">
                  <input type="checkbox" v-model="form.has_temperature" class="h-4 w-4 accent-primary rounded" />
                  <span class="text-sm font-medium">Температурный режим</span>
                </label>
                <div v-if="form.has_temperature" class="grid grid-cols-2 gap-4 pl-6">
                  <div class="space-y-1.5">
                    <Label for="tmin">Мин., °C</Label>
                    <Input id="tmin" v-model.number="form.temp_min" type="number" placeholder="-20" />
                  </div>
                  <div class="space-y-1.5">
                    <Label for="tmax">Макс., °C</Label>
                    <Input id="tmax" v-model.number="form.temp_max" type="number" placeholder="+4" />
                  </div>
                </div>
              </div>
              <label class="flex items-center gap-2.5 cursor-pointer">
                <input type="checkbox" v-model="form.thermograph" class="h-4 w-4 accent-primary rounded" />
                <span class="text-sm">Термописец</span>
              </label>
            </template>

            <div class="space-y-2.5">
              <label class="flex items-center gap-2.5 cursor-pointer">
                <input type="checkbox" v-model="form.requires_adr" class="h-4 w-4 accent-primary rounded" />
                <span class="text-sm">Опасный груз (ADR)</span>
              </label>
            </div>

            <p v-if="formError" class="text-sm text-destructive">{{ formError }}</p>

            <Separator />

            <div class="flex gap-3">
              <Button type="button" variant="outline" class="h-12 text-base md:h-10 md:text-sm" @click="router.push({ name: 'fleet' })">
                Отмена
              </Button>
              <Button type="submit" class="h-12 text-base md:h-10 md:text-sm" :disabled="isSaving">
                {{ isSaving ? 'Сохранение...' : 'Добавить' }}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>
