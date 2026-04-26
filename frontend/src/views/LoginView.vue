<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAsyncAction } from '@/composables/useAsyncData'

// UI Components
import { Button } from '@/components/ui/button'
import { AppLink } from '@/components/ui/app-link'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

// Icons
import { LogIn, AlertCircle, CheckCircle2 } from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const email = ref('')
const password = ref('')
const showRegistrationSuccess = computed(() => route.query.registered === 'true')

const { execute: handleSubmit, isLoading, error } = useAsyncAction(async () => {
  await auth.login({ email: email.value, password: password.value })
  const redirect = (route.query.redirect as string) || '/'
  router.push(redirect)
})
</script>

<template>
  <div class="flex items-center justify-center bg-background py-12 px-4" style="min-height: calc(100vh - 3.5rem)">
    <Card class="w-full max-w-md">
      <CardHeader class="text-center">
        <CardTitle class="text-2xl">Вход в систему</CardTitle>
        <CardDescription>Войдите в свой аккаунт</CardDescription>
      </CardHeader>

      <CardContent>
        <!-- Registration success -->
        <div
          v-if="showRegistrationSuccess"
          class="mb-6 flex items-center gap-2 rounded-lg border border-success/50 bg-success/10 p-3 text-sm text-success"
        >
          <CheckCircle2 class="h-4 w-4 shrink-0" />
          Регистрация успешна! Войдите с указанными данными.
        </div>

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

          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <Label for="password">Пароль</Label>
              <router-link
                to="/forgot-password"
                class="text-sm text-muted-foreground hover:text-primary"
              >
                Забыли пароль?
              </router-link>
            </div>
            <Input
              id="password"
              v-model="password"
              type="password"
              placeholder="Введите пароль"
              required
              autocomplete="current-password"
            />
          </div>

          <Button type="submit" class="w-full" :disabled="isLoading">
            <LogIn v-if="!isLoading" class="mr-2 h-4 w-4" />
            {{ isLoading ? 'Вход...' : 'Войти' }}
          </Button>
        </form>
      </CardContent>

      <CardFooter class="justify-center">
        <p class="text-sm text-muted-foreground">
          Нет аккаунта?
          <AppLink to="/register">Регистрация организации</AppLink>
        </p>
      </CardFooter>
    </Card>
  </div>
</template>
