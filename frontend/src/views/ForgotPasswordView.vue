<script setup lang="ts">
import { ref } from 'vue'
import { authApi } from '@/api/auth'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

// Icons
import { Mail, AlertCircle, CheckCircle2, ArrowLeft } from 'lucide-vue-next'

const email = ref('')
const error = ref('')
const isLoading = ref(false)
const isSubmitted = ref(false)

async function handleSubmit() {
  error.value = ''
  isLoading.value = true

  try {
    await authApi.forgotPassword({ email: email.value })
    isSubmitted.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка отправки запроса'
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background py-12 px-4">
    <Card class="w-full max-w-md">
      <CardHeader class="text-center">
        <CardTitle class="text-2xl">Восстановление пароля</CardTitle>
        <CardDescription>
          {{ isSubmitted ? 'Проверьте почту' : 'Введите email для получения ссылки' }}
        </CardDescription>
      </CardHeader>

      <CardContent>
        <!-- Success state -->
        <div v-if="isSubmitted" class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-success/50 bg-success/10 p-4 text-sm text-success"
          >
            <CheckCircle2 class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Письмо отправлено!</p>
              <p class="mt-1 text-success/80">
                Если аккаунт с email <strong>{{ email }}</strong> существует, вы получите письмо со ссылкой для сброса пароля.
              </p>
            </div>
          </div>

          <p class="text-sm text-muted-foreground text-center">
            Не получили письмо? Проверьте папку «Спам» или
            <button
              type="button"
              class="text-primary hover:underline"
              @click="isSubmitted = false"
            >
              попробуйте ещё раз
            </button>
          </p>
        </div>

        <!-- Form -->
        <template v-else>
          <!-- Error -->
          <div
            v-if="error"
            class="mb-6 flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
          >
            <AlertCircle class="h-4 w-4 shrink-0" />
            {{ error }}
          </div>

          <form @submit.prevent="handleSubmit" class="space-y-4">
            <div class="space-y-2">
              <Label for="email">Email</Label>
              <Input
                id="email"
                v-model="email"
                type="email"
                placeholder="your@email.com"
                required
                autocomplete="email"
              />
            </div>

            <Button type="submit" class="w-full" :disabled="isLoading">
              <Mail v-if="!isLoading" class="mr-2 h-4 w-4" />
              {{ isLoading ? 'Отправка...' : 'Отправить ссылку' }}
            </Button>
          </form>
        </template>
      </CardContent>

      <CardFooter class="justify-center">
        <router-link
          to="/login"
          class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-primary"
        >
          <ArrowLeft class="h-4 w-4" />
          Вернуться к входу
        </router-link>
      </CardFooter>
    </Card>
  </div>
</template>
