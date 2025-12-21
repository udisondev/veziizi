<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

// Icons
import { ShieldCheck, AlertCircle } from 'lucide-vue-next'

const router = useRouter()
const admin = useAdminStore()

const email = ref('')
const password = ref('')
const error = ref('')
const isLoading = ref(false)

async function handleSubmit() {
  error.value = ''
  isLoading.value = true

  try {
    await admin.login({ email: email.value, password: password.value })
    router.push('/admin')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка входа'
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-900 py-12 px-4">
    <Card class="w-full max-w-md bg-slate-800 border-slate-700">
      <CardHeader class="text-center">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-indigo-500/10 mb-4">
          <ShieldCheck class="h-6 w-6 text-indigo-400" />
        </div>
        <CardTitle class="text-2xl text-white">Панель администратора</CardTitle>
        <CardDescription class="text-slate-400">
          Вход для модераторов платформы
        </CardDescription>
      </CardHeader>

      <CardContent>
        <!-- Error -->
        <div
          v-if="error"
          class="mb-6 flex items-center gap-2 rounded-lg border border-red-500/50 bg-red-500/10 p-3 text-sm text-red-400"
        >
          <AlertCircle class="h-4 w-4 shrink-0" />
          {{ error }}
        </div>

        <form @submit.prevent="handleSubmit" class="space-y-4">
          <div class="space-y-2">
            <Label for="email" class="text-slate-200">Email</Label>
            <Input
              id="email"
              v-model="email"
              type="email"
              required
              placeholder="admin@example.com"
              autocomplete="email"
              class="bg-slate-700 border-slate-600 text-white placeholder:text-slate-500 focus:border-indigo-500 focus:ring-indigo-500"
            />
          </div>

          <div class="space-y-2">
            <Label for="password" class="text-slate-200">Пароль</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              required
              placeholder="Введите пароль"
              autocomplete="current-password"
              class="bg-slate-700 border-slate-600 text-white placeholder:text-slate-500 focus:border-indigo-500 focus:ring-indigo-500"
            />
          </div>

          <Button
            type="submit"
            class="w-full bg-indigo-600 hover:bg-indigo-700 text-white"
            :disabled="isLoading"
          >
            {{ isLoading ? 'Вход...' : 'Войти' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
