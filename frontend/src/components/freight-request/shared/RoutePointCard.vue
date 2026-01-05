<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { RoutePoint, Coordinates } from '@/types/freightRequest'
import CountryCitySelect from './CountryCitySelect.vue'
import { useTutorialEvent } from '@/composables/useTutorialEvent'

const { emit: emitTutorial } = useTutorialEvent()

interface Props {
  point: RoutePoint
  index: number
  totalPoints: number
  errors?: Record<string, string | null>
  canRemove?: boolean
  canMove?: boolean
}

interface Emits {
  (e: 'update', updates: Partial<RoutePoint>): void
  (e: 'remove'): void
}

const props = withDefaults(defineProps<Props>(), {
  canRemove: true,
  canMove: true,
})

const emit = defineEmits<Emits>()

// Расширенные поля (скрыты по умолчанию)
const showTime = ref(!!props.point.time_from || !!props.point.time_to)
const showContact = ref(!!props.point.contact_name || !!props.point.contact_phone)
const showComment = ref(!!props.point.comment)

// При изменении данных точки (например при загрузке для редактирования)
// автоматически раскрываем секции с данными
watch(
  () => [props.point.time_from, props.point.time_to],
  ([timeFrom, timeTo]) => {
    if (timeFrom || timeTo) showTime.value = true
  },
  { immediate: true }
)

watch(
  () => [props.point.contact_name, props.point.contact_phone],
  ([name, phone]) => {
    if (name || phone) showContact.value = true
  },
  { immediate: true }
)

watch(
  () => props.point.comment,
  (comment) => {
    if (comment) showComment.value = true
  },
  { immediate: true }
)

// Автологика: первая точка всегда loading, последняя всегда unloading
const isFirstPoint = computed(() => props.index === 0)
const isLastPoint = computed(() => props.index === props.totalPoints - 1)

const locationError = computed(() => props.errors?.[`point_${props.index}_location`] || props.errors?.[`point_${props.index}_address`])
const dateFromError = computed(() => props.errors?.[`point_${props.index}_date_from`])
const dateToError = computed(() => props.errors?.[`point_${props.index}_date_to`])
const contactNameError = computed(() => props.errors?.[`point_${props.index}_contact_name`])
const contactPhoneError = computed(() => props.errors?.[`point_${props.index}_contact_phone`])

// Цвет левой границы зависит от типов
const borderColor = computed(() => {
  if (props.point.is_loading && props.point.is_unloading) {
    return 'border-l-4 border-l-purple-500' // Оба типа
  }
  if (props.point.is_loading) {
    return 'border-l-4 border-l-blue-500'
  }
  if (props.point.is_unloading) {
    return 'border-l-4 border-l-green-500'
  }
  return 'border-l-4 border-l-gray-300'
})

function toggleLoading() {
  // Первая точка всегда loading
  if (isFirstPoint.value) return
  emit('update', { is_loading: !props.point.is_loading })
}

function toggleUnloading() {
  // Последняя точка всегда unloading
  if (isLastPoint.value) return
  emit('update', { is_unloading: !props.point.is_unloading })
}

function handleCountryIdUpdate(value: number | undefined) {
  emit('update', { country_id: value })
}

function handleCityIdUpdate(value: number | undefined) {
  emit('update', { city_id: value })
  if (value) {
    emitTutorial('route:citySelected', { pointIndex: props.index })
  }
}

function handleCoordinatesUpdate(coordinates: Coordinates | undefined) {
  emit('update', { coordinates })
}

function handleDisplayAddressUpdate(value: string) {
  // Update legacy address field for backward compatibility
  emit('update', { address: value })
}

function handleDateFromChange(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update', { date_from: value })
  if (value) {
    emitTutorial('route:dateSet', { pointIndex: props.index })
  }
}

function handleDateToChange(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update', { date_to: value || undefined })
}

function handleTimeFromChange(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update', { time_from: value || undefined })
}

function handleTimeToChange(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update', { time_to: value || undefined })
}

function handleContactNameChange(event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update', { contact_name: value || undefined })
}

// Маска телефона: +7 (XXX) XXX-XX-XX
function formatPhoneNumber(value: string): string {
  const digits = value.replace(/\D/g, '')

  if (digits.length === 0) return ''

  let result = '+7'
  if (digits.length > 1) {
    result += ' (' + digits.substring(1, 4)
  }
  if (digits.length >= 4) {
    result += ') ' + digits.substring(4, 7)
  }
  if (digits.length >= 7) {
    result += '-' + digits.substring(7, 9)
  }
  if (digits.length >= 9) {
    result += '-' + digits.substring(9, 11)
  }

  return result
}

function handlePhoneInput(event: Event) {
  const input = event.target as HTMLInputElement
  let value = input.value

  // Если пустое значение, позволяем очистить
  if (!value) {
    emit('update', { contact_phone: undefined })
    return
  }

  // Добавляем 7 в начало если нет
  let digits = value.replace(/\D/g, '')
  if (digits.length > 0 && digits[0] !== '7') {
    digits = '7' + digits
  }

  const formatted = formatPhoneNumber(digits)
  input.value = formatted

  // Сохраняем только цифры
  emit('update', { contact_phone: digits.length > 1 ? '+' + digits : undefined })
}

function handleCommentChange(event: Event) {
  const value = (event.target as HTMLTextAreaElement).value
  emit('update', { comment: value || undefined })
}

// Функции показа/скрытия полей
function toggleShowTime() {
  showTime.value = true
  emitTutorial('route:timeToggled', { pointIndex: props.index, shown: true })
}

function toggleShowContact() {
  showContact.value = true
  emitTutorial('route:contactToggled', { pointIndex: props.index, shown: true })
}

function toggleShowComment() {
  showComment.value = true
  emitTutorial('route:commentToggled', { pointIndex: props.index, shown: true })
}

// Функции скрытия полей (очищают данные)
function hideTime() {
  showTime.value = false
  emit('update', { time_from: undefined, time_to: undefined })
  emitTutorial('route:timeToggled', { pointIndex: props.index, shown: false })
}

function hideContact() {
  showContact.value = false
  emit('update', { contact_name: undefined, contact_phone: undefined })
  emitTutorial('route:contactToggled', { pointIndex: props.index, shown: false })
}

function hideComment() {
  showComment.value = false
  emit('update', { comment: undefined })
  emitTutorial('route:commentToggled', { pointIndex: props.index, shown: false })
}

// Показ отформатированного телефона
const formattedPhone = computed(() => {
  if (!props.point.contact_phone) return ''
  return formatPhoneNumber(props.point.contact_phone.replace(/\D/g, ''))
})

const inputClass = 'appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 text-sm'
const inputErrorClass = 'appearance-none block w-full px-3 py-2 border border-red-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 text-sm'

// Следим за изменениями позиции для автообновления типов
watch(() => [props.index, props.totalPoints], () => {
  const updates: Partial<RoutePoint> = {}

  // Первая точка: всегда loading, никогда unloading
  if (isFirstPoint.value) {
    if (!props.point.is_loading) updates.is_loading = true
    if (props.point.is_unloading) updates.is_unloading = false
  }

  // Последняя точка: всегда unloading, никогда loading
  if (isLastPoint.value) {
    if (!props.point.is_unloading) updates.is_unloading = true
    if (props.point.is_loading) updates.is_loading = false
  }

  if (Object.keys(updates).length > 0) {
    emit('update', updates)
  }
}, { immediate: true })
</script>

<template>
  <div
    :class="[
      'bg-white border rounded-lg p-4 shadow-sm',
      borderColor,
    ]"
    :data-tutorial="index <= 2 ? `route-point-${index}` : undefined"
  >
    <div class="flex items-start justify-between mb-3">
      <div class="flex items-center gap-3">
        <!-- Drag handle -->
        <div
          v-if="canMove"
          class="drag-handle cursor-move text-gray-400 hover:text-gray-600"
          data-tutorial="route-drag-handle"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path d="M7 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 2zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 14zm6-8a2 2 0 1 0-.001-4.001A2 2 0 0 0 13 6zm0 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 14z" />
          </svg>
        </div>

        <span class="text-sm text-gray-500 font-medium">Точка #{{ index + 1 }}</span>

        <!-- Badge для первой/последней точки (не редактируемый) -->
        <span
          v-if="isFirstPoint"
          class="px-3 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-700 border border-blue-300"
        >
          Погрузка
        </span>
        <span
          v-if="isLastPoint"
          class="px-3 py-1 rounded-full text-xs font-medium bg-green-100 text-green-700 border border-green-300"
        >
          Разгрузка
        </span>
      </div>

      <!-- Remove button -->
      <button
        v-if="canRemove"
        type="button"
        class="text-gray-400 hover:text-red-500 transition-colors"
        title="Удалить точку"
        @click="$emit('remove')"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
          <path
            fill-rule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clip-rule="evenodd"
          />
        </svg>
      </button>
    </div>

    <!-- Чекбоксы для промежуточных точек -->
    <div v-if="!isFirstPoint && !isLastPoint" class="flex gap-4 mb-3">
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          :checked="point.is_loading"
          class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          @change="toggleLoading"
        />
        <span class="text-sm text-gray-700">Погрузка</span>
      </label>
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          :checked="point.is_unloading"
          class="w-4 h-4 text-green-600 border-gray-300 rounded focus:ring-green-500"
          @change="toggleUnloading"
        />
        <span class="text-sm text-gray-700">Разгрузка</span>
      </label>
    </div>

    <div class="space-y-3">
      <!-- Location (Country + City) -->
      <CountryCitySelect
        :country-id="point.country_id"
        :city-id="point.city_id"
        :error="locationError"
        @update:country-id="handleCountryIdUpdate"
        @update:city-id="handleCityIdUpdate"
        @update:coordinates="handleCoordinatesUpdate"
        @update:display-address="handleDisplayAddressUpdate"
      />

      <!-- Date (обязательное) -->
      <div :data-tutorial="index === 0 ? 'route-date-fields' : (index === 2 ? 'route-date-fields-2' : undefined)">
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Дата <span class="text-red-500">*</span>
        </label>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <input
              type="date"
              :value="point.date_from"
              :class="dateFromError ? inputErrorClass : inputClass"
              @change="handleDateFromChange"
            />
            <p v-if="dateFromError" class="mt-1 text-sm text-red-600">
              {{ dateFromError }}
            </p>
          </div>
          <div>
            <input
              type="date"
              :value="point.date_to || ''"
              :class="dateToError ? inputErrorClass : inputClass"
              placeholder="до (опционально)"
              @change="handleDateToChange"
            />
            <p v-if="dateToError" class="mt-1 text-sm text-red-600">
              {{ dateToError }}
            </p>
          </div>
        </div>
      </div>

      <!-- Время (раскрывается по кнопке) -->
      <div v-if="showTime" :data-tutorial="index === 0 ? 'route-time-section' : undefined">
        <div class="flex items-center justify-between mb-1">
          <label class="block text-sm font-medium text-gray-700">
            Время
          </label>
          <button
            type="button"
            class="text-gray-400 hover:text-red-500 text-xs"
            :data-tutorial="index === 0 ? 'route-hide-time' : undefined"
            @click="hideTime"
          >
            Убрать
          </button>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <input
              type="time"
              :value="point.time_from || ''"
              :class="inputClass"
              placeholder="с"
              @change="handleTimeFromChange"
            />
          </div>
          <div>
            <input
              type="time"
              :value="point.time_to || ''"
              :class="inputClass"
              placeholder="до"
              @change="handleTimeToChange"
            />
          </div>
        </div>
      </div>

      <!-- Контакт (раскрывается по кнопке) -->
      <div v-if="showContact" :data-tutorial="index === 0 ? 'route-contact-section' : undefined">
        <div class="flex items-center justify-between mb-1">
          <label class="block text-sm font-medium text-gray-700">
            Контакт
          </label>
          <button
            type="button"
            class="text-gray-400 hover:text-red-500 text-xs"
            :data-tutorial="index === 0 ? 'route-hide-contact' : undefined"
            @click="hideContact"
          >
            Убрать
          </button>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <input
              type="text"
              :value="point.contact_name || ''"
              placeholder="Имя"
              :class="contactNameError ? inputErrorClass : inputClass"
              @input="handleContactNameChange"
            />
            <p v-if="contactNameError" class="mt-1 text-sm text-red-600">
              {{ contactNameError }}
            </p>
          </div>
          <div>
            <input
              type="tel"
              :value="formattedPhone"
              placeholder="+7 (___) ___-__-__"
              :class="contactPhoneError ? inputErrorClass : inputClass"
              @input="handlePhoneInput"
            />
            <p v-if="contactPhoneError" class="mt-1 text-sm text-red-600">
              {{ contactPhoneError }}
            </p>
          </div>
        </div>
      </div>

      <!-- Комментарий (раскрывается по кнопке) -->
      <div v-if="showComment" :data-tutorial="index === 0 ? 'route-comment-section' : undefined">
        <div class="flex items-center justify-between mb-1">
          <label class="block text-sm font-medium text-gray-700">
            Примечание
          </label>
          <button
            type="button"
            class="text-gray-400 hover:text-red-500 text-xs"
            :data-tutorial="index === 0 ? 'route-hide-comment' : undefined"
            @click="hideComment"
          >
            Убрать
          </button>
        </div>
        <textarea
          :value="point.comment || ''"
          placeholder="Дополнительная информация"
          rows="2"
          class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 text-sm resize-none"
          @input="handleCommentChange"
        />
      </div>

      <!-- Secondary кнопки для раскрытия полей -->
      <div class="flex flex-wrap gap-2 pt-2">
        <button
          v-if="!showTime"
          type="button"
          class="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1"
          :data-tutorial="index === 0 ? 'route-add-time' : undefined"
          @click="toggleShowTime"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clip-rule="evenodd" />
          </svg>
          Время
        </button>
        <button
          v-if="!showContact"
          type="button"
          class="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1"
          :data-tutorial="index === 0 ? 'route-add-contact' : undefined"
          @click="toggleShowContact"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clip-rule="evenodd" />
          </svg>
          Контакт
        </button>
        <button
          v-if="!showComment"
          type="button"
          class="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1"
          :data-tutorial="index === 0 ? 'route-add-comment' : undefined"
          @click="toggleShowComment"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clip-rule="evenodd" />
          </svg>
          Примечание
        </button>
      </div>
    </div>
  </div>
</template>
