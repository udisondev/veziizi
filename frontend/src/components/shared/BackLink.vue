<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRouter, type RouteLocationRaw } from 'vue-router'
import { cn } from '@/lib/utils'
import { ArrowLeft } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    to: RouteLocationRaw
    label?: string
    class?: string
    useHistory?: boolean
    tutorialId?: string
  }>(),
  {
    label: 'Назад',
    useHistory: false,
  }
)

const router = useRouter()

const classes = computed(() =>
  cn(
    'inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer',
    props.class
  )
)

function goBack() {
  // Проверяем есть ли предыдущая страница в истории Vue Router
  const hasPreviousPage = window.history.state?.back !== null
  if (hasPreviousPage) {
    router.back()
  } else {
    router.push(props.to)
  }
}
</script>

<template>
  <a v-if="useHistory" :class="classes" :data-tutorial="tutorialId" @click="goBack">
    <ArrowLeft class="h-4 w-4" />
    {{ label }}
  </a>
  <RouterLink v-else :to="to" :class="classes" :data-tutorial="tutorialId">
    <ArrowLeft class="h-4 w-4" />
    {{ label }}
  </RouterLink>
</template>
