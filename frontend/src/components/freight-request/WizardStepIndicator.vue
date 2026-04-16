<script setup lang="ts">
import { Check } from 'lucide-vue-next'

interface Props {
  steps: string[]
  currentStep: number
}

interface Emits {
  (e: 'goTo', step: number): void
}

defineProps<Props>()
defineEmits<Emits>()
</script>

<template>
  <div class="flex items-start">
    <template v-for="(title, index) in steps" :key="index">
      <!-- Connector between steps -->
      <div
        v-if="index > 0"
        class="flex-1 h-0.5 mt-4 transition-colors duration-300 shrink"
        :class="index < currentStep ? 'bg-primary' : 'bg-border'"
      />

      <!-- Step: circle + label -->
      <div class="flex flex-col items-center gap-2 w-16 shrink-0">
        <button
          type="button"
          :disabled="index + 1 > currentStep"
          class="flex items-center justify-center w-8 h-8 rounded-full font-semibold text-sm transition-all duration-300"
          :class="[
            index + 1 === currentStep
              ? 'bg-primary text-primary-foreground ring-4 ring-primary/20'
              : index + 1 < currentStep
                ? 'bg-primary text-primary-foreground cursor-pointer hover:bg-primary/80'
                : 'bg-background border-2 border-border text-muted-foreground cursor-not-allowed',
          ]"
          @click="index + 1 < currentStep && $emit('goTo', index + 1)"
        >
          <Check v-if="index + 1 < currentStep" class="h-4 w-4" />
          <span v-else>{{ index + 1 }}</span>
        </button>

        <span
          class="text-xs text-center leading-tight transition-colors duration-300 whitespace-nowrap"
          :class="[
            index + 1 === currentStep
              ? 'text-primary font-semibold'
              : index + 1 < currentStep
                ? 'text-primary/70 font-medium'
                : 'text-muted-foreground',
          ]"
        >
          {{ title }}
        </span>
      </div>
    </template>
  </div>
</template>
