<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import { MoreVertical } from 'lucide-vue-next'

export interface ActionItem {
  label: string
  icon?: any
  onClick: () => void
  variant?: 'default' | 'destructive'
  disabled?: boolean
  separator?: boolean
}

const props = withDefaults(
  defineProps<{
    actions: ActionItem[]
    align?: 'start' | 'center' | 'end'
    class?: string
  }>(),
  {
    align: 'end',
  }
)

const classes = computed(() => cn(props.class))
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" :class="classes">
        <MoreVertical class="h-4 w-4" />
        <span class="sr-only">Открыть меню</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent :align="align">
      <template v-for="(action, index) in actions" :key="index">
        <DropdownMenuSeparator v-if="action.separator && index > 0" />
        <DropdownMenuItem
          :disabled="action.disabled"
          :class="cn(action.variant === 'destructive' && 'text-destructive focus:text-destructive')"
          @click="action.onClick"
        >
          <component v-if="action.icon" :is="action.icon" class="mr-2 h-4 w-4" />
          {{ action.label }}
        </DropdownMenuItem>
      </template>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
