<script setup lang="ts">
import type { P3SideEvent } from '@/types/p3'

defineProps<{
  title: string
  events: P3SideEvent[]
  dense?: boolean
}>()

function dotColor(status: string) {
  if (status === 'error') return '#EF4444'
  if (status === 'warning' || status === 'active') return '#F59E0B'
  if (status === 'resolved') return '#10B981'
  return '#0EA5E9'
}
</script>

<template>
  <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ title }}</h3>
    <div class="mt-4 space-y-3">
      <article
        v-for="event in events"
        :key="`${event.title}-${event.meta}`"
        class="rounded-lg border px-3 py-3"
        :class="dense ? 'space-y-1.5' : 'space-y-2'"
        style="background: var(--bg-tertiary); border-color: var(--border-color);"
      >
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <span class="h-2.5 w-2.5 rounded-full" :style="{ background: dotColor(event.status) }" />
            <span class="text-sm font-medium" style="color: var(--text-primary);">{{ event.title }}</span>
          </div>
          <span class="text-[11px] uppercase" style="color: var(--text-muted);">{{ event.status }}</span>
        </div>
        <p class="text-xs" style="color: var(--text-secondary);">{{ event.subtitle }}</p>
        <p class="text-[11px]" style="color: var(--text-muted);">{{ event.meta }}</p>
      </article>
    </div>
  </section>
</template>
