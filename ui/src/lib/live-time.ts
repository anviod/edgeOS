// =====================================================
// 实时时间工具：为业务中心提供跟随当前时间的动效时间戳
// =====================================================

export function pad(n: number) {
  return String(n).padStart(2, '0')
}

/** 将 Date 格式化为 MM-DD HH:mm:ss */
export function formatDateTime(d: Date) {
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** 将 Date 格式化为 HH:mm:ss */
export function formatTime(d: Date) {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** 生成当前时间往前的相对时间戳，步进 minutes 递增 */
export function relativeTimestamps(count: number, stepMinutes = 12) {
  const now = Date.now()
  return Array.from({ length: count }, (_, i) => formatDateTime(new Date(now - i * stepMinutes * 60000)))
}

/** 将 item 列表的 timestamp 字段重写为基于当前时间的相对时间 */
export function rebaseTimestamps<T extends { timestamp: string }>(items: T[], stepMinutes = 12): T[] {
  const stamps = relativeTimestamps(items.length, stepMinutes)
  return items.map((item, i) => ({ ...item, timestamp: stamps[i] ?? item.timestamp }))
}
