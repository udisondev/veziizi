<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { vMaska } from 'maska/vue'
import { useRegistrationForm, innLabels, innPlaceholders, countryNames } from '@/composables/useRegistrationForm'
import { organizationsApi } from '@/api/organizations'
import { getFingerprint } from '@/composables/useFingerprint'
import type { Country } from '@/types/registration'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// Utils
import { cn } from '@/lib/utils'

// Icons
import { AlertCircle, Check, ChevronLeft, ChevronRight, UserPlus } from 'lucide-vue-next'

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
    const fingerprint = await getFingerprint()
    await organizationsApi.register(requestData.value, fingerprint)
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

const stepTitles = ['Организация', 'Владелец', 'Подтверждение']
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background py-12 px-4">
    <Card class="w-full max-w-lg">
      <CardHeader class="text-center">
        <CardTitle class="text-2xl">Регистрация организации</CardTitle>
        <CardDescription>Шаг {{ currentStep }} из {{ totalSteps }}: {{ stepTitles[currentStep - 1] }}</CardDescription>
      </CardHeader>

      <CardContent>
        <!-- Step Indicator -->
        <div class="flex items-center justify-center mb-6">
          <template v-for="step in totalSteps" :key="step">
            <button
              type="button"
              :class="[
                'w-10 h-10 rounded-full flex items-center justify-center font-medium transition-colors',
                step === currentStep
                  ? 'bg-primary text-primary-foreground'
                  : step < currentStep
                    ? 'bg-success text-success-foreground cursor-pointer hover:bg-success/90'
                    : 'bg-muted text-muted-foreground cursor-not-allowed',
              ]"
              :disabled="step > currentStep"
              @click="step < currentStep && goToStep(step)"
            >
              <Check v-if="step < currentStep" class="h-5 w-5" />
              <span v-else>{{ step }}</span>
            </button>
            <div
              v-if="step < totalSteps"
              :class="['w-12 h-1 rounded mx-1', step < currentStep ? 'bg-success' : 'bg-muted']"
            />
          </template>
        </div>

        <!-- API Error -->
        <div
          v-if="apiError"
          class="mb-6 flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
        >
          <AlertCircle class="h-4 w-4 shrink-0" />
          {{ apiError }}
        </div>

        <form @submit.prevent="onFormSubmit" class="space-y-4">
          <!-- Step 1: Organization -->
          <div v-show="currentStep === 1" class="space-y-4">
            <div class="space-y-2">
              <Label for="name">Название организации</Label>
              <Input
                id="name"
                v-model="organization.name"
                :class="cn(errors.name && 'border-destructive')"
                @blur="validateField('name')"
              />
              <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
            </div>

            <div class="space-y-2">
              <Label for="legal_name">Юридическое название</Label>
              <Input
                id="legal_name"
                v-model="organization.legal_name"
                :class="cn(errors.legal_name && 'border-destructive')"
                @blur="validateField('legal_name')"
              />
              <p v-if="errors.legal_name" class="text-sm text-destructive">{{ errors.legal_name }}</p>
            </div>

            <div class="space-y-2">
              <Label for="country">Страна</Label>
              <Select v-model="organization.country">
                <SelectTrigger>
                  <SelectValue :placeholder="countryNames[organization.country]" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="c in countries" :key="c" :value="c">
                    {{ countryNames[c] }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <Label for="inn">{{ innLabels[organization.country] }}</Label>
              <Input
                id="inn"
                v-model="organization.inn"
                v-maska
                data-maska="############"
                inputmode="numeric"
                :maxlength="innMaxLength[organization.country]"
                :placeholder="innPlaceholders[organization.country]"
                :class="cn(errors.inn && 'border-destructive')"
                @blur="validateField('inn')"
              />
              <p v-if="errors.inn" class="text-sm text-destructive">{{ errors.inn }}</p>
            </div>

            <div class="space-y-2">
              <Label for="phone">Телефон организации</Label>
              <Input
                id="phone"
                v-model="organization.phone"
                v-maska
                :data-maska="currentPhoneMask"
                type="tel"
                inputmode="tel"
                :placeholder="currentPhonePlaceholder"
                :class="cn(errors.phone && 'border-destructive')"
                @blur="validateField('phone')"
              />
              <p v-if="errors.phone" class="text-sm text-destructive">{{ errors.phone }}</p>
            </div>

            <div class="space-y-2">
              <Label for="email">Email организации</Label>
              <Input
                id="email"
                v-model="organization.email"
                type="email"
                :class="cn(errors.email && 'border-destructive')"
                @blur="validateField('email')"
              />
              <p v-if="errors.email" class="text-sm text-destructive">{{ errors.email }}</p>
            </div>

            <div class="space-y-2">
              <Label for="address">Адрес</Label>
              <Input
                id="address"
                v-model="organization.address"
                :class="cn(errors.address && 'border-destructive')"
                @blur="validateField('address')"
              />
              <p v-if="errors.address" class="text-sm text-destructive">{{ errors.address }}</p>
            </div>
          </div>

          <!-- Step 2: Owner -->
          <div v-show="currentStep === 2" class="space-y-4">
            <div class="space-y-2">
              <Label for="owner_name">Имя владельца</Label>
              <Input
                id="owner_name"
                v-model="owner.owner_name"
                :class="cn(errors.owner_name && 'border-destructive')"
                @blur="validateField('owner_name')"
              />
              <p v-if="errors.owner_name" class="text-sm text-destructive">{{ errors.owner_name }}</p>
            </div>

            <div class="space-y-2">
              <Label for="owner_email">Email владельца</Label>
              <Input
                id="owner_email"
                v-model="owner.owner_email"
                type="email"
                :class="cn(errors.owner_email && 'border-destructive')"
                @blur="validateField('owner_email')"
              />
              <p v-if="errors.owner_email" class="text-sm text-destructive">{{ errors.owner_email }}</p>
            </div>

            <div class="space-y-2">
              <Label for="owner_phone">Телефон владельца</Label>
              <Input
                id="owner_phone"
                v-model="owner.owner_phone"
                v-maska
                :data-maska="currentPhoneMask"
                type="tel"
                inputmode="tel"
                :placeholder="currentPhonePlaceholder"
                :class="cn(errors.owner_phone && 'border-destructive')"
                @blur="validateField('owner_phone')"
              />
              <p v-if="errors.owner_phone" class="text-sm text-destructive">{{ errors.owner_phone }}</p>
            </div>

            <div class="space-y-2">
              <Label for="owner_password">Пароль</Label>
              <Input
                id="owner_password"
                v-model="owner.owner_password"
                type="password"
                placeholder="Минимум 8 символов"
                :class="cn(errors.owner_password && 'border-destructive')"
                @blur="validateField('owner_password')"
              />
              <p v-if="errors.owner_password" class="text-sm text-destructive">{{ errors.owner_password }}</p>
            </div>

            <div class="space-y-2">
              <Label for="confirm_password">Подтверждение пароля</Label>
              <Input
                id="confirm_password"
                v-model="owner.confirm_password"
                type="password"
                :class="cn(errors.confirm_password && 'border-destructive')"
                @blur="validateField('confirm_password')"
              />
              <p v-if="errors.confirm_password" class="text-sm text-destructive">{{ errors.confirm_password }}</p>
            </div>
          </div>

          <!-- Step 3: Confirmation -->
          <div v-show="currentStep === 3" class="space-y-6">
            <h3 class="text-lg font-medium text-foreground">Проверьте данные</h3>

            <div class="rounded-lg border bg-muted/50 p-4 space-y-2">
              <h4 class="font-medium text-foreground">Организация</h4>
              <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                <dt class="text-muted-foreground">Название:</dt>
                <dd class="text-foreground">{{ organization.name }}</dd>
                <dt class="text-muted-foreground">Юр. название:</dt>
                <dd class="text-foreground">{{ organization.legal_name }}</dd>
                <dt class="text-muted-foreground">{{ innLabels[organization.country] }}:</dt>
                <dd class="text-foreground">{{ organization.inn }}</dd>
                <dt class="text-muted-foreground">Страна:</dt>
                <dd class="text-foreground">{{ countryNames[organization.country] }}</dd>
                <dt class="text-muted-foreground">Телефон:</dt>
                <dd class="text-foreground">{{ organization.phone }}</dd>
                <dt class="text-muted-foreground">Email:</dt>
                <dd class="text-foreground">{{ organization.email }}</dd>
                <dt class="text-muted-foreground">Адрес:</dt>
                <dd class="text-foreground">{{ organization.address }}</dd>
              </dl>
            </div>

            <div class="rounded-lg border bg-muted/50 p-4 space-y-2">
              <h4 class="font-medium text-foreground">Владелец</h4>
              <dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                <dt class="text-muted-foreground">Имя:</dt>
                <dd class="text-foreground">{{ owner.owner_name }}</dd>
                <dt class="text-muted-foreground">Email:</dt>
                <dd class="text-foreground">{{ owner.owner_email }}</dd>
                <dt class="text-muted-foreground">Телефон:</dt>
                <dd class="text-foreground">{{ owner.owner_phone }}</dd>
              </dl>
            </div>

            <p class="text-sm text-muted-foreground">
              После регистрации организация будет отправлена на модерацию.
              Вы сможете войти в систему сразу после регистрации.
            </p>
          </div>

          <!-- Navigation Buttons -->
          <div class="flex gap-3 pt-2">
            <Button
              v-if="currentStep > 1"
              type="button"
              variant="outline"
              class="flex-1"
              @click="prevStep"
            >
              <ChevronLeft class="mr-1 h-4 w-4" />
              Назад
            </Button>
            <Button
              type="submit"
              :disabled="isLoading"
              class="flex-1"
            >
              <template v-if="currentStep === 3">
                <UserPlus v-if="!isLoading" class="mr-1 h-4 w-4" />
                {{ isLoading ? 'Регистрация...' : 'Зарегистрировать' }}
              </template>
              <template v-else>
                Далее
                <ChevronRight class="ml-1 h-4 w-4" />
              </template>
            </Button>
          </div>
        </form>
      </CardContent>

      <CardFooter class="justify-center">
        <p class="text-sm text-muted-foreground">
          Уже есть аккаунт?
          <router-link to="/login" class="text-primary hover:underline">
            Войти
          </router-link>
        </p>
      </CardFooter>
    </Card>
  </div>
</template>
