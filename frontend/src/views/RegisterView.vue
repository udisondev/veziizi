<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { vMaska } from 'maska/vue'
import { useRegistrationForm, innLabels, innPlaceholders, countryNames } from '@/composables/useRegistrationForm'
import { organizationsApi } from '@/api/organizations'
import type { Country } from '@/types/registration'

const router = useRouter()
const {
  currentStep,
  totalSteps,
  organization,
  owner,
  errors,
  requestData,
  nextStep,
  prevStep,
  goToStep,
  validateField,
} = useRegistrationForm()

const isLoading = ref(false)
const apiError = ref('')

async function handleSubmit() {
  apiError.value = ''
  isLoading.value = true

  try {
    await organizationsApi.register(requestData.value)
    router.push({
      path: '/login',
      query: { registered: 'true' },
    })
  } catch (e) {
    apiError.value = e instanceof Error ? e.message : 'Ошибка регистрации'
  } finally {
    isLoading.value = false
  }
}

function onFormSubmit() {
  if (currentStep.value === 3) {
    handleSubmit()
  } else {
    nextStep()
  }
}

const countries: Country[] = ['RU', 'KZ', 'BY']

// Маски телефонов по странам
const phoneMasks: Record<Country, string> = {
  RU: '+7 (###) ###-##-##',
  KZ: '+7 (###) ###-##-##',
  BY: '+375 (##) ###-##-##',
}

const phonePlaceholders: Record<Country, string> = {
  RU: '+7 (999) 999-99-99',
  KZ: '+7 (999) 999-99-99',
  BY: '+375 (99) 999-99-99',
}

// Максимальная длина ИНН по странам
const innMaxLength: Record<Country, number> = {
  RU: 12,
  KZ: 12,
  BY: 9,
}

const currentPhoneMask = computed(() => phoneMasks[organization.country])
const currentPhonePlaceholder = computed(() => phonePlaceholders[organization.country])

const inputClass = (field: string) => [
  'appearance-none block w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500',
  errors[field] ? 'border-red-300' : 'border-gray-300',
]
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4">
    <div class="max-w-lg w-full space-y-8">
      <div>
        <h2 class="text-center text-3xl font-extrabold text-gray-900">
          Регистрация организации
        </h2>
      </div>

      <!-- Step Indicator -->
      <div class="flex items-center justify-center space-x-4">
        <template v-for="step in totalSteps" :key="step">
          <button
            type="button"
            :class="[
              'w-10 h-10 rounded-full flex items-center justify-center font-medium transition-colors',
              step === currentStep
                ? 'bg-blue-600 text-white'
                : step < currentStep
                  ? 'bg-green-500 text-white cursor-pointer hover:bg-green-600'
                  : 'bg-gray-200 text-gray-500 cursor-not-allowed',
            ]"
            :disabled="step > currentStep"
            @click="step < currentStep && goToStep(step)"
          >
            <span v-if="step < currentStep">&#10003;</span>
            <span v-else>{{ step }}</span>
          </button>
          <div
            v-if="step < totalSteps"
            :class="['w-12 h-1 rounded', step < currentStep ? 'bg-green-500' : 'bg-gray-200']"
          />
        </template>
      </div>

      <!-- API Error -->
      <div v-if="apiError" class="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded">
        {{ apiError }}
      </div>

      <form class="space-y-6" @submit.prevent="onFormSubmit">
        <!-- Step 1: Organization -->
        <div v-show="currentStep === 1" class="space-y-4">
          <div>
            <label for="name" class="block text-sm font-medium text-gray-700 mb-1">
              Название организации
            </label>
            <input
              id="name"
              v-model="organization.name"
              type="text"
              :class="inputClass('name')"
              @blur="validateField('name')"
            />
            <p v-if="errors.name" class="mt-1 text-sm text-red-600">{{ errors.name }}</p>
          </div>

          <div>
            <label for="legal_name" class="block text-sm font-medium text-gray-700 mb-1">
              Юридическое название
            </label>
            <input
              id="legal_name"
              v-model="organization.legal_name"
              type="text"
              :class="inputClass('legal_name')"
              @blur="validateField('legal_name')"
            />
            <p v-if="errors.legal_name" class="mt-1 text-sm text-red-600">{{ errors.legal_name }}</p>
          </div>

          <div>
            <label for="country" class="block text-sm font-medium text-gray-700 mb-1">
              Страна
            </label>
            <select
              id="country"
              v-model="organization.country"
              class="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            >
              <option v-for="c in countries" :key="c" :value="c">
                {{ countryNames[c] }}
              </option>
            </select>
          </div>

          <div>
            <label for="inn" class="block text-sm font-medium text-gray-700 mb-1">
              {{ innLabels[organization.country] }}
            </label>
            <input
              id="inn"
              v-model="organization.inn"
              v-maska
              data-maska="############"
              type="text"
              inputmode="numeric"
              :maxlength="innMaxLength[organization.country]"
              :placeholder="innPlaceholders[organization.country]"
              :class="inputClass('inn')"
              @blur="validateField('inn')"
            />
            <p v-if="errors.inn" class="mt-1 text-sm text-red-600">{{ errors.inn }}</p>
          </div>

          <div>
            <label for="phone" class="block text-sm font-medium text-gray-700 mb-1">
              Телефон организации
            </label>
            <input
              id="phone"
              v-model="organization.phone"
              v-maska
              :data-maska="currentPhoneMask"
              type="tel"
              inputmode="tel"
              :placeholder="currentPhonePlaceholder"
              :class="inputClass('phone')"
              @blur="validateField('phone')"
            />
            <p v-if="errors.phone" class="mt-1 text-sm text-red-600">{{ errors.phone }}</p>
          </div>

          <div>
            <label for="email" class="block text-sm font-medium text-gray-700 mb-1">
              Email организации
            </label>
            <input
              id="email"
              v-model="organization.email"
              type="email"
              :class="inputClass('email')"
              @blur="validateField('email')"
            />
            <p v-if="errors.email" class="mt-1 text-sm text-red-600">{{ errors.email }}</p>
          </div>

          <div>
            <label for="address" class="block text-sm font-medium text-gray-700 mb-1">
              Адрес
            </label>
            <input
              id="address"
              v-model="organization.address"
              type="text"
              :class="inputClass('address')"
              @blur="validateField('address')"
            />
            <p v-if="errors.address" class="mt-1 text-sm text-red-600">{{ errors.address }}</p>
          </div>
        </div>

        <!-- Step 2: Owner -->
        <div v-show="currentStep === 2" class="space-y-4">
          <div>
            <label for="owner_name" class="block text-sm font-medium text-gray-700 mb-1">
              Имя владельца
            </label>
            <input
              id="owner_name"
              v-model="owner.owner_name"
              type="text"
              :class="inputClass('owner_name')"
              @blur="validateField('owner_name')"
            />
            <p v-if="errors.owner_name" class="mt-1 text-sm text-red-600">{{ errors.owner_name }}</p>
          </div>

          <div>
            <label for="owner_email" class="block text-sm font-medium text-gray-700 mb-1">
              Email владельца
            </label>
            <input
              id="owner_email"
              v-model="owner.owner_email"
              type="email"
              :class="inputClass('owner_email')"
              @blur="validateField('owner_email')"
            />
            <p v-if="errors.owner_email" class="mt-1 text-sm text-red-600">{{ errors.owner_email }}</p>
          </div>

          <div>
            <label for="owner_phone" class="block text-sm font-medium text-gray-700 mb-1">
              Телефон владельца
            </label>
            <input
              id="owner_phone"
              v-model="owner.owner_phone"
              v-maska
              :data-maska="currentPhoneMask"
              type="tel"
              inputmode="tel"
              :placeholder="currentPhonePlaceholder"
              :class="inputClass('owner_phone')"
              @blur="validateField('owner_phone')"
            />
            <p v-if="errors.owner_phone" class="mt-1 text-sm text-red-600">{{ errors.owner_phone }}</p>
          </div>

          <div>
            <label for="owner_password" class="block text-sm font-medium text-gray-700 mb-1">
              Пароль
            </label>
            <input
              id="owner_password"
              v-model="owner.owner_password"
              type="password"
              placeholder="Минимум 8 символов"
              :class="inputClass('owner_password')"
              @blur="validateField('owner_password')"
            />
            <p v-if="errors.owner_password" class="mt-1 text-sm text-red-600">{{ errors.owner_password }}</p>
          </div>

          <div>
            <label for="confirm_password" class="block text-sm font-medium text-gray-700 mb-1">
              Подтверждение пароля
            </label>
            <input
              id="confirm_password"
              v-model="owner.confirm_password"
              type="password"
              :class="inputClass('confirm_password')"
              @blur="validateField('confirm_password')"
            />
            <p v-if="errors.confirm_password" class="mt-1 text-sm text-red-600">{{ errors.confirm_password }}</p>
          </div>
        </div>

        <!-- Step 3: Confirmation -->
        <div v-show="currentStep === 3" class="space-y-6">
          <h3 class="text-lg font-medium text-gray-900">Проверьте данные</h3>

          <div class="bg-gray-50 rounded-lg p-4 space-y-2">
            <h4 class="font-medium text-gray-700">Организация</h4>
            <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
              <dt class="text-gray-500">Название:</dt>
              <dd class="text-gray-900">{{ organization.name }}</dd>
              <dt class="text-gray-500">Юр. название:</dt>
              <dd class="text-gray-900">{{ organization.legal_name }}</dd>
              <dt class="text-gray-500">{{ innLabels[organization.country] }}:</dt>
              <dd class="text-gray-900">{{ organization.inn }}</dd>
              <dt class="text-gray-500">Страна:</dt>
              <dd class="text-gray-900">{{ countryNames[organization.country] }}</dd>
              <dt class="text-gray-500">Телефон:</dt>
              <dd class="text-gray-900">{{ organization.phone }}</dd>
              <dt class="text-gray-500">Email:</dt>
              <dd class="text-gray-900">{{ organization.email }}</dd>
              <dt class="text-gray-500">Адрес:</dt>
              <dd class="text-gray-900">{{ organization.address }}</dd>
            </dl>
          </div>

          <div class="bg-gray-50 rounded-lg p-4 space-y-2">
            <h4 class="font-medium text-gray-700">Владелец</h4>
            <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
              <dt class="text-gray-500">Имя:</dt>
              <dd class="text-gray-900">{{ owner.owner_name }}</dd>
              <dt class="text-gray-500">Email:</dt>
              <dd class="text-gray-900">{{ owner.owner_email }}</dd>
              <dt class="text-gray-500">Телефон:</dt>
              <dd class="text-gray-900">{{ owner.owner_phone }}</dd>
            </dl>
          </div>

          <p class="text-sm text-gray-500">
            После регистрации организация будет отправлена на модерацию.
            Вы сможете войти в систему сразу после регистрации.
          </p>
        </div>

        <!-- Navigation Buttons -->
        <div class="flex gap-4">
          <button
            v-if="currentStep > 1"
            type="button"
            class="flex-1 py-2 px-4 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            @click="prevStep"
          >
            Назад
          </button>
          <button
            type="submit"
            :disabled="isLoading"
            class="flex-1 py-2 px-4 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
          >
            {{ isLoading ? 'Регистрация...' : currentStep === 3 ? 'Зарегистрировать' : 'Далее' }}
          </button>
        </div>
      </form>

      <div class="text-center">
        <router-link to="/login" class="text-blue-600 hover:text-blue-500">
          Уже есть аккаунт? Войти
        </router-link>
      </div>
    </div>
  </div>
</template>
