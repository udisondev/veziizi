<script setup lang="ts">
import { computed } from 'vue'
import { Primitive, type PrimitiveProps } from 'reka-ui'
import { cn } from '@/lib/utils'
import { buttonVariants, type ButtonVariants } from '.'

interface Props extends PrimitiveProps {
  variant?: ButtonVariants['variant']
  size?: ButtonVariants['size']
  class?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  as: 'button',
  variant: 'default',
  size: 'default',
})

const classes = computed(() =>
  cn(buttonVariants({ variant: props.variant, size: props.size }), props.class)
)
</script>

<template>
  <Primitive :as="as" :as-child="asChild" :class="classes" :disabled="disabled">
    <slot />
  </Primitive>
</template>
