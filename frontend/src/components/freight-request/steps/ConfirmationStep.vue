<script setup lang="ts">
import { computed } from 'vue'
import type { CreateFreightRequestRequest, RoutePoint } from '@/types/freightRequest'
import {
  vehicleTypeLabels,
  vehicleSubTypeLabels,
  loadingTypeLabels,
  currencyLabels,
  vatTypeLabels,
  paymentMethodLabels,
  paymentTermsLabels,
} from '@/types/freightRequest'
import LeafletMap from '../shared/LeafletMap.vue'

interface Props {
  requestData: CreateFreightRequestRequest
  comment: string
}

interface Emits {
  (e: 'update:comment', value: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function getPointTypeLabel(point: RoutePoint): string {
  if (point.is_loading && point.is_unloading) {
    return 'Погрузка/Разгрузка'
  }
  if (point.is_loading) {
    return 'Погрузка'
  }
  if (point.is_unloading) {
    return 'Разгрузка'
  }
  return 'Точка'
}

function getPointBorderClass(point: RoutePoint): string {
  if (point.is_loading && point.is_unloading) {
    return 'border-l-purple-500'
  }
  if (point.is_loading) {
    return 'border-l-blue-500'
  }
  if (point.is_unloading) {
    return 'border-l-green-500'
  }
  return 'border-l-gray-300'
}

function getPointBadgeClass(point: RoutePoint): string {
  if (point.is_loading && point.is_unloading) {
    return 'bg-purple-100 text-purple-700'
  }
  if (point.is_loading) {
    return 'bg-blue-100 text-blue-700'
  }
  if (point.is_unloading) {
    return 'bg-green-100 text-green-700'
  }
  return 'bg-gray-100 text-gray-700'
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—'
  // Если это просто дата (YYYY-MM-DD), форматируем её
  if (dateStr.length === 10) {
    const [year, month, day] = dateStr.split('-')
    return `${day}.${month}.${year}`
  }
  // Если ISO дата, парсим
  const date = new Date(dateStr)
  return date.toLocaleDateString('ru-RU')
}

function formatTime(timeStr: string | undefined): string {
  if (!timeStr) return ''
  return timeStr
}

function formatPrice(amount: number, currency: string): string {
  const value = amount / 100
  return `${value.toLocaleString('ru-RU')} ${currencyLabels[currency as keyof typeof currencyLabels]}`
}

const hasCoordinates = computed(() =>
  props.requestData.route.points.some((p) => p.coordinates)
)

const hasPrice = computed(() =>
  props.requestData.payment.price && props.requestData.payment.price.amount > 0
)

function handleCommentInput(event: Event) {
  emit('update:comment', (event.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <div class="space-y-6">
    <div class="bg-green-50 border border-green-200 rounded-lg p-4 text-green-800">
      <p class="font-medium">Проверьте данные перед публикацией</p>
      <p class="text-sm mt-1">После публикации заявка станет доступна перевозчикам.</p>
    </div>

    <!-- Route -->
    <div class="bg-gray-50 rounded-lg p-4">
      <h4 class="font-medium text-gray-900 mb-3 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-gray-600" viewBox="0 0 20 20" fill="currentColor">
          <path d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
        </svg>
        Маршрут
      </h4>

      <div class="space-y-3">
        <div
          v-for="(point, index) in requestData.route.points"
          :key="index"
          :class="[
            'flex items-start gap-3 p-3 bg-white rounded-md border-l-4',
            getPointBorderClass(point),
          ]"
        >
          <span
            :class="[
              'px-2 py-0.5 text-xs font-medium rounded',
              getPointBadgeClass(point),
            ]"
          >
            {{ getPointTypeLabel(point) }}
          </span>
          <div class="flex-1">
            <div class="font-medium text-gray-900">{{ point.address }}</div>
            <div class="text-sm text-gray-500">
              {{ formatDate(point.date_from) }}
              <template v-if="point.date_to"> — {{ formatDate(point.date_to) }}</template>
              <template v-if="point.time_from">
                , {{ formatTime(point.time_from) }}
                <template v-if="point.time_to"> — {{ formatTime(point.time_to) }}</template>
              </template>
            </div>
            <div v-if="point.contact_name || point.contact_phone" class="text-sm text-gray-500">
              Контакт: {{ point.contact_name }}
              <template v-if="point.contact_phone">, {{ point.contact_phone }}</template>
            </div>
            <div v-if="point.comment" class="text-sm text-gray-400 italic">
              {{ point.comment }}
            </div>
          </div>
        </div>
      </div>

      <LeafletMap
        v-if="hasCoordinates"
        :points="requestData.route.points"
        height="200px"
        class="mt-3"
      />
    </div>

    <!-- Cargo -->
    <div class="bg-gray-50 rounded-lg p-4">
      <h4 class="font-medium text-gray-900 mb-3 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-gray-600" viewBox="0 0 20 20" fill="currentColor">
          <path d="M4 3a2 2 0 100 4h12a2 2 0 100-4H4z" />
          <path fill-rule="evenodd" d="M3 8h14v7a2 2 0 01-2 2H5a2 2 0 01-2-2V8zm5 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" clip-rule="evenodd" />
        </svg>
        Груз
      </h4>

      <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
        <dt class="text-gray-500">Описание:</dt>
        <dd class="text-gray-900">{{ requestData.cargo.description }}</dd>

        <dt class="text-gray-500">Вес:</dt>
        <dd class="text-gray-900">{{ requestData.cargo.weight.toLocaleString('ru-RU') }} кг</dd>

        <template v-if="requestData.cargo.volume">
          <dt class="text-gray-500">Объём:</dt>
          <dd class="text-gray-900">{{ requestData.cargo.volume }} м³</dd>
        </template>

        <template v-if="requestData.cargo.dimensions">
          <dt class="text-gray-500">Габариты:</dt>
          <dd class="text-gray-900">
            {{ requestData.cargo.dimensions.length }} × {{ requestData.cargo.dimensions.width }} × {{ requestData.cargo.dimensions.height }} м
          </dd>
        </template>

        <template v-if="requestData.cargo.quantity">
          <dt class="text-gray-500">Количество:</dt>
          <dd class="text-gray-900">{{ requestData.cargo.quantity }} мест</dd>
        </template>
      </dl>
    </div>

    <!-- Vehicle -->
    <div class="bg-gray-50 rounded-lg p-4">
      <h4 class="font-medium text-gray-900 mb-3 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-gray-600" viewBox="0 0 20 20" fill="currentColor">
          <path d="M8 16.5a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0zM15 16.5a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0z" />
          <path d="M3 4a1 1 0 00-1 1v10a1 1 0 001 1h1.05a2.5 2.5 0 014.9 0H10a1 1 0 001-1V5a1 1 0 00-1-1H3zM14 7a1 1 0 00-1 1v6.05A2.5 2.5 0 0115.95 16H17a1 1 0 001-1v-5a1 1 0 00-.293-.707l-2-2A1 1 0 0015 7h-1z" />
        </svg>
        Требования к транспорту
      </h4>

      <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
        <dt class="text-gray-500">Тип транспорта:</dt>
        <dd class="text-gray-900">
          {{ vehicleTypeLabels[requestData.vehicle_requirements.vehicle_type] }}
        </dd>

        <dt class="text-gray-500">Подтип:</dt>
        <dd class="text-gray-900">
          {{ vehicleSubTypeLabels[requestData.vehicle_requirements.vehicle_subtype] }}
        </dd>

        <template v-if="requestData.vehicle_requirements.loading_types?.length">
          <dt class="text-gray-500">Погрузка:</dt>
          <dd class="text-gray-900">
            {{ requestData.vehicle_requirements.loading_types.map(t => loadingTypeLabels[t]).join(', ') }}
          </dd>
        </template>

        <template v-if="requestData.vehicle_requirements.capacity">
          <dt class="text-gray-500">Грузоподъёмность:</dt>
          <dd class="text-gray-900">{{ requestData.vehicle_requirements.capacity.toLocaleString('ru-RU') }} кг</dd>
        </template>

        <template v-if="requestData.vehicle_requirements.volume">
          <dt class="text-gray-500">Объём кузова:</dt>
          <dd class="text-gray-900">{{ requestData.vehicle_requirements.volume }} м³</dd>
        </template>

        <template v-if="requestData.vehicle_requirements.temperature">
          <dt class="text-gray-500">Температура:</dt>
          <dd class="text-gray-900">
            {{ requestData.vehicle_requirements.temperature.min }}°C — {{ requestData.vehicle_requirements.temperature.max }}°C
          </dd>
        </template>

        <template v-if="requestData.vehicle_requirements.requires_adr">
          <dt class="text-gray-500">ADR:</dt>
          <dd class="text-gray-900 text-orange-600 font-medium">Требуется</dd>
        </template>
      </dl>
    </div>

    <!-- Payment -->
    <div class="bg-gray-50 rounded-lg p-4">
      <h4 class="font-medium text-gray-900 mb-3 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-gray-600" viewBox="0 0 20 20" fill="currentColor">
          <path d="M4 4a2 2 0 00-2 2v4a2 2 0 002 2V6h10a2 2 0 00-2-2H4zm2 6a2 2 0 012-2h8a2 2 0 012 2v4a2 2 0 01-2 2H8a2 2 0 01-2-2v-4zm6 4a2 2 0 100-4 2 2 0 000 4z" />
        </svg>
        Оплата
      </h4>

      <!-- Если цена указана -->
      <template v-if="hasPrice">
        <div class="text-2xl font-bold text-gray-900 mb-2">
          {{ formatPrice(requestData.payment.price!.amount, requestData.payment.price!.currency) }}
        </div>

        <dl class="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
          <dt class="text-gray-500">НДС:</dt>
          <dd class="text-gray-900">{{ vatTypeLabels[requestData.payment.vat_type] }}</dd>

          <dt class="text-gray-500">Способ:</dt>
          <dd class="text-gray-900">{{ paymentMethodLabels[requestData.payment.method] }}</dd>

          <dt class="text-gray-500">Условия:</dt>
          <dd class="text-gray-900">
            {{ paymentTermsLabels[requestData.payment.terms] }}
            <template v-if="requestData.payment.deferred_days">
              ({{ requestData.payment.deferred_days }} дн.)
            </template>
          </dd>
        </dl>
      </template>

      <!-- Если цена не указана -->
      <template v-else>
        <p class="text-gray-500">Цена не указана — перевозчики предложат свою</p>
      </template>
    </div>

    <!-- Comment -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Комментарий к заявке
      </label>
      <textarea
        :value="comment"
        placeholder="Дополнительная информация для перевозчиков"
        rows="3"
        class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        @input="handleCommentInput"
      />
    </div>
  </div>
</template>
