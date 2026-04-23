<script setup lang="ts">
import { computed } from 'vue'
import { ArrowUp, ArrowDown } from 'lucide-vue-next'
import StatusIndicator from './StatusIndicator.vue'

const props = defineProps<{
  label: string
  value: string | number
  unit?: string
  status?: 'running' | 'standby' | 'fault' | 'unknown'
  showStatus?: boolean
  trend?: number
}>()

const trendUp = computed(() => props.trend && props.trend > 0)
const trendDown = computed(() => props.trend && props.trend < 0)
</script>

<template>
  <div class="metric-card border p-4 rounded-xl">
    <div class="flex items-center justify-between">
      <span class="text-xs font-medium uppercase tracking-wide metric-label">
        {{ label }}
      </span>
      <StatusIndicator v-if="showStatus && status" :status="status" />
    </div>
    <div class="mt-2 flex items-baseline gap-1">
      <span class="text-2xl font-semibold metric-value">{{ value }}</span>
      <span v-if="unit" class="text-sm metric-unit">{{ unit }}</span>
    </div>
    <div v-if="trend !== undefined" class="mt-2 flex items-center gap-1">
      <ArrowUp v-if="trendUp" class="h-3 w-3 text-emerald-500" />
      <ArrowDown v-else-if="trendDown" class="h-3 w-3 text-red-500" />
      <span
        v-if="trend !== 0"
        class="text-xs"
        :class="trendUp ? 'text-emerald-600' : 'text-red-600'"
      >
        {{ Math.abs(trend) }}%
      </span>
      <span class="text-xs metric-label">较昨日</span>
    </div>
  </div>
</template>

<style scoped>
.metric-card {
  background: var(--bg-secondary);
  border-color: var(--border-color);
  box-shadow: var(--shadow-card);
  transition: background-color 0.2s ease, border-color 0.2s ease;
}
.metric-label { color: var(--text-muted); }
.metric-value { color: var(--text-primary); }
.metric-unit  { color: var(--text-secondary); }
</style>
