<script setup lang="ts">
/**
 * FirstLoginHint
 * Подсветка кнопки "?" при первом входе пользователя
 * Показывает tooltip с информацией о разделе помощи
 */

import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useOnboardingStore } from '@/stores/onboarding'
import { useAuthStore } from '@/stores/auth'
import { storeToRefs } from 'pinia'
import { Button } from '@/components/ui/button'
import { X, GraduationCap, HelpCircle } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const onboarding = useOnboardingStore()
const auth = useAuthStore()
const { hasSeenHelpHint, isSandboxMode } = storeToRefs(onboarding)
const { isAuthenticated } = storeToRefs(auth)

// Показываем если: авторизован, не видел hint, не в sandbox, на главной
const isVisible = computed(() =>
  isAuthenticated.value &&
  !hasSeenHelpHint.value &&
  !isSandboxMode.value &&
  route.path === '/'
)

// Позиция "дырки" вокруг кнопки
const holeRect = ref({ top: 0, left: 0, width: 0, height: 0 })
const tooltipPosition = ref({ top: 0, left: 0 })
const isReady = ref(false)

// Отступы
const padding = 8
const borderRadius = 24 // для круглой кнопки

// Размеры экрана
const screenWidth = ref(window.innerWidth)
const screenHeight = ref(window.innerHeight)

// Стили для 4 div вокруг "дырки"
const topStyle = computed(() => ({
  top: '0',
  left: '0',
  width: '100%',
  height: `${Math.max(0, holeRect.value.top)}px`,
}))

const bottomStyle = computed(() => ({
  top: `${holeRect.value.top + holeRect.value.height}px`,
  left: '0',
  width: '100%',
  height: `${Math.max(0, screenHeight.value - holeRect.value.top - holeRect.value.height)}px`,
}))

const leftStyle = computed(() => ({
  top: `${holeRect.value.top}px`,
  left: '0',
  width: `${Math.max(0, holeRect.value.left)}px`,
  height: `${holeRect.value.height}px`,
}))

const rightStyle = computed(() => ({
  top: `${holeRect.value.top}px`,
  left: `${holeRect.value.left + holeRect.value.width}px`,
  width: `${Math.max(0, screenWidth.value - holeRect.value.left - holeRect.value.width)}px`,
  height: `${holeRect.value.height}px`,
}))

function updatePosition() {
  screenWidth.value = window.innerWidth
  screenHeight.value = window.innerHeight

  const target = document.querySelector('[data-tutorial="help-btn"]')
  if (!target) {
    isReady.value = false
    return
  }

  const rect = target.getBoundingClientRect()

  holeRect.value = {
    top: rect.top - padding,
    left: rect.left - padding,
    width: rect.width + padding * 2,
    height: rect.height + padding * 2,
  }

  // Tooltip слева от кнопки (с отступом)
  tooltipPosition.value = {
    top: rect.top + rect.height / 2,
    left: rect.left - 16,
  }

  isReady.value = true
}

function dismiss() {
  onboarding.markHelpHintSeen()
}

function goToSupport() {
  onboarding.markHelpHintSeen()
  router.push('/support')
}

// Следим за видимостью
watch(isVisible, async (visible) => {
  if (visible) {
    await nextTick()
    setTimeout(updatePosition, 200)
  } else {
    isReady.value = false
  }
}, { immediate: true })

onMounted(() => {
  window.addEventListener('resize', updatePosition)
  window.addEventListener('scroll', updatePosition, true)
})

onUnmounted(() => {
  window.removeEventListener('resize', updatePosition)
  window.removeEventListener('scroll', updatePosition, true)
})

function blockClick(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-300"
      leave-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div v-if="isVisible && isReady" class="fixed inset-0 z-[55]">
        <!-- 4 div вокруг "дырки" - они блокируют клики -->
        <div
          class="absolute bg-black/50 pointer-events-auto"
          :style="topStyle"
          @click.capture="blockClick"
        />
        <div
          class="absolute bg-black/50 pointer-events-auto"
          :style="bottomStyle"
          @click.capture="blockClick"
        />
        <div
          class="absolute bg-black/50 pointer-events-auto"
          :style="leftStyle"
          @click.capture="blockClick"
        />
        <div
          class="absolute bg-black/50 pointer-events-auto"
          :style="rightStyle"
          @click.capture="blockClick"
        />

        <!-- Подсветка кнопки -->
        <div
          class="pointer-events-none absolute ring-2 ring-amber-400 ring-offset-2"
          :style="{
            top: `${holeRect.top}px`,
            left: `${holeRect.left}px`,
            width: `${holeRect.width}px`,
            height: `${holeRect.height}px`,
            borderRadius: `${borderRadius}px`,
          }"
        />

        <!-- Tooltip слева от кнопки -->
        <div
          class="fixed z-[70] w-72 rounded-lg border bg-white p-4 shadow-xl pointer-events-auto"
          :style="{
            top: `${tooltipPosition.top}px`,
            left: `${tooltipPosition.left}px`,
            transform: 'translate(-100%, -50%)',
          }"
        >
          <!-- Стрелка вправо -->
          <div
            class="absolute top-1/2 right-0 -translate-y-1/2 translate-x-1/2 rotate-45 w-3 h-3 bg-white border-r border-t"
          />

          <!-- Кнопка закрытия -->
          <Button
            variant="ghost"
            size="icon"
            class="absolute top-1 right-1 h-6 w-6"
            @click="dismiss"
          >
            <X class="h-4 w-4" />
          </Button>

          <!-- Содержимое -->
          <div class="flex items-start gap-3 pr-6">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-amber-100">
              <GraduationCap class="h-5 w-5 text-amber-600" />
            </div>
            <div>
              <h3 class="font-medium text-foreground mb-1">Нужна помощь?</h3>
              <p class="text-sm text-muted-foreground mb-3">
                Здесь вы найдёте обучающие курсы и сможете обратиться в поддержку
              </p>
              <Button size="sm" @click="goToSupport">
                <HelpCircle class="mr-2 h-4 w-4" />
                Перейти
              </Button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
