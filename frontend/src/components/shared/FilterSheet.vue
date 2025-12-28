<script setup lang="ts">
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Filter, X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    title?: string
    description?: string
    activeFiltersCount?: number
  }>(),
  {
    title: 'Фильтры',
    description: 'Настройте параметры поиска',
    activeFiltersCount: 0,
  }
)

const emit = defineEmits<{
  apply: []
  reset: []
  open: []
}>()

const open = defineModel<boolean>('open', { default: false })

function handleOpen() {
  open.value = true
  emit('open')
}

function handleApply() {
  emit('apply')
  open.value = false
}

function handleReset() {
  emit('reset')
}
</script>

<template>
  <Sheet v-model:open="open">
    <SheetTrigger as-child>
      <Button variant="outline" class="relative" @click="handleOpen">
        <Filter class="mr-2 h-4 w-4" />
        Фильтры
        <span
          v-if="activeFiltersCount > 0"
          class="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground"
        >
          {{ activeFiltersCount }}
        </span>
      </Button>
    </SheetTrigger>
    <SheetContent side="right" class="w-full sm:max-w-md flex flex-col">
      <SheetHeader class="flex-shrink-0">
        <SheetTitle>{{ title }}</SheetTitle>
        <SheetDescription v-if="description">{{ description }}</SheetDescription>
      </SheetHeader>
      <div class="mt-6 flex-1 overflow-y-auto space-y-6 pl-1 pr-3">
        <slot />
      </div>
      <SheetFooter class="mt-6 gap-2 flex-shrink-0">
        <Button variant="outline" @click="handleReset">
          <X class="mr-2 h-4 w-4" />
          Сбросить
        </Button>
        <Button @click="handleApply">
          Применить
        </Button>
      </SheetFooter>
    </SheetContent>
  </Sheet>
</template>
