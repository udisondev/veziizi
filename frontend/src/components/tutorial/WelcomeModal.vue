<script setup lang="ts">
/**
 * WelcomeModal
 * Модалка первого входа с выбором сценария обучения
 */

import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useOnboardingStore } from '@/stores/onboarding'
import { useAuthStore } from '@/stores/auth'
import { storeToRefs } from 'pinia'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import {
  GraduationCap,
  Package,
  Truck,
  Users,
  Sparkles,
} from 'lucide-vue-next'
import type { ScenarioType } from '@/types/tutorial'

const route = useRoute()
const onboarding = useOnboardingStore()
const auth = useAuthStore()
const { hasSeenWelcome, isSandboxMode } = storeToRefs(onboarding)
const { isAuthenticated, role } = storeToRefs(auth)

// Показываем модалку если:
// 1. Пользователь авторизован
// 2. Не видел welcome ранее
// 3. Не в sandbox режиме (чтобы не показывать повторно)
// 4. На главной странице (чтобы не показывать на логине)
const isOpen = computed(() =>
  isAuthenticated.value &&
  !hasSeenWelcome.value &&
  !isSandboxMode.value &&
  route.path === '/'
)

// Проверяем роль для показа admin сценария
const canShowAdmin = computed(() =>
  role.value === 'owner' || role.value === 'administrator'
)

interface ScenarioOption {
  type: ScenarioType
  title: string
  description: string
  icon: typeof Package
  color: string
  show: boolean
}

const scenarios = computed<ScenarioOption[]>(() => [
  {
    type: 'customer_flow',
    title: 'Создание заявки',
    description: 'Научитесь создавать заявки на перевозку',
    icon: Package,
    color: 'bg-blue-100 text-blue-600',
    show: true,
  },
  {
    type: 'offers_receive_flow',
    title: 'Выбор предложения',
    description: 'Как выбирать предложения перевозчиков',
    icon: Truck,
    color: 'bg-green-100 text-green-600',
    show: true,
  },
  {
    type: 'admin_flow',
    title: 'Управление командой',
    description: 'Приглашение сотрудников и управление ролями',
    icon: Users,
    color: 'bg-purple-100 text-purple-600',
    show: canShowAdmin.value,
  },
])

const visibleScenarios = computed(() => scenarios.value.filter((s) => s.show))

async function selectScenario(type: ScenarioType) {
  await onboarding.enterSandbox(type)
}

function skipTutorial() {
  onboarding.markWelcomeSeen()
}
</script>

<template>
  <Dialog :open="isOpen">
    <DialogContent class="sm:max-w-[550px]" @interact-outside.prevent @escape-key-down.prevent>
      <DialogHeader class="text-center">
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-amber-100">
          <GraduationCap class="h-8 w-8 text-amber-600" />
        </div>
        <DialogTitle class="text-2xl">Добро пожаловать в Veziizi!</DialogTitle>
        <DialogDescription class="text-base">
          Пройдите интерактивное обучение, чтобы быстро освоить все возможности платформы
        </DialogDescription>
      </DialogHeader>

      <div class="my-6 grid gap-3">
        <Card
          v-for="scenario in visibleScenarios"
          :key="scenario.type"
          class="cursor-pointer transition-all hover:border-primary hover:shadow-md"
          @click="selectScenario(scenario.type)"
        >
          <CardHeader class="flex-row items-center gap-4 p-4">
            <div :class="['flex h-12 w-12 shrink-0 items-center justify-center rounded-lg', scenario.color]">
              <component :is="scenario.icon" class="h-6 w-6" />
            </div>
            <div class="flex-1">
              <CardTitle class="text-base">{{ scenario.title }}</CardTitle>
              <CardDescription class="mt-1">{{ scenario.description }}</CardDescription>
            </div>
          </CardHeader>
        </Card>
      </div>

      <DialogFooter class="flex-col gap-2 sm:flex-col">
        <p class="text-center text-xs text-muted-foreground">
          <Sparkles class="mr-1 inline h-3 w-3" />
          Все данные в обучении — демонстрационные и не влияют на реальную работу
        </p>
        <Button variant="ghost" size="sm" class="w-full" @click="skipTutorial">
          Пропустить обучение
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
