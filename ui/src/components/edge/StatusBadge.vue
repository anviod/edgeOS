<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  status: string
  size?: 'sm' | 'md'
  pulse?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm',
  pulse: false,
})

interface StatusConfig {
  label: string
  dotColor: string
  textColor: string
  bgColor: string
}

const statusMap: Record<string, StatusConfig> = {
  // Node / Middleware
  online:       { label: '在线',   dotColor: '#10B981', textColor: '#10B981', bgColor: 'rgba(16,185,129,0.12)' },
  connected:    { label: '已连接', dotColor: '#10B981', textColor: '#10B981', bgColor: 'rgba(16,185,129,0.12)' },
  offline:      { label: '离线',   dotColor: '#6B7280', textColor: '#9CA3AF', bgColor: 'rgba(107,114,128,0.12)' },
  disconnected: { label: '断开',   dotColor: '#6B7280', textColor: '#9CA3AF', bgColor: 'rgba(107,114,128,0.12)' },
  timeout:      { label: '超时',   dotColor: '#F59E0B', textColor: '#F59E0B', bgColor: 'rgba(245,158,11,0.12)' },
  error:        { label: '错误',   dotColor: '#EF4444', textColor: '#EF4444', bgColor: 'rgba(239,68,68,0.12)' },
  connecting:   { label: '连接中', dotColor: '#0EA5E9', textColor: '#0EA5E9', bgColor: 'rgba(14,165,233,0.12)' },
  // Device Operating State (EdgeX standard)
  UP:           { label: '在线',   dotColor: '#10B981', textColor: '#10B981', bgColor: 'rgba(16,185,129,0.12)' },
  DOWN:         { label: '离线',   dotColor: '#6B7280', textColor: '#9CA3AF', bgColor: 'rgba(107,114,128,0.12)' },
  // Commands
  pending:      { label: '执行中', dotColor: '#F59E0B', textColor: '#F59E0B', bgColor: 'rgba(245,158,11,0.12)' },
  success:      { label: '成功',   dotColor: '#10B981', textColor: '#10B981', bgColor: 'rgba(16,185,129,0.12)' },
  // Alerts
  active:       { label: '活跃',   dotColor: '#EF4444', textColor: '#EF4444', bgColor: 'rgba(239,68,68,0.12)' },
  acknowledged: { label: '已确认', dotColor: '#0EA5E9', textColor: '#0EA5E9', bgColor: 'rgba(14,165,233,0.12)' },
  resolved:     { label: '已解决', dotColor: '#10B981', textColor: '#10B981', bgColor: 'rgba(16,185,129,0.12)' },
}

const config = computed<StatusConfig>(() => {
  return statusMap[props.status] ?? {
    label: props.status,
    dotColor: '#6B7280',
    textColor: '#9CA3AF',
    bgColor: 'rgba(107,114,128,0.12)',
  }
})

const shouldPulse = computed(() =>
  props.pulse || ['online', 'connected', 'connecting', 'active'].includes(props.status)
)
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-full font-medium"
    :style="{
      backgroundColor: config.bgColor,
      color: config.textColor,
      padding: size === 'sm' ? '2px 8px' : '3px 10px',
      fontSize: size === 'sm' ? '11px' : '12px',
    }"
  >
    <span
      class="rounded-full flex-shrink-0"
      :class="shouldPulse ? 'animate-pulse' : ''"
      :style="{
        width: size === 'sm' ? '6px' : '7px',
        height: size === 'sm' ? '6px' : '7px',
        backgroundColor: config.dotColor,
        boxShadow: shouldPulse ? `0 0 6px ${config.dotColor}` : 'none',
      }"
    />
    {{ config.label }}
  </span>
</template>
