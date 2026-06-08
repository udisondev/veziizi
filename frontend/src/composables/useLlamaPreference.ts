import { useLocalStorage } from '@vueuse/core'

export function useLlamaPreference() {
  const enabled = useLocalStorage('veziizi:llama-walker', true)
  return { enabled }
}
