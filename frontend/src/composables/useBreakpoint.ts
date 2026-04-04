import { ref, onMounted, onUnmounted } from 'vue'

export type Breakpoint = 'mobile' | 'tablet' | 'desktop'

const SM = 768
const LG = 1240

function getBreakpoint(width: number): Breakpoint {
  if (width < SM) return 'mobile'
  if (width < LG) return 'tablet'
  return 'desktop'
}

export function useBreakpoint() {
  const breakpoint = ref<Breakpoint>(getBreakpoint(window.innerWidth))

  const isMobile = () => breakpoint.value === 'mobile'
  const isTablet = () => breakpoint.value === 'tablet'
  const isDesktop = () => breakpoint.value === 'desktop'
  const isMobileOrTablet = () => breakpoint.value !== 'desktop'

  function onResize() {
    breakpoint.value = getBreakpoint(window.innerWidth)
  }

  onMounted(() => window.addEventListener('resize', onResize))
  onUnmounted(() => window.removeEventListener('resize', onResize))

  return { breakpoint, isMobile, isTablet, isDesktop, isMobileOrTablet }
}
