<script setup lang="ts">
import { computed } from 'vue'
import {
  SelectContent,
  type SelectContentEmits,
  type SelectContentProps,
  SelectPortal,
  SelectViewport,
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<SelectContentProps & { class?: string }>(),
  {
    position: 'popper',
  }
)

const emits = defineEmits<SelectContentEmits>()

const classes = computed(() =>
  cn(
    'relative z-50 max-h-96 min-w-32 overflow-hidden rounded-md border bg-popover text-popover-foreground shadow-md data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
    props.position === 'popper' &&
      'data-[side=bottom]:translate-y-1 data-[side=left]:-translate-x-1 data-[side=right]:translate-x-1 data-[side=top]:-translate-y-1',
    props.class
  )
)

const viewportClasses = computed(() =>
  cn(
    'p-1',
    props.position === 'popper' &&
      'h-[var(--reka-select-trigger-height)] w-full min-w-[var(--reka-select-trigger-width)]'
  )
)
</script>

<template>
  <SelectPortal>
    <SelectContent
      :class="classes"
      :position="props.position"
      @close-auto-focus="emits('closeAutoFocus', $event)"
    >
      <SelectViewport :class="viewportClasses">
        <slot />
      </SelectViewport>
    </SelectContent>
  </SelectPortal>
</template>
