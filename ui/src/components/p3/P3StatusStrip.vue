<script setup lang="ts">
import type { P3StatusStrip } from '@/types/p3'

defineProps<{
  data: P3StatusStrip
  compact?: boolean
}>()
</script>

<template>
  <div
    class="grid gap-2"
    :class="compact ? 'grid-cols-3 xl:grid-cols-6' : 'grid-cols-2 md:grid-cols-3 xl:grid-cols-6'"
  >
    <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
      <div class="text-[11px] uppercase tracking-[0.16em]" style="color: var(--text-muted);">System</div>
      <div class="mt-1 font-mono-num text-sm" style="color: var(--text-primary);">{{ data.system }}</div>
    </div>
    <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
      <div class="text-[11px] uppercase tracking-[0.16em]" style="color: var(--text-muted);">Devices</div>
      <div class="mt-1 font-mono-num text-sm" style="color: var(--text-primary);">{{ data.devices }}</div>
    </div>
    <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
      <div class="text-[11px] uppercase tracking-[0.16em]" style="color: var(--text-muted);">Alarm</div>
      <div class="mt-1 font-mono-num text-sm" style="color: var(--text-primary);">{{ data.alarms }}</div>
    </div>
    <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
      <div class="text-[11px] uppercase tracking-[0.16em]" style="color: var(--text-muted);">Latency</div>
      <div class="mt-1 font-mono-num text-sm" style="color: var(--text-primary);">{{ data.latency }}ms</div>
    </div>
    <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
      <div class="text-[11px] uppercase tracking-[0.16em]" style="color: var(--text-muted);">Loss</div>
      <div class="mt-1 font-mono-num text-sm" style="color: var(--text-primary);">{{ data.loss }}%</div>
    </div>
    <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
      <div class="text-[11px] uppercase tracking-[0.16em]" style="color: var(--text-muted);">Quality</div>
      <div class="mt-1 font-mono-num text-sm" style="color: var(--text-primary);">{{ data.quality }}</div>
    </div>
  </div>
</template>
