<script setup lang="ts">
/**
 * FirstLoginHint
 * Подсветка кнопки "?" при первом входе пользователя
 * Показывает tooltip с информацией о разделе помощи
 */

import { ref, computed, watch, nextTick } from 'vue'
import { useWindowSize, useEventListener } from '@vueuse/core'
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
const { width: screenWidth, height: screenHeight } = useWindowSize()

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

  // Tooltip: всегда снизу от иконки
  const tooltipWidth = Math.min(screenWidth.value - 24, 384)
  const minLeft = 12 + tooltipWidth / 2
  const maxLeft = screenWidth.value - 12 - tooltipWidth / 2
  const idealLeft = rect.left + rect.width / 2

  tooltipPosition.value = {
    top: rect.bottom + 16,
    left: Math.max(minLeft, Math.min(idealLeft, maxLeft)),
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

useEventListener('resize', updatePosition)
useEventListener('scroll', updatePosition, { capture: true })

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

        <!-- Tooltip: всегда снизу от иконки -->
        <div
          class="fixed z-[70] rounded-lg border bg-white p-4 shadow-xl pointer-events-auto w-[calc(100vw-24px)] max-w-sm"
          :style="{
            top: `${tooltipPosition.top}px`,
            left: `${tooltipPosition.left}px`,
            transform: 'translate(-50%, 0)',
          }"
        >

          <!-- Кнопка закрытия -->
          <Button
            variant="ghost"
            size="icon"
            class="absolute top-1 right-1 h-9 w-9 sm:h-6 sm:w-6 [&_svg]:size-5 sm:[&_svg]:size-4"
            @click="dismiss"
          >
            <X />
          </Button>

          <!-- Содержимое -->
          <div class="flex items-start gap-3 pr-10 sm:pr-6">
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
