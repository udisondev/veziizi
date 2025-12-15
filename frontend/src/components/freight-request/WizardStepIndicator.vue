<script setup lang="ts">
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
  <div class="flex items-center justify-center space-x-2 mb-8">
    <template v-for="(title, index) in steps" :key="index">
      <button
        type="button"
        :class="[
          'flex items-center justify-center w-10 h-10 rounded-full font-medium transition-colors text-sm',
          index + 1 === currentStep
            ? 'bg-blue-600 text-white'
            : index + 1 < currentStep
              ? 'bg-green-500 text-white cursor-pointer hover:bg-green-600'
              : 'bg-gray-200 text-gray-500 cursor-not-allowed',
        ]"
        :disabled="index + 1 > currentStep"
        :title="title"
        @click="index + 1 < currentStep && $emit('goTo', index + 1)"
      >
        <span v-if="index + 1 < currentStep">&#10003;</span>
        <span v-else>{{ index + 1 }}</span>
      </button>

      <div
        v-if="index < steps.length - 1"
        :class="[
          'w-8 h-1 rounded',
          index + 1 < currentStep ? 'bg-green-500' : 'bg-gray-200',
        ]"
      />
    </template>
  </div>

  <div class="text-center mb-6">
    <h3 class="text-lg font-medium text-gray-900">
      {{ steps[currentStep - 1] }}
    </h3>
    <p class="text-sm text-gray-500">
      Шаг {{ currentStep }} из {{ steps.length }}
    </p>
  </div>
</template>
