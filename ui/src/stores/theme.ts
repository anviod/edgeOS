import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(false)

  function applyTheme(dark: boolean) {
    if (dark) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function init() {
    const saved = localStorage.getItem('edgeos-theme')
    isDark.value = saved === 'dark'
    applyTheme(isDark.value)
  }

  function toggle() {
    isDark.value = !isDark.value
    applyTheme(isDark.value)
    localStorage.setItem('edgeos-theme', isDark.value ? 'dark' : 'light')
  }

  return { isDark, init, toggle }
})
