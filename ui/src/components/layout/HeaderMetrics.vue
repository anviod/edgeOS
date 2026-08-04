<script setup lang="ts">
import { computed } from 'vue'
import { Cpu, AlertTriangle, Timer, Activity, Gauge } from 'lucide-vue-next'

const props = defineProps<{
  systemStatus?: string
  onlineDevices?: number
  alarmCount?: number
  avgLatency?: number
  packetLoss?: number
  qualityScore?: number
}>()

const statusColor = computed(() => {
  const s = (props.systemStatus || '').toLowerCase()
  if (s === 'running' || s === 'online' || s === 'ok') return '#10B981'
  if (s === 'degraded' || s === 'warning') return '#F59E0B'
  return '#EF4444'
})

const statusPulse = computed(() => statusColor.value === '#10B981')

const latencyColor = computed(() => {
  const v = props.avgLatency ?? 0
  if (v < 30) return '#10B981'
  if (v < 60) return '#F59E0B'
  return '#EF4444'
})

const lossColor = computed(() => {
  const v = props.packetLoss ?? 0
  if (v < 0.3) return '#10B981'
  if (v < 1) return '#F59E0B'
  return '#EF4444'
})

const alarmDanger = computed(() => (props.alarmCount ?? 0) > 0)

const qualityColor = computed(() => {
  const v = props.qualityScore ?? 0
  if (v >= 95) return '#10B981'
  if (v >= 90) return '#F59E0B'
  return '#EF4444'
})
</script>

<template>
  <div class="hidden items-center gap-1.5 xl:flex">

    <!-- Alarms -->
    <div class="hm-pill" title="活跃告警">
      <AlertTriangle class="hm-icon" :style="{ color: alarmDanger ? '#EF4444' : '#9CA3AF' }" />
      <span class="hm-value" :style="{ color: alarmDanger ? '#EF4444' : 'var(--text-primary)' }">{{ alarmCount }}</span>
      <span class="hm-label">告警</span>
    </div>

    <div class="hm-divider" />


    <!-- Quality -->
    <div class="hm-pill hm-quality" title="链路质量">
      <Gauge class="hm-icon" :style="{ color: qualityColor }" />
      <span class="hm-value" :style="{ color: qualityColor }">{{ qualityScore }}</span>
      <span class="hm-label">质量</span>
      <div class="hm-quality-bar">
        <div class="hm-quality-fill" :style="{ width: `${qualityScore}%`, background: qualityColor }" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.hm-pill {
  display: inline-flex;
  align-items: center;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-tertiary);
  white-space: nowrap;
}

.hm-divider {
  width: 1px;
  height: 18px;
  background: var(--border-color);
}

.hm-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.hm-status {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.hm-value {
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  font-weight: 700;
}

.hm-label {
  margin-left: 2px;
  font-size: 10px;
  color: var(--text-muted);
}

.hm-quality {
  position: relative;
  padding-bottom: 8px;
  padding-top: 2px;
  height: 34px;
}

.hm-quality-bar {
  position: absolute;
  left: 8px;
  right: 8px;
  bottom: 4px;
  height: 3px;
  border-radius: 2px;
  background: var(--border-color);
  overflow: hidden;
}

.hm-quality-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.4s ease;
}
</style>
