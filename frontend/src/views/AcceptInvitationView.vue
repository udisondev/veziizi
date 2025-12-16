<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { vMaska } from 'maska/vue'
import { invitationsApi } from '@/api/invitations'
import type { InvitationDetails, AcceptInvitationRequest } from '@/types/invitation'

const route = useRoute()
const router = useRouter()

const token = computed(() => route.params.token as string)

// Loading states
const isLoading = ref(true)
const isSubmitting = ref(false)
const loadError = ref<string | null>(null)
const submitError = ref<string | null>(null)

// Invitation data
const invitation = ref<InvitationDetails | null>(null)

// Form data
const form = ref({
  name: '',
  phone: '',
  password: '',
  confirmPassword: '',
})

// Validation errors
const errors = ref<Record<string, string>>({})

// Computed - whether name/phone are prefilled (readonly)
const isNamePrefilled = computed(() => !!invitation.value?.name)
const isPhonePrefilled = computed(() => !!invitation.value?.phone)

// Phone mask
const phoneMask = '+7 (###) ###-##-##'
const phonePlaceholder = '+7 (999) 999-99-99'

function getRoleLabel(role: string): string {
  switch (role) {
    case 'employee': return 'Сотрудник'
    case 'administrator': return 'Администратор'
    default: return role
  }
}

async function loadInvitation() {
  isLoading.value = true
  loadError.value = null

  try {
    const data = await invitationsApi.getByToken(token.value)
    invitation.value = data

    // Pre-fill form with invitation data
    if (data.name) {
      form.value.name = data.name
    }
    if (data.phone) {
      form.value.phone = data.phone
    }

    // Check if expired or already accepted
    if (data.status === 'expired') {
      loadError.value = 'Приглашение истекло'
    } else if (data.status === 'accepted') {
      loadError.value = 'Приглашение уже использовано'
    }
  } catch (e: any) {
    if (e?.status === 404) {
      loadError.value = 'Приглашение не найдено'
    } else if (e?.message?.includes('expired')) {
      loadError.value = 'Приглашение истекло'
    } else {
      loadError.value = e?.message || 'Не удалось загрузить приглашение'
    }
    console.error(e)
  } finally {
    isLoading.value = false
  }
}

function validateField(field: string) {
  errors.value[field] = ''

  switch (field) {
    case 'name':
      if (!isNamePrefilled.value && !form.value.name.trim()) {
        errors.value.name = 'Введите ФИО'
      } else if (form.value.name.trim().length < 2) {
        errors.value.name = 'ФИО слишком короткое'
      }
      break
    case 'phone':
      if (!isPhonePrefilled.value && !form.value.phone.trim()) {
        errors.value.phone = 'Введите телефон'
      } else if (form.value.phone && form.value.phone.replace(/\D/g, '').length < 11) {
        errors.value.phone = 'Неполный номер телефона'
      }
      break
    case 'password':
      if (!form.value.password) {
        errors.value.password = 'Введите пароль'
      } else if (form.value.password.length < 8) {
        errors.value.password = 'Минимум 8 символов'
      }
      break
    case 'confirmPassword':
      if (!form.value.confirmPassword) {
        errors.value.confirmPassword = 'Подтвердите пароль'
      } else if (form.value.password !== form.value.confirmPassword) {
        errors.value.confirmPassword = 'Пароли не совпадают'
      }
      break
  }
}

function validateAll(): boolean {
  validateField('name')
  validateField('phone')
  validateField('password')
  validateField('confirmPassword')
  return !Object.values(errors.value).some(e => !!e)
}

async function handleSubmit() {
  if (!validateAll()) return

  isSubmitting.value = true
  submitError.value = null

  try {
    const request: AcceptInvitationRequest = {
      password: form.value.password,
    }

    // Only send name/phone if they were not prefilled
    if (!isNamePrefilled.value) {
      request.name = form.value.name.trim()
    }
    if (!isPhonePrefilled.value) {
      request.phone = form.value.phone.trim()
    }

    await invitationsApi.accept(token.value, request)

    // Redirect to login with success message
    router.push({
      path: '/login',
      query: { invitation_accepted: 'true' },
    })
  } catch (e: any) {
    if (e?.message?.includes('expired')) {
      submitError.value = 'Приглашение истекло'
    } else if (e?.message?.includes('already')) {
      submitError.value = 'Приглашение уже использовано'
    } else {
      submitError.value = e?.message || 'Не удалось принять приглашение'
    }
    console.error(e)
  } finally {
    isSubmitting.value = false
  }
}

const inputClass = (field: string, readonly = false) => [
  'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none',
  readonly
    ? 'bg-gray-100 text-gray-600 cursor-not-allowed border-gray-200'
    : errors.value[field]
      ? 'border-red-300 focus:ring-red-500 focus:border-red-500'
      : 'border-gray-300 focus:ring-blue-500 focus:border-blue-500',
]

onMounted(() => {
  loadInvitation()
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4">
    <div class="max-w-md w-full space-y-8">
      <!-- Loading -->
      <div v-if="isLoading" class="text-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
        <p class="mt-2 text-gray-600">Загрузка приглашения...</p>
      </div>

      <!-- Error state -->
      <div v-else-if="loadError" class="text-center space-y-6">
        <div class="bg-red-50 border border-red-200 rounded-lg p-6">
          <svg class="mx-auto h-12 w-12 text-red-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <h3 class="text-lg font-medium text-red-800 mb-2">Ошибка</h3>
          <p class="text-red-600">{{ loadError }}</p>
        </div>
        <router-link
          to="/login"
          class="inline-block px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          Перейти к входу
        </router-link>
      </div>

      <!-- Form -->
      <template v-else-if="invitation">
        <div>
          <h2 class="text-center text-3xl font-extrabold text-gray-900">
            Принять приглашение
          </h2>
          <p class="mt-2 text-center text-gray-600">
            Вы приглашены в организацию
          </p>
        </div>

        <!-- Organization info -->
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <h3 class="font-medium text-blue-900 mb-2">{{ invitation.organization_name }}</h3>
          <div class="text-sm text-blue-700 space-y-1">
            <p>
              <span class="text-blue-600">Роль:</span>
              {{ getRoleLabel(invitation.role) }}
            </p>
            <p>
              <span class="text-blue-600">Email:</span>
              {{ invitation.email }}
            </p>
          </div>
        </div>

        <!-- Submit error -->
        <div v-if="submitError" class="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded">
          {{ submitError }}
        </div>

        <form class="space-y-6" @submit.prevent="handleSubmit">
          <!-- Name field -->
          <div>
            <label for="name" class="block text-sm font-medium text-gray-700 mb-1">
              ФИО
              <span v-if="!isNamePrefilled" class="text-red-500">*</span>
              <span v-else class="text-gray-400 font-normal">(заполнено организатором)</span>
            </label>
            <input
              id="name"
              v-model="form.name"
              type="text"
              :readonly="isNamePrefilled"
              :class="inputClass('name', isNamePrefilled)"
              placeholder="Иванов Иван Иванович"
              @blur="validateField('name')"
            />
            <p v-if="errors.name" class="mt-1 text-sm text-red-600">{{ errors.name }}</p>
          </div>

          <!-- Phone field -->
          <div>
            <label for="phone" class="block text-sm font-medium text-gray-700 mb-1">
              Телефон
              <span v-if="!isPhonePrefilled" class="text-red-500">*</span>
              <span v-else class="text-gray-400 font-normal">(заполнено организатором)</span>
            </label>
            <input
              id="phone"
              v-model="form.phone"
              v-maska
              :data-maska="phoneMask"
              type="tel"
              inputmode="tel"
              :readonly="isPhonePrefilled"
              :class="inputClass('phone', isPhonePrefilled)"
              :placeholder="phonePlaceholder"
              @blur="validateField('phone')"
            />
            <p v-if="errors.phone" class="mt-1 text-sm text-red-600">{{ errors.phone }}</p>
          </div>

          <!-- Password field -->
          <div>
            <label for="password" class="block text-sm font-medium text-gray-700 mb-1">
              Пароль <span class="text-red-500">*</span>
            </label>
            <input
              id="password"
              v-model="form.password"
              type="password"
              :class="inputClass('password')"
              placeholder="Минимум 8 символов"
              @blur="validateField('password')"
            />
            <p v-if="errors.password" class="mt-1 text-sm text-red-600">{{ errors.password }}</p>
          </div>

          <!-- Confirm password field -->
          <div>
            <label for="confirmPassword" class="block text-sm font-medium text-gray-700 mb-1">
              Подтверждение пароля <span class="text-red-500">*</span>
            </label>
            <input
              id="confirmPassword"
              v-model="form.confirmPassword"
              type="password"
              :class="inputClass('confirmPassword')"
              @blur="validateField('confirmPassword')"
            />
            <p v-if="errors.confirmPassword" class="mt-1 text-sm text-red-600">{{ errors.confirmPassword }}</p>
          </div>

          <!-- Submit button -->
          <button
            type="submit"
            :disabled="isSubmitting"
            class="w-full py-2 px-4 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
          >
            {{ isSubmitting ? 'Отправка...' : 'Принять приглашение' }}
          </button>
        </form>

        <div class="text-center">
          <router-link to="/login" class="text-blue-600 hover:text-blue-500">
            Уже есть аккаунт? Войти
          </router-link>
        </div>
      </template>
    </div>
  </div>
</template>
