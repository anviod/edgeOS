<script setup lang="ts">
import { computed } from 'vue'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const collapsed = computed(() => appStore.sidebarCollapsed)
</script>

<template>
  <div class="app-layout min-h-screen">
    <AppSidebar />
    <AppHeader :sidebar-collapsed="collapsed" />
    <main
      class="main-content mt-14 p-6 min-h-[calc(100vh-56px)] transition-all duration-200"
      :class="collapsed ? 'ml-16' : 'ml-60'"
    >
      <router-view v-slot="{ Component, route }">
        <keep-alive :include="['NodeList', 'DeviceList', 'PointList', 'Alerts']">
          <component :is="Component" :key="route.path" />
        </keep-alive>
      </router-view>
    </main>
  </div>
</template>

<style scoped>
.app-layout {
  background-color: var(--bg-primary);
  color: var(--text-primary);
  transition: background-color 0.2s ease, color 0.2s ease;
}
.main-content {
  background-color: var(--bg-primary);
  border-left: 1px solid var(--border-color);
  border-top: 1px solid var(--border-color);
}
</style>
