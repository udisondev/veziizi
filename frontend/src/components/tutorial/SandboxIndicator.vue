<script setup lang="ts">
/**
 * SandboxIndicator
 * Индикатор режима песочницы в нижнем левом углу
 */

import { useOnboardingStore } from '@/stores/onboarding'
import { storeToRefs } from 'pinia'
import { Button } from '@/components/ui/button'
import { X, GraduationCap, SkipForward } from 'lucide-vue-next'

const onboarding = useOnboardingStore()
const { isSandboxMode, currentStep, scenarioSteps, currentStepIndex } = storeToRefs(onboarding)

function handleExit() {
  onboarding.exitSandbox()
}

function handleSkipStep() {
  onboarding.skipStep()
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-all duration-300 ease-out"
      leave-active-class="transition-all duration-200 ease-in"
      enter-from-class="translate-y-full opacity-0"
      leave-to-class="translate-y-full opacity-0"
    >
      <div
        v-if="isSandboxMode && !currentStep?.showOffersTrainingButton"
        class="fixed bottom-4 left-4 z-[60] flex items-center gap-3 rounded-lg bg-amber-500 px-4 py-2 text-white shadow-lg"
      >
        <GraduationCap class="h-5 w-5" />

        <div class="flex flex-col">
          <span class="text-sm font-medium">Режим обучения</span>
          <span v-if="currentStep" class="text-xs opacity-90">
            Шаг {{ currentStepIndex + 1 }} из {{ scenarioSteps.length }}
          </span>
        </div>

        <div class="ml-2 flex items-center gap-1">
          <Button
            v-if="currentStep"
            variant="ghost"
            size="sm"
            class="h-7 px-2 text-white hover:bg-amber-600"
            title="Пропустить шаг"
            @click="handleSkipStep"
          >
            <SkipForward class="h-4 w-4" />
          </Button>

          <Button
            variant="ghost"
            size="sm"
            class="h-7 px-2 text-white hover:bg-amber-600"
            title="Выйти из обучения"
            @click="handleExit"
          >
            <X class="h-4 w-4" />
          </Button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
