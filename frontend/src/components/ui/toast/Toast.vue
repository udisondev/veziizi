<script setup lang="ts">
import { computed } from 'vue'
import { ToastRoot, type ToastRootEmits, type ToastRootProps, useForwardPropsEmits } from 'reka-ui'
import { type ToastVariants, toastVariants } from '.'
import { cn } from '@/lib/utils'

const props = defineProps<ToastRootProps & { class?: string; variant?: ToastVariants['variant'] }>()

const emits = defineEmits<ToastRootEmits>()

const delegatedProps = computed(() => {
  const { class: _, variant: _variant, ...delegated } = props
  return delegated
})

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <ToastRoot
    v-bind="forwarded"
    :class="cn(toastVariants({ variant }), props.class)"
    @update:open="emits('update:open', $event)"
  >
    <slot />
  </ToastRoot>
</template>
