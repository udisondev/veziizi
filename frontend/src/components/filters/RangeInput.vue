<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

interface Props {
  minValue?: number
  maxValue?: number
  label?: string
  minPlaceholder?: string
  maxPlaceholder?: string
  min?: number
  step?: number | string
}

interface Emits {
  (e: 'update:minValue', value: number | undefined): void
  (e: 'update:maxValue', value: number | undefined): void
}

withDefaults(defineProps<Props>(), {
  minPlaceholder: 'от',
  maxPlaceholder: 'до',
  min: 0,
  step: 1,
})
defineEmits<Emits>()
</script>

<template>
  <div>
    <Label v-if="label" class="text-sm">{{ label }}</Label>
    <div class="flex items-center gap-3 mt-1">
      <Input
        type="number"
        :model-value="minValue"
        :placeholder="minPlaceholder"
        :min="min"
        :step="step"
        class="flex-1"
        @update:model-value="(v: string | number) => $emit('update:minValue', v === '' ? undefined : Number(v))"
      />
      <span class="text-muted-foreground">—</span>
      <Input
        type="number"
        :model-value="maxValue"
        :placeholder="maxPlaceholder"
        :min="min"
        :step="step"
        class="flex-1"
        @update:model-value="(v: string | number) => $emit('update:maxValue', v === '' ? undefined : Number(v))"
      />
    </div>
  </div>
</template>
