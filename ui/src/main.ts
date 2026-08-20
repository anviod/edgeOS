import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './assets/css/globals.css'
import { useThemeStore } from './stores/theme'
import { vHud } from './directives/hud'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// 全局注册 2.5D 场景 HUD 指令
app.directive('hud', vHud)

// 在 mount 前初始化主题，避免 FOUC（样式闪烁）
const themeStore = useThemeStore()
themeStore.init()

app.mount('#app')