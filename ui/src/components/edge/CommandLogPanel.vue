<script setup lang="ts">
import { computed } from 'vue'
import { RefreshCw, CheckCircle2, XCircle, Clock, RotateCcw, Inbox } from 'lucide-vue-next'
import type { CommandRecord } from '@/types/edgex'

const props = defineProps<{
  commands: CommandRecord[]
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'retry', cmd: CommandRecord): void
  (e: 'clear'): void
}>()

function handleClear() {
  emit('clear')
}

interface StatusStyle {
  icon: unknown
  iconColor: string
  bgColor: string
  borderColor: string
  label: string
}

function getStyle(status: string): StatusStyle {
  const map: Record<string, StatusStyle> = {
    pending: {
      icon: Clock,
      iconColor: '#F59E0B',
      bgColor: 'rgba(245,158,11,0.06)',
      borderColor: 'rgba(245,158,11,0.15)',
      label: '执行中',
    },
    success: {
      icon: CheckCircle2,
      iconColor: '#10B981',
      bgColor: 'rgba(16,185,129,0.06)',
      borderColor: 'rgba(16,185,129,0.15)',
      label: '成功',
    },
    error: {
      icon: XCircle,
      iconColor: '#EF4444',
      bgColor: 'rgba(239,68,68,0.06)',
      borderColor: 'rgba(239,68,68,0.15)',
      label: '失败',
    },
    timeout: {
      icon: XCircle,
      iconColor: '#9CA3AF',
      bgColor: 'rgba(107,114,128,0.06)',
      borderColor: 'rgba(107,114,128,0.15)',
      label: '超时',
    },
  }
  return map[status] || map.timeout
}

function formatTime(ts: number): string {
  if (!ts) return '—'
  return new Date(ts < 1e12 ? ts * 1000 : ts).toLocaleTimeString('zh-CN', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const sorted = computed(() =>
  [...props.commands].sort((a, b) => (b.created_at || 0) - (a.created_at || 0))
)
</script>

<template>
  <div class="flex flex-col h-full rounded-xl overflow-hidden"
    style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
    <!-- Panel header -->
    <div class="flex items-center justify-between px-4 py-3"
      style="border-bottom: 1px solid var(--border-color);">
      <div class="flex items-center gap-2">
        <RefreshCw class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">命令执行记录</span>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-xs px-2 py-0.5 rounded-full" style="background: rgba(14,165,233,0.12); color: var(--accent-primary);">
          {{ commands.length }} 条
        </span>
        <button
          @click="handleClear"
          class="flex items-center gap-1 text-xs rounded px-2 py-0.5 transition-colors hover:bg-sky-500/20"
          style="color: var(--accent-primary);"
        >
          清空记录
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <RefreshCw class="w-5 h-5 animate-spin" style="color: var(--accent-primary);" />
    </div>

    <!-- Empty -->
    <div v-else-if="sorted.length === 0" class="flex-1 flex flex-col items-center justify-center gap-2 py-8">
      <Inbox class="w-8 h-8" style="color: var(--text-muted);" />
      <span class="text-sm" style="color: var(--text-secondary);">暂无命令记录</span>
    </div>

    <!-- List -->
    <div v-else class="flex-1 overflow-y-auto divide-y" style="scrollbar-width:thin; scrollbar-color: rgba(14,165,233,0.2) transparent; divide-color: var(--border-color);">
      <div
        v-for="cmd in sorted"
        :key="cmd.id"
        class="px-4 py-3 transition-colors"
        :style="{
          background: getStyle(cmd.status).bgColor,
          borderLeft: `2px solid ${getStyle(cmd.status).iconColor}`,
        }"
      >
        <div class="flex items-center justify-between gap-3">
          <!-- Left side: device, point, value -->
          <div class="flex items-center gap-2 flex-1 min-w-0">
            <!-- Status icon -->
            <component
              :is="getStyle(cmd.status).icon"
              class="w-4 h-4 flex-shrink-0"
              :class="cmd.status === 'pending' ? 'animate-spin' : ''"
              :style="{ color: getStyle(cmd.status).iconColor, width:'16px', height:'16px' }"
            />
            
            <!-- Device + Point -->
            <div class="flex items-center gap-2 flex-wrap flex-1 min-w-0">
              <span class="text-xs font-mono truncate" style="color: var(--text-primary);">{{ cmd.device_id }}</span>
              <span style="color: var(--text-muted);">›</span>
              <span class="text-xs font-mono truncate" style="color: var(--accent-primary);">{{ cmd.point_id }}</span>
            </div>
            
            <!-- Value -->
            <span class="text-xs font-mono font-semibold" style="color: var(--text-primary);">
              {{ cmd.value !== undefined ? String(cmd.value) : '—' }}
            </span>
          </div>
          
          <!-- Right side: status, time, retry -->
          <div class="flex items-center gap-3">
            <!-- Status label -->
            <span
              class="text-xs px-1.5 py-0.5 rounded"
              :style="{
                background: getStyle(cmd.status).bgColor,
                color: getStyle(cmd.status).iconColor,
                border: `1px solid ${getStyle(cmd.status).borderColor}`,
              }"
            >{{ getStyle(cmd.status).label }}</span>
            
            <!-- Time -->
            <span class="text-xs tabular-nums" style="color: var(--text-secondary);">{{ formatTime(cmd.created_at) }}</span>
            
            <!-- Retry button -->
            <button
              v-if="cmd.status === 'error' || cmd.status === 'timeout'"
              @click="emit('retry', cmd)"
              class="flex items-center gap-1 text-xs rounded px-2 py-0.5 transition-colors hover:bg-sky-500/20"
              style="color: var(--accent-primary);"
            >
              <RotateCcw class="w-3 h-3" style="width:12px;height:12px;" />
              重试
            </button>
          </div>
        </div>
        
        <!-- Error message -->
        <div v-if="cmd.error" class="text-xs truncate mt-1" style="color: var(--destructive);">
          {{ cmd.error }}
        </div>
      </div>
    </div>
  </div>
</template>
