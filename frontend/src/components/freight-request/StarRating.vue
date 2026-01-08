<script setup lang="ts">
import { ref, computed } from 'vue'
import { Star } from 'lucide-vue-next'

interface Props {
  modelValue: number
  readonly?: boolean
  size?: 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  readonly: false,
  size: 'md',
})

const emit = defineEmits<{
  'update:modelValue': [value: number]
  'click': [value: number]
}>()

const sizeClasses = {
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-6 w-6',
}

const stars = computed(() => [1, 2, 3, 4, 5])

// Hover state
const hoverRating = ref(0)

function onMouseEnter(star: number) {
  if (!props.readonly) {
    hoverRating.value = star
  }
}

function onMouseLeave() {
  hoverRating.value = 0
}

function onClick(star: number) {
  if (!props.readonly) {
    emit('update:modelValue', star)
    emit('click', star)
  }
}

// Display rating: hover takes priority, then actual value
const displayRating = computed(() => hoverRating.value || props.modelValue)
</script>

<template>
  <div class="flex gap-1" @mouseleave="onMouseLeave">
    <button
      v-for="star in stars"
      :key="star"
      type="button"
      :class="[
        'focus:outline-none transition-all duration-100',
        readonly ? 'cursor-default' : 'cursor-pointer hover:scale-110',
      ]"
      :disabled="readonly"
      @mouseenter="onMouseEnter(star)"
      @click="onClick(star)"
    >
      <Star
        :class="[
          sizeClasses[size],
          star <= displayRating ? 'text-yellow-400 fill-yellow-400' : 'text-gray-300',
        ]"
      />
    </button>
  </div>
</template>
