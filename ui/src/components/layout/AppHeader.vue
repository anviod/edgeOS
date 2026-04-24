<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bell, LogOut, Search, Settings, User } from 'lucide-vue-next'
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

const systemStatus = ref('RUNNING')
const onlineDevices = ref(128)
const alarmCount = ref(3)
const avgLatency = ref(32)
const packetLoss = ref(0.2)
const qualityScore = ref(98)

interface BreadcrumbItem {
  label: string
  path?: string
}

const breadcrumbs = computed<BreadcrumbItem[]>(() => {
  const base: BreadcrumbItem[] = [{ label: 'EdgeOS', path: '/dashboard' }]

  if (route.path.includes('/nodes/') && route.path.includes('/devices/') && route.path.includes('/points')) {
    const nodeId = String(route.params.nodeId || '')
    const deviceId = String(route.params.deviceId || '')
    return [
      ...base,
      { label: '节点管理', path: '/nodes' },
      { label: nodeId, path: `/nodes/${nodeId}/devices` },
      { label: deviceId },
    ]
  }

  if (route.path.includes('/nodes/') && route.path.includes('/devices')) {
    const nodeId = String(route.params.nodeId || '')
    return [...base, { label: '节点管理', path: '/nodes' }, { label: nodeId }]
  }

  const sectionTitle = typeof route.meta.sectionTitle === 'string' ? route.meta.sectionTitle : ''
  const parentTitle = typeof route.meta.parentTitle === 'string' ? route.meta.parentTitle : ''
  const parentPath = typeof route.meta.parentPath === 'string' ? route.meta.parentPath : ''
  const title = typeof route.meta.title === 'string' ? route.meta.title : ''

  if (sectionTitle) {
    base.push({ label: sectionTitle })
  }
  if (parentTitle && parentPath && parentPath !== route.path) {
    base.push({ label: parentTitle, path: parentPath })
  }
  if (title && title !== parentTitle) {
    base.push({ label: title })
  }

  return base
})

function handleLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  showUserMenu.value = false
  router.push('/login')
}

function goToSettings() {
  showUserMenu.value = false
  router.push('/settings')
}

const currentUser = computed(() => localStorage.getItem('username') || 'admin')

function onClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('[data-user-menu]')) {
    showUserMenu.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onUnmounted(() => document.removeEventListener('click', onClickOutside))
</script>

<template>
  <header
    class="app-header fixed right-0 top-0 z-20 flex h-14 items-center justify-between px-6 transition-all duration-200"
    :class="sidebarCollapsed ? 'left-16' : 'left-60'"
  >
    <nav class="flex items-center gap-1.5 text-sm">
      <template v-for="(crumb, index) in breadcrumbs" :key="`${crumb.label}-${index}`">
        <router-link
          v-if="crumb.path && index < breadcrumbs.length - 1"
          :to="crumb.path"
          class="breadcrumb-link transition-colors hover:text-sky-400"
        >
          {{ crumb.label }}
        </router-link>
        <span v-else :class="index < breadcrumbs.length - 1 ? 'breadcrumb-muted' : 'breadcrumb-active'">
          {{ crumb.label }}
        </span>
        <span v-if="index < breadcrumbs.length - 1" class="breadcrumb-sep">/</span>
      </template>
    </nav>

    <div class="hidden items-center gap-4 text-xs font-mono text-slate-500 xl:flex">
      <span>{{ systemStatus }}</span>
      <span>Devices: {{ onlineDevices }}</span>
      <span>Alarm: {{ alarmCount }}</span>
      <span>Latency: {{ avgLatency }}ms</span>
      <span>Loss: {{ packetLoss }}%</span>
      <span>Quality: {{ qualityScore }}</span>
    </div>

    <div class="flex items-center gap-2">
      <button
        class="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-500 hover:bg-slate-50"
      >
        <Search class="h-4 w-4" />
        <span class="hidden sm:inline">搜索业务、群控或设备...</span>
        <kbd class="hidden text-xs text-slate-400 sm:inline-block">Ctrl+K</kbd>
      </button>

      <div class="flex items-center gap-1.5 rounded-md px-2 py-1 text-xs" :class="connected ? 'ws-connected' : 'ws-disconnected'">
        <span class="ws-dot h-1.5 w-1.5 rounded-full" />
        <span>{{ connected ? '已连接' : '未连接' }}</span>
      </div>

      <ThemeToggle />

      <router-link
        to="/alerts"
        class="header-icon-btn relative flex h-9 w-9 items-center justify-center rounded-lg transition-colors"
      >
        <Bell class="header-icon h-[18px] w-[18px]" />
        <span
          v-if="alertStore.unacknowledgedCount > 0"
          class="absolute -right-0.5 -top-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full px-0.5 text-white"
          style="background: #EF4444; font-size: 10px; font-weight: 700;"
        >
          {{ alertStore.unacknowledgedCount > 9 ? '9+' : alertStore.unacknowledgedCount }}
        </span>
      </router-link>

      <div class="header-divider h-5 w-px" />

      <div class="relative" data-user-menu>
        <button
          class="header-user-btn flex items-center gap-2 rounded-lg px-2 py-1.5 transition-colors"
          @click="showUserMenu = !showUserMenu"
        >
          <div class="flex h-7 w-7 items-center justify-center rounded-lg" style="background: #0EA5E9;">
            <User class="h-3.5 w-3.5 text-white" />
          </div>
          <span class="header-username text-sm font-medium">{{ currentUser }}</span>
        </button>

        <div
          v-if="showUserMenu"
          class="header-dropdown absolute right-0 mt-2 w-44 overflow-hidden rounded-lg"
          style="top: 100%;"
        >
          <button
            class="header-dropdown-item flex w-full items-center gap-2.5 px-4 py-2.5 text-sm transition-colors"
            @click="goToSettings"
          >
            <Settings class="header-icon-muted h-4 w-4" />
            系统设置
          </button>
          <div class="header-dropdown-divider" />
          <button
            class="flex w-full items-center gap-2.5 px-4 py-2.5 text-sm transition-colors hover:bg-red-500/10"
            style="color: #EF4444;"
            @click="handleLogout"
          >
            <LogOut class="h-4 w-4" />
            退出登录
          </button>
        </div>
      </div>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  background: var(--header-bg);
  border-bottom: 1px solid var(--border-color);
}

.breadcrumb-link,
.breadcrumb-muted {
  color: var(--text-secondary);
}

.breadcrumb-active {
  color: var(--text-primary);
  font-weight: 600;
}

.breadcrumb-sep {
  color: var(--border-color);
}

.header-icon-btn,
.header-icon {
  color: var(--text-secondary);
}

.header-icon-btn:hover,
.header-user-btn:hover {
  background: var(--bg-hover);
}

.header-divider,
.header-dropdown-divider {
  background: var(--border-color);
}

.header-username {
  color: var(--text-primary);
}

.header-dropdown {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
}

.header-dropdown-item {
  color: var(--text-primary);
}

.header-dropdown-item:hover {
  background: var(--bg-hover);
}

.header-icon-muted {
  color: var(--text-muted);
}

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
}

.ws-disconnected .ws-dot {
  background: #EF4444;
}
</style>
