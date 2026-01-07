<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ChevronDown } from 'lucide-vue-next'
import type { Component } from 'vue'

export interface TabItem {
  value: string
  label: string
  icon?: Component
  badge?: number | string
  separator?: boolean
}

const props = defineProps<{
  items: TabItem[]
  modelValue: string
  triggerTutorialId?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = defineModel<boolean>('open', { default: false })

const activeItem = computed(() =>
  props.items.find(item => item.value === props.modelValue)
)

function selectTab(value: string) {
  emit('update:modelValue', value)
}
</script>

<template>
  <DropdownMenu v-model:open="open">
    <DropdownMenuTrigger as-child>
      <Button variant="outline" class="w-full sm:w-auto justify-between gap-2" :data-tutorial="triggerTutorialId">
        <span class="flex items-center">
          <component
            v-if="activeItem?.icon"
            :is="activeItem.icon"
            class="mr-2 h-4 w-4"
          />
          {{ activeItem?.label }}
          <template v-if="activeItem?.badge">
            ({{ activeItem.badge }})
          </template>
        </span>
        <ChevronDown
          class="h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-200"
          :class="{ 'rotate-180': open }"
        />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="start" class="w-56 z-[80]">
      <template v-for="(item, index) in items" :key="item.value">
        <DropdownMenuSeparator v-if="item.separator && index > 0" />
        <DropdownMenuItem @click="selectTab(item.value)">
          <component
            v-if="item.icon"
            :is="item.icon"
            class="mr-2 h-4 w-4"
          />
          {{ item.label }}
          <Badge v-if="item.badge" variant="secondary" class="ml-auto">
            {{ item.badge }}
          </Badge>
        </DropdownMenuItem>
      </template>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
