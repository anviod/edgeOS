<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Maximize2, Minimize2 } from 'lucide-vue-next'

withDefaults(defineProps<{
  // 页面自身已做全屏铺满（如工业大屏），进入全屏时不做内边距
  bleed?: boolean
}>(), {
  bleed: false,
})

const root = ref<HTMLElement>()
const isFullscreen = ref(false)

function onChange() {
  isFullscreen.value = document.fullscreenElement === root.value
}

function toggle() {
  if (!root.value) return
  if (document.fullscreenElement) {
    document.exitFullscreen()
  } else {
    root.value.requestFullscreen()
  }
}

onMounted(() => document.addEventListener('fullscreenchange', onChange))
onUnmounted(() => document.removeEventListener('fullscreenchange', onChange))
</script>

<template>
  <div
    ref="root"
    class="visual-screen"
    :class="{ 'visual-screen--fs': isFullscreen, 'visual-screen--bleed': bleed }"
  >
    <button class="visual-screen__btn" :title="isFullscreen ? '退出全屏' : '全屏展示'" @click="toggle">
      <component :is="isFullscreen ? Minimize2 : Maximize2" class="h-4 w-4" />
      <span>{{ isFullscreen ? '退出全屏' : '全屏展示' }}</span>
    </button>
    <slot />
  </div>
</template>

<style scoped>
.visual-screen {
  position: relative;
}

.visual-screen__btn {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 90;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.04em;
  transition: all 0.15s ease;
  cursor: pointer;
}

.visual-screen__btn:hover {
  border-color: rgba(14, 165, 233, 0.5);
  color: #0EA5E9;
}

.visual-screen--fs {
  position: fixed;
  inset: 0;
  z-index: 999;
  background: var(--bg-primary);
}

.visual-screen--fs:not(.visual-screen--bleed) {
  padding: 24px;
  overflow: auto;
}

.visual-screen--fs.visual-screen--bleed {
  padding: 0;
  overflow: hidden;
  background: var(--bg-primary);
}

.visual-screen--fs .visual-screen__btn {
  position: fixed;
}
</style>
