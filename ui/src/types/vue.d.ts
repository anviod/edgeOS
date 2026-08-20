import type { Directive } from 'vue'

declare module 'vue' {
  interface GlobalDirectives {
    hud: Directive<HTMLElement, unknown>
  }
}

export {}