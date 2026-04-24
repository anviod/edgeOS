import { computed } from 'vue'
import { getP3PageData } from '@/mock/p3'

export function useP3Page(pageKey: string) {
  const page = computed(() => getP3PageData(pageKey))
  return { page }
}
