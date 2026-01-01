<script setup lang="ts">
import { AccordionRoot, type AccordionRootEmits, type AccordionRootProps } from 'radix-vue'

const props = defineProps<AccordionRootProps>()
const emits = defineEmits<AccordionRootEmits>()
</script>

<template>
  <AccordionRoot
    v-bind="props"
    @update:model-value="(v) => emits('update:modelValue', v)"
  >
    <slot />
  </AccordionRoot>
</template>
