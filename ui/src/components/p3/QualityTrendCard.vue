<script setup lang="ts">
import { computed } from 'vue'
import StatusIndicator from '@/components/edge/StatusIndicator.vue'
import type { P3TrendCardData } from '@/types/p3'

const props = defineProps<{
  item: P3TrendCardData
}>()

const normalized = computed(() => {
  const max = Math.max(...props.item.points, 1)
  return props.item.points.map(point => `${Math.max((point / max) * 100, 8)}%`)
})
</script>

<template>
  <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ item.title }}</h3>
        <p class="mt-1 text-xs" style="color: var(--text-secondary);">{{ item.summary }}</p>
      </div>
      <StatusIndicator :status="item.status" />
    </div>
    <div class="mt-4 flex items-end gap-1.5">
      <div
        v-for="(height, index) in normalized"
        :key="`${item.title}-${index}`"
        class="flex-1 rounded-sm"
        :style="{
          height,
          minHeight: '10px',
          background: 'rgba(14,165,233,0.22)',
          border: '1px solid rgba(14,165,233,0.2)',
        }"
      />
    </div>
    <div class="mt-4 grid grid-cols-3 gap-2 text-xs">
      <div class="rounded-lg px-2.5 py-2" style="background: var(--bg-tertiary);">
        <div style="color: var(--text-muted);">Latency</div>
        <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ item.latency }}ms</div>
      </div>
      <div class="rounded-lg px-2.5 py-2" style="background: var(--bg-tertiary);">
        <div style="color: var(--text-muted);">Loss</div>
        <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ item.loss }}%</div>
      </div>
      <div class="rounded-lg px-2.5 py-2" style="background: var(--bg-tertiary);">
        <div style="color: var(--text-muted);">Quality</div>
        <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ item.quality }}</div>
      </div>
    </div>
  </article>
</template>
