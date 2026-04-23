<script setup lang="ts">
import { Sun, Moon } from 'lucide-vue-next'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
</script>

<template>
  <button
    @click="themeStore.toggle()"
    class="theme-toggle-btn"
    :title="themeStore.isDark ? '切换亮色模式' : '切换暗色模式'"
  >
    <Transition name="icon-flip" mode="out-in">
      <Moon v-if="!themeStore.isDark" key="moon" class="w-4 h-4" />
      <Sun v-else key="sun" class="w-4 h-4" />
    </Transition>
  </button>
</template>

<style scoped>
.theme-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}
.theme-toggle-btn:hover {
  background: var(--bg-hover);
  color: var(--accent-primary);
  border-color: var(--accent-border);
  transform: scale(1.05);
}

/* icon flip transition */
.icon-flip-enter-active,
.icon-flip-leave-active {
  transition: all 0.2s ease;
}
.icon-flip-enter-from {
  opacity: 0;
  transform: rotate(-90deg) scale(0.7);
}
.icon-flip-leave-to {
  opacity: 0;
  transform: rotate(90deg) scale(0.7);
}
</style>
