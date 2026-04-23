<script setup lang="ts">
import { computed } from 'vue'
import { Radio, Trash2, Edit3, PlugZap, RefreshCw } from 'lucide-vue-next'
import StatusBadge from './StatusBadge.vue'
import type { MiddlewareConfig } from '@/types/edgex'

const props = defineProps<{
  middleware: MiddlewareConfig
  connecting?: boolean
}>()

const emit = defineEmits<{
  (e: 'connect', id: string): void
  (e: 'edit', item: MiddlewareConfig): void
  (e: 'delete', id: string): void
}>()

const address = computed(() => `${props.middleware.host}:${props.middleware.port}`)

const isConnected = computed(() => props.middleware.status === 'connected')
</script>

<template>
  <div
    class="relative rounded-xl p-4 flex flex-col gap-3 transition-all duration-200 hover:scale-[1.01]"
    :style="{
      background: isConnected ? 'rgba(16,185,129,0.04)' : 'var(--bg-secondary)',
      border: isConnected
        ? '1px solid rgba(16,185,129,0.25)'
        : '1px solid var(--border-color)',
    }"
  >
    <!-- Header -->
    <div class="flex items-start justify-between gap-2">
      <div class="flex items-center gap-2 min-w-0">
        <div
          class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
          :style="{ background: isConnected ? 'rgba(16,185,129,0.15)' : 'rgba(14,165,233,0.1)' }"
        >
          <Radio class="w-4 h-4" :style="{ color: isConnected ? '#10B981' : '#0EA5E9', width:'16px', height:'16px' }" />
        </div>
        <div class="min-w-0">
          <div class="text-sm font-semibold truncate" style="color: var(--text-primary);">{{ middleware.name }}</div>
          <div class="text-xs uppercase tracking-wide" style="color: var(--text-secondary);">{{ middleware.type }}</div>
        </div>
      </div>
      <StatusBadge :status="middleware.status" size="sm" />
    </div>

    <!-- Info rows -->
    <div class="space-y-1.5 text-xs" style="color: var(--text-secondary);">
      <div class="flex items-center justify-between">
        <span>地址</span>
        <span class="font-mono" style="color: var(--text-primary);">{{ address }}</span>
      </div>
      <div class="flex items-center justify-between">
        <span>Client ID</span>
        <span class="font-mono truncate max-w-[120px]" style="color: var(--text-primary);">{{ middleware.client_id || '—' }}</span>
      </div>
      <div class="flex items-center justify-between">
        <span>订阅主题</span>
        <span style="color: var(--accent-primary);">{{ middleware.topics?.length ?? 0 }} 个</span>
      </div>
    </div>

    <!-- Error tip -->
    <div
      v-if="middleware.last_error"
      class="rounded-lg px-3 py-2 text-xs truncate"
      style="background: rgba(239,68,68,0.08); color: #EF4444; border: 1px solid rgba(239,68,68,0.2);"
    >
      {{ middleware.last_error }}
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2 pt-1" style="border-top: 1px solid var(--border-color);">
      <button
        @click="emit('connect', middleware.id)"
        :disabled="connecting || isConnected"
        class="flex-1 flex items-center justify-center gap-1.5 rounded-lg py-1.5 text-xs font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :style="isConnected
          ? 'background: rgba(16,185,129,0.1); color:#10B981;'
          : 'background: rgba(14,165,233,0.1); color:#0EA5E9;'"
      >
        <RefreshCw v-if="connecting" class="w-3 h-3 animate-spin" style="width:12px;height:12px;" />
        <PlugZap v-else class="w-3 h-3" style="width:12px;height:12px;" />
        {{ isConnected ? '已连接' : '连接' }}
      </button>
      <button
        @click="emit('edit', middleware)"
        class="flex items-center justify-center w-8 h-8 rounded-lg transition-colors hover:bg-white/5"
      >
        <Edit3 class="w-3.5 h-3.5" style="color: var(--text-secondary); width:14px;height:14px;" />
      </button>
      <button
        @click="emit('delete', middleware.id)"
        class="flex items-center justify-center w-8 h-8 rounded-lg transition-colors hover:bg-red-500/10"
      >
        <Trash2 class="w-3.5 h-3.5" style="color: var(--destructive); width:14px;height:14px;" />
      </button>
    </div>
  </div>
</template>
