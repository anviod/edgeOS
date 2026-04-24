<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { ChevronDown, PanelLeftClose, PanelLeftOpen, Zap } from 'lucide-vue-next'
import { useAlertStore } from '@/stores/alert'
import { useAppStore } from '@/stores/app'
import { isPathActive, navSections, type NavItem } from '@/lib/navigation'

const route = useRoute()
const alertStore = useAlertStore()
const appStore = useAppStore()

const collapsed = computed(() => appStore.sidebarCollapsed)
const expandedGroups = reactive<Record<string, boolean>>({
  business: true,
  'group-control': true,
})

function toggleSidebar() {
  appStore.toggleSidebar()
}

function toggleGroup(groupKey: string) {
  expandedGroups[groupKey] = !expandedGroups[groupKey]
}

function isActive(path: string): boolean {
  return isPathActive(path, route.path)
}

function hasActiveChild(item: NavItem): boolean {
  return Boolean(item.children?.some(child => isActive(child.path)))
}
</script>

<template>
  <aside
    class="sidebar fixed left-0 top-0 z-30 flex h-screen flex-col transition-all duration-200"
    :class="collapsed ? 'w-16' : 'w-64'"
  >
    <div class="sidebar-logo flex h-14 flex-shrink-0 items-center gap-3 overflow-hidden px-4">
      <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center" style="background: #0EA5E9;">
        <Zap class="h-5 w-5 text-white" />
      </div>
      <div v-if="!collapsed" class="overflow-hidden">
        <div class="sidebar-title whitespace-nowrap text-base font-bold tracking-wide">EdgeOS</div>
        <div class="sidebar-subtitle whitespace-nowrap text-xs tracking-[0.08em]">EDGE PLATFORM</div>
      </div>
    </div>

    <div
      v-if="!collapsed"
      class="sidebar-status flex flex-shrink-0 items-center gap-2 border-b border-slate-200 px-4 py-2.5"
    >
      <span class="inline-flex h-2.5 w-2.5 animate-pulse rounded-full" style="background: #10B981;" />
      <span class="sidebar-status-text text-xs font-medium">系统运行中</span>
      <span class="sidebar-version ml-auto text-xs tabular-nums">v0.0.1</span>
    </div>
    <div
      v-else
      class="sidebar-status flex flex-shrink-0 items-center justify-center border-b border-slate-200 py-2.5"
    >
      <span class="inline-flex h-2.5 w-2.5 animate-pulse rounded-full" style="background: #10B981;" />
    </div>

    <nav class="flex-1 overflow-y-auto px-2 py-3 scrollbar-industrial">
      <section
        v-for="section in navSections"
        :key="section.key"
        class="mb-4"
      >
        <div v-if="!collapsed" class="px-3 py-1">
          <span class="sidebar-nav-label text-xs font-semibold uppercase tracking-[0.18em]">{{ section.label }}</span>
        </div>

        <div class="space-y-1">
          <template v-for="item in section.items" :key="item.key">
            <div v-if="item.children?.length && !collapsed">
              <div
                class="nav-item nav-item--parent flex items-center gap-2 rounded-lg px-3 py-2"
                :class="(isActive(item.path) || hasActiveChild(item)) ? 'nav-item--active' : 'nav-item--inactive'"
              >
                <router-link
                  :to="item.path"
                  class="flex min-w-0 flex-1 items-center gap-3"
                >
                  <component
                    :is="item.icon"
                    class="h-[18px] w-[18px] flex-shrink-0"
                    :class="(isActive(item.path) || hasActiveChild(item)) ? 'nav-icon--active' : 'nav-icon--inactive'"
                  />
                  <span class="truncate text-sm font-medium">{{ item.label }}</span>
                </router-link>
                <button
                  class="rounded-md p-1 transition-colors hover:bg-white/10"
                  :title="expandedGroups[section.key] ? '收起子菜单' : '展开子菜单'"
                  @click="toggleGroup(section.key)"
                >
                  <ChevronDown class="h-4 w-4 transition-transform" :class="expandedGroups[section.key] ? '' : '-rotate-90'" />
                </button>
              </div>

              <div v-show="expandedGroups[section.key]" class="mt-1 space-y-1 pl-4">
                <router-link
                  v-for="child in item.children"
                  :key="child.key"
                  :to="child.path"
                  class="nav-item flex items-center gap-3 rounded-lg px-3 py-2"
                  :class="isActive(child.path) ? 'nav-item--active child-active' : 'nav-item--inactive'"
                >
                  <component
                    :is="child.icon"
                    class="h-4 w-4 flex-shrink-0"
                    :class="isActive(child.path) ? 'nav-icon--active' : 'nav-icon--inactive'"
                  />
                  <span class="text-sm">{{ child.label }}</span>
                </router-link>
              </div>
            </div>

            <router-link
              v-else
              :to="item.path"
              class="nav-item group relative flex items-center rounded-lg"
              :class="[
                isActive(item.path) ? 'nav-item--active' : 'nav-item--inactive',
                collapsed ? 'mx-auto h-10 w-11 justify-center px-0' : 'gap-3 px-4 py-2',
              ]"
              :title="collapsed ? item.label : undefined"
            >
              <component
                :is="item.icon"
                class="h-[18px] w-[18px] flex-shrink-0"
                :class="isActive(item.path) ? 'nav-icon--active' : 'nav-icon--inactive'"
              />
              <span v-if="!collapsed" class="flex-1 whitespace-nowrap text-sm font-medium">{{ item.label }}</span>
              <span
                v-if="!collapsed && item.badge === 'alerts' && alertStore.unacknowledgedCount > 0"
                class="flex h-[18px] min-w-[18px] items-center justify-center rounded-full px-1 text-xs font-bold text-white"
                style="background: #EF4444;"
              >
                {{ alertStore.unacknowledgedCount > 99 ? '99+' : alertStore.unacknowledgedCount }}
              </span>
              <span
                v-if="collapsed && item.badge === 'alerts' && alertStore.unacknowledgedCount > 0"
                class="absolute right-1 top-1 h-2 w-2 rounded-full"
                style="background: #EF4444;"
              />
            </router-link>
          </template>
        </div>
      </section>
    </nav>

    <div class="sidebar-footer flex-shrink-0">
      <button
        class="collapse-btn flex w-full items-center transition-colors"
        :class="collapsed ? 'justify-center h-12' : 'gap-3 px-4 py-3'"
        :title="collapsed ? '展开菜单' : '折叠菜单'"
        @click="toggleSidebar"
      >
        <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" class="h-[18px] w-[18px] flex-shrink-0" />
        <span v-if="!collapsed" class="whitespace-nowrap text-sm font-medium">折叠菜单</span>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border-color);
}

.sidebar-logo,
.sidebar-footer {
  border-bottom: 1px solid transparent;
}

.sidebar-logo {
  border-bottom-color: var(--border-color);
}

.sidebar-title {
  color: var(--text-primary);
}

.sidebar-subtitle {
  color: var(--accent-primary);
}

.sidebar-status {
  background: var(--bg-tertiary);
}

.sidebar-status-text {
  color: var(--text-secondary);
}

.sidebar-version,
.sidebar-nav-label {
  color: var(--text-muted);
}

.nav-item {
  border-left: 2px solid transparent;
  transition: background-color 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.nav-item--active {
  background: var(--bg-tertiary);
  border-left-color: var(--accent-primary);
  color: var(--text-primary);
}

.nav-item--inactive {
  color: var(--text-secondary);
}

.nav-item--inactive:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.nav-item--parent {
  border-left-width: 2px;
}

.child-active {
  background: rgba(14, 165, 233, 0.08);
}

.nav-icon--active {
  color: var(--accent-primary);
}

.nav-icon--inactive {
  color: var(--text-muted);
}

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
</style>
