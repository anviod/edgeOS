<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { Pin, X } from 'lucide-vue-next'
import { useIsoHud } from '@/composables/useIsoHud'

const { state, togglePin, hide } = useIsoHud()

const rootEl = ref<HTMLElement | null>(null)
const tipEl = ref<HTMLElement | null>(null)
const tipSize = ref({ w: 244, h: 132 })

function measure() {
  if (tipEl.value) {
    tipSize.value = { w: tipEl.value.offsetWidth, h: tipEl.value.offsetHeight }
  }
}

// 尺寸变化后重新测量（内容行数变化 / 首次显示）
watch(
  () => [state.visible, state.pinned, state.data],
  () => {
    if (state.visible) requestAnimationFrame(() => nextTick(measure))
  },
)

// 场景重新挂载时重置陈旧状态，避免跨页面残留提示
onMounted(() => {
  hide()
  window.addEventListener('resize', measure)
})
onUnmounted(() => window.removeEventListener('resize', measure))

const place = computed(() => {
  if (!state.visible) return { opacity: 0 }
  const el = rootEl.value
  if (!el) return { opacity: 0 }
  const scene = el.parentElement
  if (!scene) return { opacity: 0 }
  const rect = scene.getBoundingClientRect()
  const { w, h } = tipSize.value
  const pad = 10
  let left = state.x - rect.left + 16
  let top = state.y - rect.top + 16
  const maxL = rect.width - w - pad
  if (left > maxL) left = state.x - rect.left - w - 12
  left = Math.min(Math.max(pad, left), Math.max(pad, maxL))
  const maxT = rect.height - h - pad
  top = Math.min(Math.max(pad, top), Math.max(pad, maxT))
  return { left: `${left}px`, top: `${top}px` }
})

// 颜色工具：rgba 与 hex 都兼容的发光 / 描边值
function glow(color?: string) {
  if (!color) return '0 0 0 rgba(0,0,0,0)'
  return color.trim().startsWith('#') ? `0 0 8px ${color}80` : `0 0 8px ${color}`
}

function borderC(color?: string) {
  if (!color) return 'var(--border-color)'
  return color.trim().startsWith('#') ? `${color}55` : color
}

// Esc 解锁
let keyHandler: ((e: KeyboardEvent) => void) | null = null
watch(
  () => state.pinned,
  (p) => {
    if (p) {
      keyHandler = (e: KeyboardEvent) => {
        if (e.key === 'Escape') hide()
      }
      window.addEventListener('keydown', keyHandler)
    } else if (keyHandler) {
      window.removeEventListener('keydown', keyHandler)
      keyHandler = null
    }
  },
)

onUnmounted(() => {
  if (keyHandler) window.removeEventListener('keydown', keyHandler)
  window.removeEventListener('resize', measure)
})
</script>

<template>
  <div ref="rootEl" class="iso-hud" :class="{ 'iso-hud--pin': state.pinned }">
    <Transition name="iso-hud-fade">
      <div v-if="state.visible && state.data" ref="tipEl" class="iso-hud-tip" :style="place">
        <div class="iso-hud-card">
          <div class="iso-hud-head">
            <span v-if="state.data.accent" class="iso-hud-dot" :style="{ background: state.data.accent, boxShadow: glow(state.data.accent) }" />
            <span class="iso-hud-title">{{ state.data.title }}</span>
            <span v-if="state.data.badge" class="iso-hud-badge" :style="{ color: state.data.accent ?? '#0EA5E9', borderColor: borderC(state.data.accent) }">{{ state.data.badge }}</span>
            <button class="iso-hud-close" :title="state.pinned ? '解锁提示' : '锁定提示'" @mousedown.prevent.stop="togglePin(state.x, state.y)">
              <X v-if="state.pinned" class="h-3 w-3" />
              <Pin v-else class="h-3 w-3" />
            </button>
          </div>
          <div v-if="state.data.rows?.length" class="iso-hud-rows">
            <div v-for="(r, i) in state.data.rows" :key="i" class="iso-hud-row">
              <span class="iso-hud-row-label">{{ r.label }}</span>
              <span class="iso-hud-row-value" :style="r.color ? { color: r.color } : {}">{{ r.value }}</span>
            </div>
          </div>
          <div v-if="state.data.foot" class="iso-hud-foot">{{ state.data.foot }}</div>
        </div>
        <div class="iso-hud-hint">{{ state.pinned ? '已锁定 · Esc 解锁' : '点击锁定' }}</div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.iso-hud {
  position: absolute;
  inset: 0;
  z-index: 50;
  pointer-events: none;
}

.iso-hud-tip {
  position: absolute;
  pointer-events: none;
  will-change: left, top;
}

.iso-hud-card {
  min-width: 196px;
  max-width: 268px;
  background: var(--bg-secondary);
  border: 1px solid var(--accent-border);
  border-radius: 10px;
  padding: 10px 12px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
}

.iso-hud--pin .iso-hud-card {
  border-color: rgba(245, 158, 11, 0.5);
}

.iso-hud-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.iso-hud-dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.iso-hud-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: 0.02em;
}

.iso-hud-badge {
  flex: none;
  padding: 1px 6px;
  border: 1px solid;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  white-space: nowrap;
}

.iso-hud-close {
  flex: none;
  margin-left: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 6px;
  color: var(--text-muted);
  cursor: pointer;
  pointer-events: auto;
}

.iso-hud-close:hover {
  color: #0EA5E9;
  background: rgba(14, 165, 233, 0.12);
}

.iso-hud-rows {
  margin-top: 8px;
  display: grid;
  gap: 4px;
}

.iso-hud-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 11px;
}

.iso-hud-row-label {
  color: var(--text-secondary);
}

.iso-hud-row-value {
  font-family: 'JetBrains Mono', 'SF Mono', 'Fira Code', Consolas, monospace;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--text-primary);
}

.iso-hud-foot {
  margin-top: 8px;
  padding-top: 7px;
  border-top: 1px dashed var(--border-color);
  font-size: 10px;
  color: var(--text-muted);
}

.iso-hud-hint {
  margin-top: 6px;
  font-size: 9px;
  letter-spacing: 0.16em;
  color: var(--text-muted);
  opacity: 0.85;
}

.iso-hud-fade-enter-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.iso-hud-fade-leave-active {
  transition: opacity 0.08s ease;
}

.iso-hud-fade-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
</style>