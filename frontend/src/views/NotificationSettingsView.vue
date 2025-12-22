<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useNotificationsStore } from '@/stores/notifications'
import { notificationsApi } from '@/api/notifications'
import {
  categoryLabels,
  categoryDescriptions,
  allCategories,
  type NotificationCategory,
  type TelegramLinkCodeResponse,
} from '@/types/notification'
import { useToast } from '@/components/ui/toast/use-toast'

// UI Components
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'

// Shared Components
import { PageHeader, LoadingSpinner, BackLink } from '@/components/shared'

// Icons
import { Bell, MessageCircle, Check, X, AlertCircle, Copy, ExternalLink } from 'lucide-vue-next'

const { toast } = useToast()
const notificationsStore = useNotificationsStore()

const isGeneratingCode = ref(false)
const isDisconnecting = ref(false)
const isSaving = ref(false)
const linkCode = ref<TelegramLinkCodeResponse | null>(null)
const countdown = ref(0)
let countdownInterval: ReturnType<typeof setInterval> | null = null

const preferences = computed(() => notificationsStore.preferences)
const isLoading = computed(() => notificationsStore.isLoadingPreferences)
const isTelegramConnected = computed(() => notificationsStore.isTelegramConnected)

async function toggleSetting(
  category: NotificationCategory,
  channel: 'in_app' | 'telegram',
  value: boolean
) {
  isSaving.value = true
  try {
    await notificationsStore.updateCategorySetting(category, channel, value)
    toast({
      title: 'Настройки сохранены',
    })
  } catch {
    toast({
      title: 'Ошибка',
      description: 'Не удалось сохранить настройки',
      variant: 'destructive',
    })
  } finally {
    isSaving.value = false
  }
}

async function generateLinkCode() {
  isGeneratingCode.value = true
  try {
    linkCode.value = await notificationsApi.generateLinkCode()
    startCountdown(linkCode.value.expires_in)
    toast({
      title: 'Код создан',
      description: 'Отправьте код боту в Telegram',
    })
  } catch {
    toast({
      title: 'Ошибка',
      description: 'Telegram не настроен на сервере',
      variant: 'destructive',
    })
  } finally {
    isGeneratingCode.value = false
  }
}

function startCountdown(seconds: number) {
  countdown.value = seconds
  if (countdownInterval) {
    clearInterval(countdownInterval)
  }
  countdownInterval = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      clearInterval(countdownInterval!)
      countdownInterval = null
      linkCode.value = null
    }
  }, 1000)
}

function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

function copyCode() {
  if (linkCode.value) {
    navigator.clipboard.writeText(linkCode.value.code)
    toast({
      title: 'Код скопирован',
    })
  }
}

function openBot() {
  if (linkCode.value) {
    window.open(linkCode.value.bot_url, '_blank')
  }
}

async function disconnectTelegram() {
  isDisconnecting.value = true
  try {
    await notificationsStore.disconnectTelegram()
    toast({
      title: 'Telegram отключён',
    })
  } catch {
    toast({
      title: 'Ошибка',
      description: 'Не удалось отключить Telegram',
      variant: 'destructive',
    })
  } finally {
    isDisconnecting.value = false
  }
}

onMounted(() => {
  notificationsStore.fetchPreferences()
})

onUnmounted(() => {
  if (countdownInterval) {
    clearInterval(countdownInterval)
  }
})
</script>

<template>
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <BackLink to="/notifications" label="К уведомлениям" class="mb-4" />

    <PageHeader title="Настройки уведомлений" class="mb-6" />

    <LoadingSpinner v-if="isLoading" text="Загрузка настроек..." />

    <template v-else-if="preferences">
      <!-- Telegram Integration -->
      <Card class="mb-6">
        <CardHeader>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900">
              <MessageCircle class="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <CardTitle class="text-lg">Telegram</CardTitle>
              <CardDescription>
                Получайте уведомления в Telegram мессенджер
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <!-- Подключён -->
          <div v-if="isTelegramConnected" class="space-y-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Check class="h-5 w-5 text-green-500" />
                <span class="font-medium">Подключён</span>
                <Badge v-if="preferences.telegram.username" variant="secondary">
                  @{{ preferences.telegram.username }}
                </Badge>
              </div>
              <Button
                variant="outline"
                size="sm"
                :disabled="isDisconnecting"
                @click="disconnectTelegram"
              >
                <X class="mr-2 h-4 w-4" />
                Отключить
              </Button>
            </div>
            <p class="text-sm text-muted-foreground">
              Вы будете получать уведомления в Telegram для включённых категорий
            </p>
          </div>

          <!-- Не подключён -->
          <div v-else class="space-y-4">
            <!-- Показываем код если есть -->
            <template v-if="linkCode">
              <div class="rounded-lg border bg-muted/50 p-4 space-y-3">
                <div class="flex items-center justify-between">
                  <span class="text-sm text-muted-foreground">Код привязки:</span>
                  <span class="text-sm text-muted-foreground">
                    Истекает через {{ formatTime(countdown) }}
                  </span>
                </div>

                <div class="flex items-center gap-2">
                  <code class="flex-1 text-center text-2xl font-mono font-bold tracking-widest bg-background rounded px-4 py-2">
                    {{ linkCode.code }}
                  </code>
                  <Button variant="outline" size="icon" @click="copyCode">
                    <Copy class="h-4 w-4" />
                  </Button>
                </div>

                <div class="flex flex-col gap-2">
                  <Button @click="openBot" class="w-full">
                    <ExternalLink class="mr-2 h-4 w-4" />
                    Открыть бота в Telegram
                  </Button>
                  <p class="text-xs text-center text-muted-foreground">
                    Нажмите кнопку или отправьте код боту @veziizi_bot
                  </p>
                </div>
              </div>
            </template>

            <!-- Кнопка генерации кода -->
            <template v-else>
              <div class="flex items-center gap-2 text-muted-foreground">
                <AlertCircle class="h-5 w-5" />
                <span>Telegram не подключён</span>
              </div>
              <Button
                :disabled="isGeneratingCode"
                @click="generateLinkCode"
              >
                <MessageCircle class="mr-2 h-4 w-4" />
                {{ isGeneratingCode ? 'Создание кода...' : 'Подключить Telegram' }}
              </Button>
            </template>
          </div>
        </CardContent>
      </Card>

      <!-- Notification Categories -->
      <Card>
        <CardHeader>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Bell class="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle class="text-lg">Категории уведомлений</CardTitle>
              <CardDescription>
                Выберите какие уведомления вы хотите получать
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-6">
          <div
            v-for="(category, index) in allCategories"
            :key="category"
          >
            <div class="flex items-center justify-between">
              <div>
                <p class="font-medium">{{ categoryLabels[category] }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ categoryDescriptions[category] }}
                </p>
              </div>

              <div class="flex items-center gap-4">
                <!-- In-app toggle -->
                <div class="flex items-center gap-2">
                  <Label class="text-sm text-muted-foreground">В приложении</Label>
                  <Switch
                    :checked="preferences.enabled_categories[category].in_app"
                    :disabled="isSaving"
                    @update:checked="(v: boolean) => toggleSetting(category, 'in_app', v)"
                  />
                </div>

                <!-- Telegram toggle (only if connected) -->
                <div v-if="isTelegramConnected" class="flex items-center gap-2">
                  <Label class="text-sm text-muted-foreground">Telegram</Label>
                  <Switch
                    :checked="preferences.enabled_categories[category].telegram"
                    :disabled="isSaving"
                    @update:checked="(v: boolean) => toggleSetting(category, 'telegram', v)"
                  />
                </div>
              </div>
            </div>

            <Separator v-if="index < allCategories.length - 1" class="mt-4" />
          </div>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
