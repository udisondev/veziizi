<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { vMaska } from 'maska/vue'
import { invitationsApi } from '@/api/invitations'
import { useAuthStore } from '@/stores/auth'
import { getFingerprint } from '@/composables/useFingerprint'
import type { InvitationDetails, AcceptInvitationRequest } from '@/types/invitation'
import { logger } from '@/utils/logger'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

// Shared Components
import { LoadingSpinner } from '@/components/shared'

// Icons
import { Building2, Mail, Shield, AlertCircle, CheckCircle2 } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const token = computed(() => route.params.token as string)

// Loading states
const isLoading = ref(true)
const isSubmitting = ref(false)
const loadError = ref<string | null>(null)
const submitError = ref<string | null>(null)
const showSuccess = ref(false)

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
  } catch (e: unknown) {
    if (e instanceof Error && 'status' in e && (e as { status: number }).status === 404) {
      loadError.value = 'Приглашение не найдено'
    } else if (e instanceof Error && e.message?.includes('expired')) {
      loadError.value = 'Приглашение истекло'
    } else {
      loadError.value = e instanceof Error ? e.message : 'Не удалось загрузить приглашение'
    }
    logger.error('Failed to load invitation', e)
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

    const fingerprint = await getFingerprint()
    await invitationsApi.accept(token.value, request, fingerprint)

    // Show success animation
    showSuccess.value = true

    // Redirect after 2 seconds
    setTimeout(() => {
      if (auth.isAuthenticated) {
        router.push('/')
      } else {
        router.push({
          path: '/login',
          query: { invitation_accepted: 'true' },
        })
      }
    }, 2000)
  } catch (e: unknown) {
    const message = e instanceof Error ? e.message : ''
    if (message.includes('expired')) {
      submitError.value = 'Приглашение истекло'
    } else if (message.includes('already')) {
      submitError.value = 'Приглашение уже использовано'
    } else {
      submitError.value = message || 'Не удалось принять приглашение'
    }
    logger.error('Failed to accept invitation', e)
  } finally {
    isSubmitting.value = false
  }
}

onMounted(() => {
  loadInvitation()
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background py-12 px-4">
    <!-- Success animation -->
    <div v-if="showSuccess" class="success-overlay">
      <div class="success-circle">
        <CheckCircle2 class="h-10 w-10 text-white" />
      </div>
    </div>

    <template v-else>
      <!-- Loading -->
      <LoadingSpinner v-if="isLoading" text="Загрузка приглашения..." />

      <!-- Error state -->
      <Card v-else-if="loadError" class="w-full max-w-md">
        <CardContent class="pt-6 text-center space-y-6">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-destructive/10">
            <AlertCircle class="h-8 w-8 text-destructive" />
          </div>
          <div>
            <h3 class="text-lg font-semibold text-foreground mb-2">Ошибка</h3>
            <p class="text-muted-foreground">{{ loadError }}</p>
          </div>
          <Button @click="router.push('/login')">
            Перейти к входу
          </Button>
        </CardContent>
      </Card>

      <!-- Form -->
      <Card v-else-if="invitation" class="w-full max-w-md">
        <CardHeader class="text-center">
          <CardTitle class="text-2xl">Принять приглашение</CardTitle>
          <CardDescription>Вы приглашены в организацию</CardDescription>
        </CardHeader>

        <CardContent class="space-y-6">
          <!-- Organization info -->
          <div class="rounded-lg border border-primary/20 bg-primary/5 p-4">
            <div class="flex items-center gap-3 mb-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <Building2 class="h-5 w-5 text-primary" />
              </div>
              <div>
                <p class="font-medium text-foreground">{{ invitation.organization_name }}</p>
              </div>
            </div>
            <div class="flex flex-wrap gap-4 text-sm">
              <div class="flex items-center gap-2">
                <Shield class="h-4 w-4 text-muted-foreground" />
                <Badge variant="secondary">{{ getRoleLabel(invitation.role) }}</Badge>
              </div>
              <div class="flex items-center gap-2 text-muted-foreground">
                <Mail class="h-4 w-4" />
                {{ invitation.email }}
              </div>
            </div>
          </div>

          <!-- Submit error -->
          <div
            v-if="submitError"
            class="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
          >
            <AlertCircle class="h-4 w-4 shrink-0" />
            {{ submitError }}
          </div>

          <form @submit.prevent="handleSubmit" class="space-y-4">
            <!-- Name field -->
            <div class="space-y-2">
              <Label for="name">
                ФИО
                <span v-if="!isNamePrefilled" class="text-destructive">*</span>
                <span v-else class="text-muted-foreground font-normal">(заполнено организатором)</span>
              </Label>
              <Input
                id="name"
                v-model="form.name"
                :readonly="isNamePrefilled"
                :class="cn(
                  errors.name && 'border-destructive',
                  isNamePrefilled && 'bg-muted cursor-not-allowed'
                )"
                placeholder="Иванов Иван Иванович"
                @blur="validateField('name')"
              />
              <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
            </div>

            <!-- Phone field -->
            <div class="space-y-2">
              <Label for="phone">
                Телефон
                <span v-if="!isPhonePrefilled" class="text-destructive">*</span>
                <span v-else class="text-muted-foreground font-normal">(заполнено организатором)</span>
              </Label>
              <Input
                id="phone"
                v-model="form.phone"
                v-maska
                :data-maska="phoneMask"
                type="tel"
                inputmode="tel"
                :readonly="isPhonePrefilled"
                :class="cn(
                  errors.phone && 'border-destructive',
                  isPhonePrefilled && 'bg-muted cursor-not-allowed'
                )"
                :placeholder="phonePlaceholder"
                @blur="validateField('phone')"
              />
              <p v-if="errors.phone" class="text-sm text-destructive">{{ errors.phone }}</p>
            </div>

            <!-- Password field -->
            <div class="space-y-2">
              <Label for="password">
                Пароль <span class="text-destructive">*</span>
              </Label>
              <Input
                id="password"
                v-model="form.password"
                type="password"
                :class="cn(errors.password && 'border-destructive')"
                placeholder="Минимум 8 символов"
                @blur="validateField('password')"
              />
              <p v-if="errors.password" class="text-sm text-destructive">{{ errors.password }}</p>
            </div>

            <!-- Confirm password field -->
            <div class="space-y-2">
              <Label for="confirmPassword">
                Подтверждение пароля <span class="text-destructive">*</span>
              </Label>
              <Input
                id="confirmPassword"
                v-model="form.confirmPassword"
                type="password"
                :class="cn(errors.confirmPassword && 'border-destructive')"
                @blur="validateField('confirmPassword')"
              />
              <p v-if="errors.confirmPassword" class="text-sm text-destructive">{{ errors.confirmPassword }}</p>
            </div>

            <!-- Submit button -->
            <Button type="submit" class="w-full" :disabled="isSubmitting">
              {{ isSubmitting ? 'Отправка...' : 'Принять приглашение' }}
            </Button>
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
    </template>
  </div>
</template>

<style scoped>
.success-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: hsl(var(--background));
  animation: fadeIn 0.3s ease-out;
  z-index: 50;
}

.success-circle {
  width: 80px;
  height: 80px;
  background: hsl(var(--success));
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: scaleIn 0.4s ease-out, rotate360 0.6s ease-out 0.4s forwards;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes scaleIn {
  from { transform: scale(0); }
  to { transform: scale(1); }
}

@keyframes rotate360 {
  from { transform: rotateY(0deg); }
  to { transform: rotateY(360deg); }
}
</style>
