<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { notificationsApi } from '@/api/notifications'
import { ApiError } from '@/api/errors'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'

// Icons
import { CheckCircle2, XCircle, Clock, Mail } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()

const token = computed(() => route.query.token as string | undefined)

type VerifyState = 'loading' | 'success' | 'invalid' | 'rate_limit' | 'no_token'

const state = ref<VerifyState>('loading')

onMounted(async () => {
  if (!token.value) {
    state.value = 'no_token'
    return
  }

  try {
    await notificationsApi.verifyEmail(token.value)
    state.value = 'success'
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 429) {
        state.value = 'rate_limit'
      } else {
        state.value = 'invalid'
      }
    } else {
      state.value = 'invalid'
    }
  }
})

function goToSettings() {
  router.push('/notifications/settings')
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
          <template v-if="state === 'loading'">Проверка...</template>
          <template v-else-if="state === 'success'">Email подтверждён</template>
          <template v-else-if="state === 'invalid'">Ошибка</template>
          <template v-else-if="state === 'rate_limit'">Слишком много попыток</template>
          <template v-else>Ошибка</template>
        </CardTitle>
        <CardDescription>
          <template v-if="state === 'loading'">Подождите</template>
          <template v-else-if="state === 'success'">Теперь вы будете получать уведомления</template>
          <template v-else-if="state === 'invalid'">Ссылка недействительна</template>
          <template v-else-if="state === 'rate_limit'">Попробуйте позже</template>
          <template v-else>Отсутствует токен</template>
        </CardDescription>
      </CardHeader>

      <CardContent>
        <!-- Loading -->
        <div v-if="state === 'loading'" class="flex flex-col items-center gap-4 py-8">
          <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <p class="text-sm text-muted-foreground">Подтверждение email...</p>
        </div>

        <!-- Success -->
        <div v-else-if="state === 'success'" class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-success/50 bg-success/10 p-4 text-sm text-success"
          >
            <CheckCircle2 class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Email успешно подтверждён!</p>
              <p class="mt-1 text-success/80">
                Теперь вы будете получать уведомления на указанный адрес.
              </p>
            </div>
          </div>

          <Button class="w-full" @click="goToSettings">
            <Mail class="mr-2 h-4 w-4" />
            Перейти к настройкам
          </Button>
        </div>

        <!-- Invalid token -->
        <div v-else-if="state === 'invalid'" class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive"
          >
            <XCircle class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Ссылка недействительна</p>
              <p class="mt-1 text-destructive/80">
                Ссылка для подтверждения email не найдена или уже была использована.
              </p>
            </div>
          </div>

          <Button class="w-full" variant="outline" @click="goToSettings">
            Перейти к настройкам
          </Button>
        </div>

        <!-- Rate limit -->
        <div v-else-if="state === 'rate_limit'" class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-warning/50 bg-warning/10 p-4 text-sm text-warning"
          >
            <Clock class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Слишком много попыток</p>
              <p class="mt-1 text-warning/80">
                Подождите некоторое время и попробуйте снова.
              </p>
            </div>
          </div>

          <Button class="w-full" variant="outline" @click="goToSettings">
            Перейти к настройкам
          </Button>
        </div>

        <!-- No token -->
        <div v-else class="space-y-4">
          <div
            class="flex items-center gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive"
          >
            <XCircle class="h-5 w-5 shrink-0" />
            <div>
              <p class="font-medium">Отсутствует токен</p>
              <p class="mt-1 text-destructive/80">
                Ссылка повреждена. Попробуйте запросить новое письмо.
              </p>
            </div>
          </div>

          <Button class="w-full" variant="outline" @click="goToSettings">
            Перейти к настройкам
          </Button>
        </div>
      </CardContent>

      <CardFooter class="justify-center">
        <button
          type="button"
          class="text-sm text-muted-foreground hover:text-primary"
          @click="goToLogin"
        >
          Вернуться к входу
        </button>
      </CardFooter>
    </Card>
  </div>
</template>
