import { ref, onMounted, onUnmounted } from 'vue'
import { formatDateTime, formatTime } from '@/lib/live-time'

/** 每秒刷新的实时时钟，dateTime 为 MM-DD HH:mm:ss，time 为 HH:mm:ss */
export function useLiveClock() {
  const dateTime = ref(formatDateTime(new Date()))
  const time = ref(formatTime(new Date()))
  let timer: ReturnType<typeof setInterval> | null = null

  function start() {
    if (timer) return
    timer = setInterval(() => {
      dateTime.value = formatDateTime(new Date())
      time.value = formatTime(new Date())
    }, 1000)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(start)
  onUnmounted(stop)

  return { dateTime, time }
}
