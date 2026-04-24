<script setup lang="ts">
import { computed } from 'vue'
import type { P3KpiItem, P3Tone } from '@/types/p3'

const props = defineProps<{
  items: P3KpiItem[]
}>()

const toneMap: Record<P3Tone, { border: string; text: string; soft: string }> = {
  sky: { border: 'rgba(14,165,233,0.28)', text: '#0EA5E9', soft: 'rgba(14,165,233,0.08)' },
  emerald: { border: 'rgba(16,185,129,0.28)', text: '#10B981', soft: 'rgba(16,185,129,0.08)' },
  amber: { border: 'rgba(245,158,11,0.28)', text: '#F59E0B', soft: 'rgba(245,158,11,0.08)' },
  red: { border: 'rgba(239,68,68,0.28)', text: '#EF4444', soft: 'rgba(239,68,68,0.08)' },
  violet: { border: 'rgba(139,92,246,0.28)', text: '#8B5CF6', soft: 'rgba(139,92,246,0.08)' },
  slate: { border: 'rgba(100,116,139,0.28)', text: '#64748B', soft: 'rgba(100,116,139,0.08)' },
}

const cards = computed(() =>
  props.items.map(item => ({
    ...item,
    toneStyle: toneMap[item.tone],
  }))
)
</script>

<template>
  <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
    <article
      v-for="item in cards"
      :key="item.label"
      class="rounded-xl border p-4"
      :style="{
        background: 'var(--bg-secondary)',
        borderColor: item.toneStyle.border,
      }"
    >
      <div class="flex items-center justify-between">
        <span class="text-xs uppercase tracking-[0.18em]" style="color: var(--text-muted);">{{ item.label }}</span>
        <span
          class="rounded-lg px-2 py-1 text-[11px] font-medium"
          :style="{ color: item.toneStyle.text, background: item.toneStyle.soft }"
        >
          {{ item.delta !== undefined ? `${item.delta > 0 ? '+' : ''}${item.delta}%` : '静态占位' }}
        </span>
      </div>
      <div class="mt-3 flex items-baseline gap-1">
        <span class="text-3xl font-semibold font-mono-num" :style="{ color: item.toneStyle.text }">{{ item.value }}</span>
        <span class="text-sm" style="color: var(--text-secondary);">{{ item.unit }}</span>
      </div>
      <p class="mt-2 text-xs" style="color: var(--text-secondary);">{{ item.note }}</p>
    </article>
  </div>
</template>
