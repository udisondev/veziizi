import { computed } from 'vue'
import { useWindowSize } from '@vueuse/core'

export type Breakpoint = 'mobile' | 'tablet' | 'desktop'

const SM = 768
const LG = 1240
const TW_LG = 1024 // Tailwind lg: breakpoint

export function useBreakpoint() {
  const { width } = useWindowSize()

  const breakpoint = computed<Breakpoint>(() => {
    if (width.value < SM) return 'mobile'
    if (width.value < LG) return 'tablet'
    return 'desktop'
  })

  const isMobile = computed(() => breakpoint.value === 'mobile')
  const isTablet = computed(() => breakpoint.value === 'tablet')
  const isDesktop = computed(() => breakpoint.value === 'desktop')
  const isMobileOrTablet = computed(() => breakpoint.value !== 'desktop')
  // Совпадает с Tailwind lg: (1024px) — для синхронизации с CSS-брейкпоинтами
  const isBelowLg = computed(() => width.value < TW_LG)

  return {
    breakpoint,
    isMobile,
    isTablet,
    isDesktop,
    isMobileOrTablet,
    isBelowLg,
  }
}
