<script setup lang="ts">
import { computed } from 'vue'
import { AvatarImage, type AvatarImageProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<AvatarImageProps & { class?: string }>()

const classes = computed(() => cn('aspect-square h-full w-full', props.class))
</script>

<template>
  <AvatarImage v-bind="props" :class="classes" />
</template>
