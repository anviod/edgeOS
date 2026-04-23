export function useDebounce<T extends (...args: any[]) => any>(
  fn: T,
  delay: number = 300
) {
  let timer: NodeJS.Timeout | null = null
  
  return (...args: Parameters<T>) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => fn(...args), delay)
  }
}

export function useThrottle<T extends (...args: any[]) => any>(
  fn: T,
  delay: number = 200
) {
  let lastCall = 0
  
  return (...args: Parameters<T>) => {
    const now = Date.now()
    if (now - lastCall >= delay) {
      lastCall = now
      fn(...args)
    }
  }
}