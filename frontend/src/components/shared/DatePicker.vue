<script setup lang="ts">
/**
 * Обёртка над vue-flatpickr-component.
 *
 * На мобильных (< 768px) перехватывает открытие flatpickr и показывает
 * календарь внутри bottom sheet вместо нативного popup.
 * На десктопе — прозрачная обёртка, всё поведение стандартное.
 *
 * Использование идентично <FlatPickr>:
 *   <DatePicker
 *     :model-value="value"
 *     :config="config"
 *     :label="'Выберите дату'"
 *     @on-close="handleChange"
 *   />
 */
import { ref, computed, watch } from 'vue'
import FlatPickr from 'vue-flatpickr-component'
import type { BaseOptions } from 'flatpickr/dist/types/options'
import type { Instance } from 'flatpickr/dist/types/instance'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BottomSheet from '@/components/shared/BottomSheet.vue'

defineOptions({ inheritAttrs: false })

interface Props {
  modelValue?: string | string[] | null
  config?: Partial<BaseOptions>
  placeholder?: string
  /** Заголовок bottom sheet на мобильных */
  label?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: null,
  config: () => ({}),
  label: 'Выберите дату',
})

const { breakpoint } = useBreakpoint()
const isMobileNow = computed(() => breakpoint.value === 'mobile')

const sheetOpen = ref(false)
const calendarContainer = ref<HTMLElement | null>(null)
let fpInstance: Instance | null = null

// Когда BottomSheet закрывается через X/overlay — синхронизируем с flatpickr
watch(sheetOpen, (open) => {
  if (!open) fpInstance?.close()
})


type HookFn = (dates: Date[], str: string, fp: Instance) => void
type Hook = BaseOptions['onReady']
function toArray(hook: Hook | undefined): HookFn[] {
  if (!hook) return []
  return Array.isArray(hook) ? (hook as HookFn[]) : [hook as HookFn]
}

const enhancedConfig = computed<Partial<BaseOptions>>(() => {
  const base = props.config ?? {}

  if (!isMobileNow.value) return base

  return {
    ...base,

    onReady: [
      ...toArray(base.onReady),
      (_dates: Date[], _str: string, fp: Instance) => {
        fpInstance = fp
        if (!calendarContainer.value) return

        calendarContainer.value.appendChild(fp.calendarContainer)

        fp.calendarContainer.style.position = 'static'
        fp.calendarContainer.style.top = 'auto'
        fp.calendarContainer.style.left = 'auto'
        fp.calendarContainer.style.boxShadow = 'none'
        fp.calendarContainer.style.border = 'none'

        // Отключаем встроенную анимацию flatpickr (fpFadeInDown),
        // которая сбрасывает transform и вызывает рывок scale
        fp.calendarContainer.classList.remove('animate')

        // Заменяем стрелки года на кнопки - и + по бокам
        const yearWrapper = fp.calendarContainer.querySelector<HTMLElement>('.cur-year')?.parentElement
        if (yearWrapper) {
          // Скрываем стандартные стрелки
          yearWrapper.querySelectorAll<HTMLElement>('.arrowUp, .arrowDown').forEach(el => {
            el.style.display = 'none'
          })

          const btnMinus = document.createElement('button')
          btnMinus.type = 'button'
          btnMinus.textContent = '−'
          btnMinus.className = 'fp-year-btn fp-year-btn--minus'
          btnMinus.addEventListener('click', () => fp.changeYear(fp.currentYear - 1))

          const btnPlus = document.createElement('button')
          btnPlus.type = 'button'
          btnPlus.textContent = '+'
          btnPlus.className = 'fp-year-btn fp-year-btn--plus'
          btnPlus.addEventListener('click', () => fp.changeYear(fp.currentYear + 1))

          yearWrapper.insertBefore(btnMinus, yearWrapper.firstChild)
          yearWrapper.appendChild(btnPlus)
        }

        fp._positionCalendar = () => {}
      },
    ],

    onOpen: [
      ...toArray(base.onOpen),
      () => { sheetOpen.value = true },
    ],

    onClose: [
      ...toArray(base.onClose),
      () => { sheetOpen.value = false },
    ],
  }
})

</script>

<template>
  <FlatPickr
    :model-value="modelValue"
    :config="enhancedConfig"
    :placeholder="placeholder"
    v-bind="$attrs"
  />

  <!--
    BottomSheet всегда в DOM — нужен чтобы ref="calendarContainer"
    был валиден при инициализации flatpickr. На десктопе sheet никогда
    не откроется, т.к. onOpen не ставит sheetOpen=true.
  -->
  <BottomSheet v-model="sheetOpen" :label="label">
    <div
      ref="calendarContainer"
      class="date-picker-sheet overflow-y-auto flex justify-center p-4"
      style="max-height: calc(80dvh - 56px)"
    />
  </BottomSheet>
</template>

<!-- Глобальные стили для flatpickr внутри bottom sheet -->
<!-- (не scoped — flatpickr рендерит DOM вне компонента) -->
<style>
/* Убираем стрелочку-треугольник */
.date-picker-sheet .flatpickr-calendar::before,
.date-picker-sheet .flatpickr-calendar::after {
  display: none !important;
}

/* Внутри sheet видимостью управляем мы сами (translate + opacity) */
.date-picker-sheet .flatpickr-calendar {
  display: inline-block;
  opacity: 1;
  visibility: visible;
  transform: scale(1.13);
  transform-origin: top center;
  margin-bottom: calc(308px * 0.13);
}

/* Фиксируем layout навигации — prev/next по умолчанию position:absolute */
.date-picker-sheet .flatpickr-months {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.date-picker-sheet .flatpickr-months .flatpickr-prev-month,
.date-picker-sheet .flatpickr-months .flatpickr-next-month {
  position: static;
  display: flex;
  align-items: center;
  padding: 4px 6px;
  flex-shrink: 0;
}

.date-picker-sheet .flatpickr-months .flatpickr-month {
  flex: 1;
  overflow: visible;
}

/* Выводим current-month из абсолютного позиционирования в нормальный flow */
.date-picker-sheet .flatpickr-current-month {
  position: static;
  width: auto;
  left: auto;
  padding: 0;
  height: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  font-size: 115%;
}

/* Враппер года: кнопки по бокам от числа */
.date-picker-sheet .cur-year {
  width: 4ch;
  text-align: center;
  padding: 0 !important;
}

.date-picker-sheet .numInputWrapper:has(.cur-year) {
  display: flex;
  align-items: center;
  gap: 4px;
  width: auto;
  height: auto;
}

.fp-year-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 1px solid #e5e7eb;
  background: #f3f4f6;
  font-size: 1.1rem;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #374151;
  flex-shrink: 0;
}

.fp-year-btn:active {
  background: #d1d5db;
}

.flatpickr-monthDropdown-months {
  appearance: none !important;
  -webkit-appearance: none !important;
  text-align: center;
  padding: 0 !important;
}


/* ─── Timepicker: увеличенный размер  ─── */

.flatpickr-calendar.noCalendar {
  width: auto;
  padding: 16px;
}

.flatpickr-time {
  height: auto !important;
  max-height: none !important;
}

.flatpickr-time .numInputWrapper {
  width: 100px !important;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.flatpickr-time input {
  font-size: 2.5rem;
  height: 120px;
  line-height: 120px;
  width: 100px;
}

.flatpickr-time .flatpickr-time-separator {
  font-size: 2.5rem;
  line-height: 120px;
  height: 120px;
}

/* Стрелки: крупные, всегда видимые */
.flatpickr-time .numInputWrapper span {
  width: 100%;
  right: 0;
  left: 0;
  height: 36px;
  opacity: 1;
  background: #f3f4f6;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #e5e7eb;
}

.flatpickr-time .numInputWrapper span:hover {
  background: #e5e7eb;
}

.flatpickr-time .numInputWrapper span:active {
  background: #d1d5db;
}

.flatpickr-time .numInputWrapper span.arrowUp {
  top: 0;
}

.flatpickr-time .numInputWrapper span.arrowDown {
  top: auto;
  bottom: 0;
}

.flatpickr-time .numInputWrapper span.arrowUp::after {
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-bottom: 7px solid rgba(55,65,81,0.8);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}

.flatpickr-time .numInputWrapper span.arrowDown::after {
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-top: 7px solid rgba(55,65,81,0.8);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}
</style>
