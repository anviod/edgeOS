<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bell, User, LogOut, Settings, ChevronRight, Search } from 'lucide-vue-next'
import { useAlertStore } from '@/stores/alert'
import { useWebSocket } from '@/composables/useWebSocket'
import ThemeToggle from './ThemeToggle.vue'

const props = defineProps<{
  sidebarCollapsed?: boolean
}>()

const route = useRoute()
const router = useRouter()
const alertStore = useAlertStore()
const { connected } = useWebSocket()
const showUserMenu = ref(false)

// 工业运行态信息
const systemStatus = ref('RUNNING')
const onlineDevices = ref(128)
const alarmCount = ref(3)
const avgLatency = ref(32)
const packetLoss = ref(0.2)

const routeTitles: Record<string, string> = {
  '/dashboard': '系统总览',
  '/middleware': '中间件连接',
  '/nodes': '节点管理',
  '/control': '设备控制',
  '/alerts': '告警管理',
  '/settings': '系统设置',
}

const breadcrumbs = computed(() => {
  const crumbs: { label: string; path?: string }[] = [{ label: 'EdgeOS', path: '/dashboard' }]
  const path = route.path

  if (path.includes('/devices') && path.includes('/nodes')) {
    crumbs.push({ label: '节点管理', path: '/nodes' })
    const nodeId = route.params.nodeId as string
    if (nodeId) crumbs.push({ label: nodeId, path: `/nodes/${nodeId}/devices` })
    if (path.includes('/points')) {
      const deviceId = route.params.deviceId as string
      if (deviceId) crumbs.push({ label: deviceId })
    }
  } else {
    const title = routeTitles[path] || route.meta?.title as string || ''
    if (title) crumbs.push({ label: title })
  }
  return crumbs
})

function handleLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  showUserMenu.value = false
  router.push('/login')
}

function goToSettings() {
  router.push('/settings')
  showUserMenu.value = false
}

const currentUser = computed(() => localStorage.getItem('username') || 'admin')

function onClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('[data-user-menu]')) showUserMenu.value = false
}

onMounted(() => document.addEventListener('click', onClickOutside))
onUnmounted(() => document.removeEventListener('click', onClickOutside))
</script>

<template>
  <header
    class="app-header fixed right-0 top-0 z-20 h-14 flex items-center justify-between px-6 transition-all duration-200"
    :class="sidebarCollapsed ? 'left-16' : 'left-60'"
  >
    <!-- Breadcrumb -->
    <nav class="flex items-center gap-1.5 text-sm">
      <template v-for="(crumb, idx) in breadcrumbs" :key="idx">
        <router-link
          v-if="crumb.path && idx < breadcrumbs.length - 1"
          :to="crumb.path"
          class="breadcrumb-link transition-colors hover:text-sky-400"
        >{{ crumb.label }}</router-link>
        <span v-else-if="idx < breadcrumbs.length - 1" class="breadcrumb-muted">{{ crumb.label }}</span>
        <span v-else class="font-semibold breadcrumb-active">{{ crumb.label }}</span>
        <span
          v-if="idx < breadcrumbs.length - 1"
          class="w-3.5 h-3.5 flex-shrink-0 breadcrumb-sep"
          style="width:14px;height:14px;"
        >{{ '/' }}</span>
      </template>
    </nav>

    <!-- 工业运行态信息条 -->
    <div class="flex items-center gap-4 text-xs font-mono text-slate-500">
      <span>{{ systemStatus }}</span>
      <span>Devices: {{ onlineDevices }}</span>
      <span>Alarm: {{ alarmCount }}</span>
      <span>Latency: {{ avgLatency }}ms</span>
      <span>Loss: {{ packetLoss }}%</span>
    </div>

    <!-- Right controls -->
    <div class="flex items-center gap-2">
      <!-- 全局搜索 -->
      <button
        class="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-500 hover:bg-slate-50"
      >
        <Search class="h-4 w-4" />
        <span class="hidden sm:inline">搜索...</span>
        <kbd class="hidden text-xs text-slate-400 sm:inline-block">Ctrl+K</kbd>
      </button>

      <!-- WebSocket status indicator -->
      <div class="flex items-center gap-1.5 px-2 py-1 rounded-md text-xs" :class="connected ? 'ws-connected' : 'ws-disconnected'">
        <span class="w-1.5 h-1.5 rounded-full ws-dot"></span>
        <span>{{ connected ? '已连接' : '未连接' }}</span>
      </div>

      <!-- Theme toggle -->
      <ThemeToggle />

      <!-- Alert bell -->
      <router-link
        to="/alerts"
        class="header-icon-btn relative flex items-center justify-center w-9 h-9 rounded-lg transition-colors"
      >
        <Bell style="width:18px;height:18px;" class="header-icon" />
        <span
          v-if="alertStore.unacknowledgedCount > 0"
          class="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[16px] h-4 rounded-full text-white px-0.5"
          style="background:#EF4444; font-size:10px; font-weight:700;"
        >
          {{ alertStore.unacknowledgedCount > 9 ? '9+' : alertStore.unacknowledgedCount }}
        </span>
      </router-link>

      <!-- Divider -->
      <div class="header-divider w-px h-5" />

      <!-- User menu -->
      <div class="relative" data-user-menu>
        <button
          @click="showUserMenu = !showUserMenu"
          class="header-user-btn flex items-center gap-2 rounded-lg px-2 py-1.5 transition-colors"
        >
          <div class="w-7 h-7 rounded-lg flex items-center justify-center"
            style="background: #0EA5E9;">
            <User class="w-3.5 h-3.5 text-white" style="width:14px;height:14px;" />
          </div>
          <span class="text-sm font-medium header-username">{{ currentUser }}</span>
        </button>

        <!-- Dropdown -->
        <Transition
          enter-active-class="transition-all duration-150"
          enter-from-class="opacity-0 -translate-y-1 scale-95"
          enter-to-class="opacity-100 translate-y-0 scale-100"
          leave-active-class="transition-all duration-100"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div
            v-if="showUserMenu"
            class="header-dropdown absolute right-0 mt-2 w-44 rounded-lg overflow-hidden"
            style="top: 100%;"
          >
            <button
              @click="goToSettings"
              class="header-dropdown-item w-full flex items-center gap-2.5 px-4 py-2.5 text-sm transition-colors"
            >
              <Settings class="w-4 h-4 header-icon-muted" />
              系统设置
            </button>
            <div class="header-dropdown-divider" />
            <button
              @click="handleLogout"
              class="w-full flex items-center gap-2.5 px-4 py-2.5 text-sm transition-colors hover:bg-red-500/10"
              style="color: #EF4444;"
            >
              <LogOut class="w-4 h-4" />
              退出登录
            </button>
          </div>
        </Transition>
      </div>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  background: var(--header-bg);
  border-bottom: 1px solid var(--border-color);
  transition: background-color 0.2s ease, border-color 0.2s ease, left 0.2s ease;
}
.breadcrumb-link { color: var(--text-secondary); }
.breadcrumb-link:hover { color: var(--text-primary); }
.breadcrumb-muted { color: var(--text-secondary); }
.breadcrumb-active { color: var(--text-primary); font-weight: 600; }
.breadcrumb-sep { color: var(--border-color); }

.header-icon-btn { color: var(--text-secondary); }
.header-icon-btn:hover { background: var(--bg-hover); }
.header-icon { color: var(--text-secondary); }
.header-divider { background: var(--border-color); }
.header-username { color: var(--text-primary); }
.header-user-btn:hover { background: var(--bg-hover); }

.header-dropdown {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
}
.header-dropdown-item {
  color: var(--text-primary);
}
.header-dropdown-item:hover { background: var(--bg-hover); }
.header-icon-muted { color: var(--text-muted); }
.header-dropdown-divider {
  height: 1px;
  background: var(--border-color);
}

/* WebSocket status indicator */
.ws-connected {
  background: rgba(16, 185, 129, 0.1);
  color: #10B981;
}
.ws-disconnected {
  background: rgba(239, 68, 68, 0.1);
  color: #EF4444;
}
.ws-connected .ws-dot {
  background: #10B981;
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.5);
}
.ws-disconnected .ws-dot {
  background: #EF4444;
}
</style>
