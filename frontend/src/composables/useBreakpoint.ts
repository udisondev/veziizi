import { computed } from 'vue'
import { useWindowSize } from '@vueuse/core'

export type Breakpoint = 'mobile' | 'tablet' | 'desktop'

const SM = 768
const LG = 1240

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

  return {
    breakpoint,
    isMobile,
    isTablet,
    isDesktop,
    isMobileOrTablet,
  }
}
