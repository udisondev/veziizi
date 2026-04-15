<script setup lang="ts">
import {
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
  TooltipPortal,
  TooltipContent,
} from 'reka-ui'

defineProps<{
  text: string
  side?: 'top' | 'right' | 'bottom' | 'left'
  delayDuration?: number
}>()
</script>

<template>
  <TooltipProvider :delay-duration="delayDuration ?? 400">
    <TooltipRoot>
      <TooltipTrigger as-child>
        <slot />
      </TooltipTrigger>
      <TooltipPortal>
        <TooltipContent
          :side="side ?? 'bottom'"
          :side-offset="6"
          class="z-50 overflow-hidden rounded-md bg-popover px-3 py-1.5 text-sm text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95"
        >
          {{ text }}
        </TooltipContent>
      </TooltipPortal>
    </TooltipRoot>
  </TooltipProvider>
</template>
