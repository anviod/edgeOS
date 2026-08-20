import type { Directive } from 'vue'
import { useIsoHud, type IsoHudData } from '@/composables/useIsoHud'

// =====================================================
// v-hud 指令：为 2.5D 等距对象绑定 悬停/移动/点击 HUD 交互
// 用法：<IsoCube :hud="{ title, badge, rows, foot }" />
// =====================================================

const cache = new WeakMap<HTMLElement, IsoHudData | undefined>()

function dataOf(el: HTMLElement): IsoHudData | undefined {
  return cache.get(el)
}

function track(el: HTMLElement, data?: IsoHudData) {
  cache.set(el, data)
}

function onEnter(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const data = dataOf(el)
  if (!data) return
  const { state, show } = useIsoHud()
  if (state.target === el && state.visible) return
  show(data, el, e.clientX, e.clientY, state.pinned)
}

function onMove(e: MouseEvent) {
  const { state, move } = useIsoHud()
  if (state.visible) move(e.clientX, e.clientY)
}

function onLeave(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const { state, hide } = useIsoHud()
  // 锁定状态下离开对象，提示保持显示
  if (state.target === el && !state.pinned) hide()
}

function onClick(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const data = dataOf(el)
  if (!data) return
  const { state, show, togglePin } = useIsoHud()
  if (state.target !== el) show(data, el, e.clientX, e.clientY)
  togglePin(e.clientX, e.clientY)
}

export const vHud: Directive<HTMLElement, IsoHudData | undefined> = {
  mounted(el, binding) {
    track(el, binding.value)
    el.addEventListener('mouseenter', onEnter, { passive: true })
    el.addEventListener('mousemove', onMove, { passive: true })
    el.addEventListener('mouseleave', onLeave, { passive: true })
    el.addEventListener('click', onClick)
  },
  updated(el, binding) {
    track(el, binding.value)
  },
  unmounted(el) {
    cache.delete(el)
    el.removeEventListener('mouseenter', onEnter)
    el.removeEventListener('mousemove', onMove)
    el.removeEventListener('mouseleave', onLeave)
    el.removeEventListener('click', onClick)
  },
}