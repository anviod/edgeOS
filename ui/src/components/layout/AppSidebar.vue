<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  LayoutDashboard,
  Server,
  Radio,
  Sliders,
  AlertTriangle,
  Settings,
  Activity,
  Zap,
  PanelLeftClose,
  PanelLeftOpen,
} from 'lucide-vue-next'
import { useAlertStore } from '@/stores/alert'
import { useAppStore } from '@/stores/app'

const route = useRoute()
const router = useRouter()
const alertStore = useAlertStore()
const appStore = useAppStore()

const navItems = [
  { name: '系统总览', path: '/dashboard', icon: LayoutDashboard },
  { name: '中间件连接', path: '/middleware', icon: Radio },
  { name: '节点管理', path: '/nodes', icon: Server },
  { name: '设备控制', path: '/control', icon: Sliders },
  { name: '告警管理', path: '/alerts', icon: AlertTriangle, badge: 'alerts' },
  { name: '系统设置', path: '/settings', icon: Settings },
]

function isActive(path: string): boolean {
  if (path === '/dashboard') return route.path === '/dashboard'
  return route.path === path || route.path.startsWith(path + '/')
}

const collapsed = computed(() => appStore.sidebarCollapsed)

function toggleSidebar() {
  appStore.toggleSidebar()
}
</script>

<template>
  <aside
    class="sidebar fixed left-0 top-0 z-30 h-screen flex flex-col transition-all duration-200"
    :class="collapsed ? 'w-16' : 'w-64'"
  >

    <!-- Logo -->
    <div class="sidebar-logo flex items-center gap-3 px-4 h-14 flex-shrink-0 overflow-hidden">
      <div
        class="relative flex items-center justify-center w-9 h-9 flex-shrink-0"
        style="background: #0EA5E9;"
      >
        <Zap class="w-5 h-5 text-white" />
      </div>
      <Transition name="fade-slide">
        <div v-if="!collapsed" class="overflow-hidden">
          <div class="text-base font-bold tracking-wide sidebar-title whitespace-nowrap">EdgeOS</div>
          <div class="text-xs sidebar-subtitle" style="letter-spacing:0.08em;">EDGE PLATFORM</div>
        </div>
      </Transition>
    </div>

    <!-- System status bar -->
    <div v-if="!collapsed" class="sidebar-status flex items-center gap-2 px-4 py-2.5 flex-shrink-0 border-b border-slate-200">
      <Activity class="w-3.5 h-3.5 flex-shrink-0" style="color: #10B981;" />
      <span class="text-xs font-medium sidebar-status-text">系统运行中</span>
      <span class="ml-auto text-xs tabular-nums sidebar-version">v0.0.1</span>
    </div>
    <div v-else class="sidebar-status flex items-center justify-center py-2.5 flex-shrink-0 border-b border-slate-200">
      <Activity class="w-3.5 h-3.5" style="color: #10B981;" />
    </div>

    <!-- Nav -->
    <nav class="flex-1 overflow-y-auto py-3 scrollbar-industrial"
      :class="collapsed ? 'px-1.5' : 'px-2'"
    >
      <div v-if="!collapsed" class="mb-1 px-3 py-1">
        <span class="text-xs font-semibold tracking-widest uppercase sidebar-nav-label">导航</span>
      </div>

      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item group relative flex items-center rounded-lg mb-0.5 transition-all duration-150"
        :class="[
          isActive(item.path) ? 'nav-item--active' : 'nav-item--inactive',
          collapsed ? 'justify-center w-11 h-10 mx-auto px-0' : 'gap-3 px-4 py-2',
        ]"
        :title="collapsed ? item.name : undefined"
      >
        <!-- Active indicator bar -->
        <span
          v-if="isActive(item.path) && !collapsed"
          class="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-5 rounded-r nav-active-bar"
        />
        <component
          :is="item.icon"
          class="flex-shrink-0 transition-colors"
          :class="isActive(item.path) ? 'nav-icon--active' : 'nav-icon--inactive group-hover:text-gray-300'"
          style="width:18px;height:18px;"
        />
        <Transition name="fade-slide">
          <span v-if="!collapsed" class="text-sm font-medium flex-1 whitespace-nowrap overflow-hidden">{{ item.name }}</span>
        </Transition>
        <!-- Alert badge -->
        <span
          v-if="!collapsed && item.badge === 'alerts' && alertStore.unacknowledgedCount > 0"
          class="flex items-center justify-center min-w-[18px] h-[18px] rounded-full text-xs font-bold px-1"
          style="background: #EF4444; color: #fff; font-size:10px;"
        >
          {{ alertStore.unacknowledgedCount > 99 ? '99+' : alertStore.unacknowledgedCount }}
        </span>
        <!-- Collapsed alert dot -->
        <span
          v-if="collapsed && item.badge === 'alerts' && alertStore.unacknowledgedCount > 0"
          class="absolute top-1 right-1 w-2 h-2 rounded-full"
          style="background: #EF4444;"
        />
      </router-link>
    </nav>

    <!-- Collapse toggle button -->
    <div class="sidebar-footer flex-shrink-0">
      <button
        @click="toggleSidebar"
        class="collapse-btn w-full flex items-center transition-colors"
        :class="collapsed ? 'justify-center h-12' : 'gap-3 px-4 py-3'"
        :title="collapsed ? '展开菜单' : '折叠菜单'"
      >
        <component
          :is="collapsed ? PanelLeftOpen : PanelLeftClose"
          class="flex-shrink-0"
          style="width:18px;height:18px;"
        />
        <Transition name="fade-slide">
          <span v-if="!collapsed" class="text-sm font-medium whitespace-nowrap">折叠菜单</span>
        </Transition>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border-color);
  transition: width 0.2s ease, background-color 0.2s ease, border-color 0.2s ease;
}

.sidebar-logo {
  border-bottom: 1px solid var(--border-color);
}
.sidebar-title { color: var(--text-primary); }
.sidebar-subtitle { color: var(--accent-primary); }

.sidebar-status {
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
}
.sidebar-status-text { color: var(--text-secondary); }
.sidebar-version { color: var(--text-muted); }

.sidebar-nav-label { color: var(--text-muted); }

/* nav item */
.nav-item--active {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border-left: 2px solid var(--accent-primary);
}
.nav-item--inactive {
  color: var(--text-secondary);
  border-left: 2px solid transparent;
}
.nav-item--inactive:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.nav-active-bar {
  background: var(--accent-primary);
}
.nav-icon--active { color: var(--accent-primary); }
.nav-icon--inactive { color: var(--text-muted); }

/* footer */
.sidebar-footer {
  border-top: 1px solid var(--border-color);
}
.collapse-btn {
  color: var(--text-secondary);
}
.collapse-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

/* 折叠时去掉active的左边框位移 */
:deep(.nav-item--active) {
  border-left-color: var(--accent-primary);
}

/* 工业级滚动条 */
.scrollbar-industrial {
  scrollbar-width: thin;
  scrollbar-color: var(--border-color) transparent;
}

.scrollbar-industrial::-webkit-scrollbar {
  width: 4px;
}

.scrollbar-industrial::-webkit-scrollbar-track {
  background: transparent;
}

.scrollbar-industrial::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 2px;
}

.scrollbar-industrial::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

/* 淡入淡出过渡 */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: opacity 0.15s ease;
}
.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
}
</style>
