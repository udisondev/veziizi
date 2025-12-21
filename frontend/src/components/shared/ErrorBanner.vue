<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { AlertCircle, RefreshCw } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    title?: string
    message?: string
    retryLabel?: string
    class?: string
  }>(),
  {
    title: 'Произошла ошибка',
    message: 'Не удалось загрузить данные. Попробуйте ещё раз.',
    retryLabel: 'Повторить',
  }
)

const emit = defineEmits<{
  retry: []
}>()

const classes = computed(() => cn('border-destructive/50 bg-destructive/5', props.class))
</script>

<template>
  <Card :class="classes">
    <CardContent class="flex items-center gap-4 py-4">
      <div class="rounded-full bg-destructive/10 p-2">
        <AlertCircle class="h-5 w-5 text-destructive" />
      </div>
      <div class="flex-1">
        <h4 class="text-sm font-medium text-destructive">{{ title }}</h4>
        <p class="text-sm text-muted-foreground">{{ message }}</p>
      </div>
      <Button variant="outline" size="sm" @click="emit('retry')">
        <RefreshCw class="mr-2 h-4 w-4" />
        {{ retryLabel }}
      </Button>
    </CardContent>
  </Card>
</template>
