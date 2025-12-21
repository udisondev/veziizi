<script setup lang="ts">
import { useToast } from './use-toast'
import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
} from '.'

const { toasts } = useToast()
</script>

<template>
  <ToastProvider>
    <Toast
      v-for="toast in toasts"
      :key="toast.id"
      :variant="toast.variant"
      :open="toast.open"
      @update:open="toast.onOpenChange"
    >
      <div class="grid gap-1">
        <ToastTitle v-if="toast.title">
          {{ toast.title }}
        </ToastTitle>
        <ToastDescription v-if="toast.description">
          {{ toast.description }}
        </ToastDescription>
      </div>
      <component :is="toast.action" v-if="toast.action" />
      <ToastClose />
    </Toast>
    <ToastViewport />
  </ToastProvider>
</template>
