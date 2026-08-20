import { reactive } from 'vue'

// =====================================================
// 2.5D 等距场景 悬停/点击 HUD 提示（全局单例）
// 任意 Iso 组件通过 v-hud 指令上报提示信息，
// IsoScene 内的 IsoHud 组件读取该状态渲染浮层。
// =====================================================

export interface IsoHudRow {
  label: string
  value: string
  color?: string
}

export interface IsoHudData {
  title?: string
  badge?: string
  accent?: string
  rows?: IsoHudRow[]
  foot?: string
}

interface IsoHudState {
  visible: boolean
  pinned: boolean
  x: number
  y: number
  data: IsoHudData | null
  target: HTMLElement | null
}

const state = reactive<IsoHudState>({
  visible: false,
  pinned: false,
  x: 0,
  y: 0,
  data: null,
  target: null,
})

export function useIsoHud() {
  function show(data: IsoHudData, el: HTMLElement, x: number, y: number, keepPos = false) {
    const changed = state.target !== el
    state.target = el
    // 已锁定时悬停到其它对象：仅更新内容，不跟随光标
    if (!state.pinned || !keepPos) {
      state.x = x
      state.y = y
    }
    if (changed && !state.pinned) state.pinned = false
    state.data = data
    state.visible = true
  }

  function move(x: number, y: number) {
    if (!state.pinned) {
      state.x = x
      state.y = y
    }
  }

  function togglePin(x: number, y: number) {
    if (state.pinned) {
      state.pinned = false
      state.visible = false
    } else {
      state.pinned = true
      state.x = x
      state.y = y
      state.visible = true
    }
  }

  function hide() {
    state.pinned = false
    state.visible = false
    state.target = null
  }

  return { state, show, move, togglePin, hide }
}