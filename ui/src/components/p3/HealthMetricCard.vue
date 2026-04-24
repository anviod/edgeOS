<script setup lang="ts">
import StatusIndicator from '@/components/edge/StatusIndicator.vue'
import type { P3HealthMetric } from '@/types/p3'

defineProps<{
  item: P3HealthMetric
}>()
</script>

<template>
  <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <div class="flex items-center justify-between gap-3">
      <div>
        <p class="text-xs uppercase tracking-[0.18em]" style="color: var(--text-muted);">{{ item.label }}</p>
        <div class="mt-2 flex items-baseline gap-1">
          <span class="text-2xl font-semibold font-mono-num" style="color: var(--text-primary);">{{ item.value }}</span>
          <span class="text-sm" style="color: var(--text-secondary);">{{ item.unit }}</span>
        </div>
      </div>
      <StatusIndicator :status="item.status" />
    </div>
    <div class="mt-3 flex items-center justify-between gap-3 text-xs">
      <span style="color: var(--text-secondary);">{{ item.hint }}</span>
      <span :style="{ color: item.trend !== undefined && item.trend > 0 ? '#10B981' : '#F59E0B' }">
        {{ item.trend !== undefined ? `${item.trend > 0 ? '+' : ''}${item.trend}%` : '—' }}
      </span>
    </div>
  </article>
</template>
