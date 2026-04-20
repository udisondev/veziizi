<script setup lang="ts">
import { Label } from '@/components/ui/label'

defineProps<{
  label?: string
  required?: boolean
  error?: string | null
  hint?: string
}>()
</script>

<template>
  <div>
    <Label v-if="label">
      {{ label }}
      <span v-if="required" class="text-destructive">*</span>
    </Label>
    <slot />
    <p v-if="error" class="mt-1 text-sm text-destructive">{{ error }}</p>
    <p v-else-if="hint" class="mt-1 text-sm text-muted-foreground">{{ hint }}</p>
  </div>
</template>
