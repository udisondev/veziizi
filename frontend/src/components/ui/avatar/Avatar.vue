<script setup lang="ts">
import { computed } from 'vue'
import { AvatarRoot, type AvatarRootProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<AvatarRootProps & { class?: string }>()

const classes = computed(() =>
  cn('relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full', props.class)
)
</script>

<template>
  <AvatarRoot :class="classes">
    <slot />
  </AvatarRoot>
</template>
