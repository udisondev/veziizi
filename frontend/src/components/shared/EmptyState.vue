<script setup lang="ts">
import { computed, type Component } from 'vue'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { Inbox } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    title: string
    description?: string
    icon?: Component
    actionLabel?: string
    class?: string
  }>(),
  {
    icon: () => Inbox,
  }
)

const emit = defineEmits<{
  action: []
}>()

const classes = computed(() =>
  cn('flex flex-col items-center justify-center gap-4 py-12 text-center', props.class)
)
</script>

<template>
  <div :class="classes">
    <div class="rounded-full bg-muted p-4">
      <component :is="icon" class="h-8 w-8 text-muted-foreground" />
    </div>
    <div class="space-y-1">
      <h3 class="text-lg font-medium text-foreground">{{ title }}</h3>
      <p v-if="description" class="text-sm text-muted-foreground max-w-sm">
        {{ description }}
      </p>
    </div>
    <Button v-if="actionLabel" @click="emit('action')">
      {{ actionLabel }}
    </Button>
  </div>
</template>
