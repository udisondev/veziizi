<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authApi } from '@/api/auth'
import { ApiError } from '@/api/errors'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

// Icons
import { KeyRound, AlertCircle, CheckCircle2, Eye, EyeOff, XCircle, Clock } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()

const token = computed(() => route.params.token as string)

// Token validation state
const isValidating = ref(true)
const tokenError = ref<'not_found' | 'expired' | null>(null)

const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const showConfirmPassword = ref(false)
const error = ref('')
const isLoading = ref(false)
const isSuccess = ref(false)

const passwordsMatch = computed(() => password.value === confirmPassword.value)
const isValidPassword = computed(() => password.value.length >= 8)

onMounted(async () => {
  try {
    await authApi.validateResetToken(token.value)
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 404) {
        tokenError.value = 'not_found'
      } else if (e.status === 410) {
        tokenError.value = 'expired'
      } else {
        tokenError.value = 'not_found'
      }
    } else {
      tokenError.value = 'not_found'
    }
  } finally {
    isValidating.value = false
  }
})

async function handleSubmit() {
  error.value = ''

  if (!isValidPassword.value) {
    error.value = 'Пароль должен быть не менее 8 символов'
    return
  }

  if (!passwordsMatch.value) {
    error.value = 'Пароли не совпадают'
    return
  }

  isLoading.value = true

  try {
    await authApi.resetPassword({
      token: token.value,
      password: password.value,
    })
    isSuccess.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка сброса пароля'
  } finally {
    isLoading.value = false
  }
}

function goToLogin() {
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background py-12 px-4">
    <Card class="w-full max-w-md">
      <CardHeader class="text-center">
        <CardTitle class="text-2xl">
          <template v-if="isValidating">Проверка...</template>
          <template v-else-if="tokenError">Ошибка</template>
          <template v-else-if="isSuccess">Пароль изменён</template>
          <template v-else>Новый пароль</template>
        </CardTitle>
        <CardDescription>
          <template v-if="isValidating">Подождите</template>
          <template v-else-if="tokenError">Ссылка недействительна</template>
          <template v-else-if="isSuccess">Теперь вы можете войти</template>
          <template v-else>Придумайте новый пароль</template>
        </CardDescription>
      </CardHeader>

      <CardContent>
        <!-- Validating token -->
        <div v-if="isValidating" class="flex flex-col items-center gap-4 py-8">
          <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <p class="text-sm text-muted-foreground">Проверка ссылки...</p>
        </div>

        <!-- Token not found -->
        <div v-else-if="tokenError === 'not_found'" class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive"
          >
            <XCircle class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Ссылка недействительна</p>
              <p class="mt-1 text-destructive/80">
                Ссылка для сброса пароля не найдена или уже была использована.
              </p>
            </div>
          </div>

          <Button class="w-full" variant="outline" @click="router.push('/forgot-password')">
            Запросить новую ссылку
          </Button>
        </div>

        <!-- Token expired -->
        <div v-else-if="tokenError === 'expired'" class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-warning/50 bg-warning/10 p-4 text-sm text-warning"
          >
            <Clock class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Ссылка устарела</p>
              <p class="mt-1 text-warning/80">
                Срок действия ссылки истёк. Запросите новую ссылку для сброса пароля.
              </p>
            </div>
          </div>

          <Button class="w-full" variant="outline" @click="router.push('/forgot-password')">
            Запросить новую ссылку
          </Button>
        </div>

        <!-- Success state -->
        <div v-else-if="isSuccess" class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-success/50 bg-success/10 p-4 text-sm text-success"
          >
            <CheckCircle2 class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Пароль успешно изменён!</p>
              <p class="mt-1 text-success/80">
                Используйте новый пароль для входа в систему.
              </p>
            </div>
          </div>

          <Button class="w-full" @click="goToLogin">
            Перейти к входу
          </Button>
        </div>

        <!-- Form (token is valid) -->
        <template v-else-if="!tokenError">
          <!-- Error -->
          <div
            v-if="error"
            class="mb-6 flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
          >
            <AlertCircle class="h-4 w-4 shrink-0" />
            {{ error }}
          </div>

          <form @submit.prevent="handleSubmit" class="space-y-4">
            <!-- Password -->
            <div class="space-y-2">
              <Label for="password">Новый пароль</Label>
              <div class="relative">
                <Input
                  id="password"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  placeholder="Минимум 8 символов"
                  required
                  autocomplete="new-password"
                  class="pr-10"
                />
                <button
                  type="button"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  @click="showPassword = !showPassword"
                >
                  <Eye v-if="!showPassword" class="h-4 w-4" />
                  <EyeOff v-else class="h-4 w-4" />
                </button>
              </div>
              <p
                v-if="password && !isValidPassword"
                class="text-xs text-destructive"
              >
                Пароль должен быть не менее 8 символов
              </p>
            </div>

            <!-- Confirm Password -->
            <div class="space-y-2">
              <Label for="confirmPassword">Подтвердите пароль</Label>
              <div class="relative">
                <Input
                  id="confirmPassword"
                  v-model="confirmPassword"
                  :type="showConfirmPassword ? 'text' : 'password'"
                  placeholder="Повторите пароль"
                  required
                  autocomplete="new-password"
                  class="pr-10"
                />
                <button
                  type="button"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  @click="showConfirmPassword = !showConfirmPassword"
                >
                  <Eye v-if="!showConfirmPassword" class="h-4 w-4" />
                  <EyeOff v-else class="h-4 w-4" />
                </button>
              </div>
              <p
                v-if="confirmPassword && !passwordsMatch"
                class="text-xs text-destructive"
              >
                Пароли не совпадают
              </p>
            </div>

            <Button
              type="submit"
              class="w-full"
              :disabled="isLoading || !isValidPassword || !passwordsMatch"
            >
              <KeyRound v-if="!isLoading" class="mr-2 h-4 w-4" />
              {{ isLoading ? 'Сохранение...' : 'Сохранить пароль' }}
            </Button>
          </form>
        </template>
      </CardContent>

      <CardFooter v-if="!isSuccess && !tokenError && !isValidating" class="justify-center">
        <router-link
          to="/login"
          class="text-sm text-muted-foreground hover:text-primary"
        >
          Вернуться к входу
        </router-link>
      </CardFooter>
    </Card>
  </div>
</template>
