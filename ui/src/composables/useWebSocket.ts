import { ref, onUnmounted } from 'vue'
import type { RealtimeEvent, WsEventType } from '@/types/edgex'

type EventHandler<T = unknown> = (payload: T) => void

const WS_URL = 'ws://localhost:8000/ws'
const RECONNECT_DELAY_MS = 3000
const MAX_RECONNECT = 10

let ws: WebSocket | null = null
let reconnectCount = 0
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
const handlers = new Map<WsEventType, Set<EventHandler>>()
const connected = ref(false)

function dispatch(event: RealtimeEvent) {
  const set = handlers.get(event.type)
  if (set) {
    set.forEach(fn => fn(event.payload))
  }
}

function connect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return
  }

  const token = localStorage.getItem('token')
  if (!token) {
    console.warn('[WS] No token found, WebSocket connection will be rejected')
    return
  }

  const url = `${WS_URL}?token=${encodeURIComponent(token)}`
  console.info('[WS] Connecting to', url)
  ws = new WebSocket(url)

  ws.onopen = () => {
    connected.value = true
    reconnectCount = 0
    console.info('[WS] connected')
  }

  ws.onmessage = (e) => {
    try {
      const event: RealtimeEvent = JSON.parse(e.data)
      dispatch(event)
    } catch (err) {
      console.error('[WS] parse error', err)
    }
  }

  ws.onclose = (event) => {
    connected.value = false
    console.info('[WS] disconnected', event.code, event.reason)
    ws = null
    scheduleReconnect()
  }

  ws.onerror = (err) => {
    console.error('[WS] error', err)
    // 不要在onerror中关闭连接，让onclose处理重连
  }
}

function scheduleReconnect() {
  if (reconnectCount >= MAX_RECONNECT) return
  reconnectCount++
  reconnectTimer = setTimeout(() => {
    connect()
  }, RECONNECT_DELAY_MS * Math.min(reconnectCount, 4))
}

function disconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  reconnectCount = MAX_RECONNECT // prevent auto reconnect
  ws?.close()
  ws = null
  connected.value = false
}

// ==================== Composable ====================

export function useWebSocket() {
  function on<T = unknown>(type: WsEventType, handler: EventHandler<T>) {
    if (!handlers.has(type)) {
      handlers.set(type, new Set())
    }
    handlers.get(type)!.add(handler as EventHandler)

    onUnmounted(() => {
      handlers.get(type)?.delete(handler as EventHandler)
    })
  }

  function off(type: WsEventType, handler: EventHandler) {
    handlers.get(type)?.delete(handler)
  }

  return {
    connected,
    connect,
    disconnect,
    on,
    off,
  }
}

// 全局自动连接（登录后调用）
export function initWebSocket() {
  connect()
}

export function destroyWebSocket() {
  disconnect()
  handlers.clear()
}
