<script setup lang="ts">
/**
 * Переиспользуемый bottom sheet для мобильных.
 * Используется как замена выпадающим спискам на мобильных устройствах.
 */

interface Props {
  label?: string
}

withDefaults(defineProps<Props>(), {
  label: '',
})

const open = defineModel<boolean>({ default: false })

function close() {
  open.value = false
}
</script>

<template>
  <Teleport to="body">
    <div
      class="fixed inset-0 z-[200] flex flex-col justify-end"
      :class="open ? 'pointer-events-auto' : 'pointer-events-none'"
    >
      <!-- Overlay -->
      <div
        class="absolute inset-0 bg-black/50 transition-opacity duration-300"
        :class="open ? 'opacity-100' : 'opacity-0 pointer-events-none'"
        @click="close"
      />

      <!-- Panel -->
      <div
        class="relative bg-white rounded-t-2xl flex flex-col transition-transform duration-300 pointer-events-auto"
        :class="open ? 'translate-y-0' : 'translate-y-full'"
        style="max-height: 80dvh"
      >
        <!-- Хедер -->
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-100 flex-shrink-0">
          <span class="text-sm font-medium text-gray-700">{{ label }}</span>
          <button
            type="button"
            class="p-1 rounded-full text-gray-400 hover:text-gray-600 active:text-gray-800 transition-colors"
            @click="close"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path
                fill-rule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clip-rule="evenodd"
              />
            </svg>
          </button>
        </div>

        <!-- Контент -->
        <div class="flex flex-col overflow-hidden flex-1">
          <slot />
        </div>
      </div>
    </div>
  </Teleport>
</template>
