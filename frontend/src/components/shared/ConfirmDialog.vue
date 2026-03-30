<script setup lang="ts">
import { computed } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { AlertTriangle, Info, AlertCircle } from 'lucide-vue-next'

type DialogVariant = 'default' | 'destructive' | 'warning'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    confirmLabel?: string
    cancelLabel?: string
    variant?: DialogVariant
    loading?: boolean
  }>(),
  {
    confirmLabel: 'Подтвердить',
    cancelLabel: 'Отмена',
    variant: 'default',
    loading: false,
  }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: []
  cancel: []
}>()

const iconComponent = computed(() => {
  switch (props.variant) {
    case 'destructive':
      return AlertCircle
    case 'warning':
      return AlertTriangle
    default:
      return Info
  }
})

const iconClass = computed(() => {
  switch (props.variant) {
    case 'destructive':
      return 'text-destructive'
    case 'warning':
      return 'text-warning'
    default:
      return 'text-primary'
  }
})

const confirmVariant = computed(() => {
  return props.variant === 'destructive' ? 'destructive' : 'default'
})

function handleCancel() {
  emit('cancel')
  emit('update:open', false)
}

function handleConfirm() {
  emit('confirm')
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent>
      <DialogHeader>
        <div class="flex items-start gap-4">
          <div
            class="rounded-full p-2"
            :class="{
              'bg-destructive/10': variant === 'destructive',
              'bg-warning/10': variant === 'warning',
              'bg-primary/10': variant === 'default',
            }"
          >
            <component :is="iconComponent" class="h-5 w-5" :class="iconClass" />
          </div>
          <div class="flex-1">
            <DialogTitle>{{ title }}</DialogTitle>
            <DialogDescription :class="description ? '' : 'sr-only'">
              {{ description || title }}
            </DialogDescription>
          </div>
        </div>
      </DialogHeader>
      <DialogFooter>
        <Button variant="outline" :disabled="loading" @click="handleCancel">
          {{ cancelLabel }}
        </Button>
        <Button :variant="confirmVariant" :disabled="loading" @click="handleConfirm">
          {{ confirmLabel }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
