<script setup lang="ts">
import { computed } from 'vue'
import { useLiveClock } from '@/composables/useLiveClock'
import { rebaseTimestamps } from '@/lib/live-time'
import type { P3TimelineItem } from '@/types/p3'

const props = defineProps<{
  title: string
  items: P3TimelineItem[]
}>()

const { time } = useLiveClock()

const liveItems = computed(() => rebaseTimestamps(props.items, 15))
</script>

<template>
  <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <div class="flex items-center justify-between">
      <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ title }}</h3>
      <span class="font-mono-num text-[11px]" style="color: var(--text-muted);">{{ time }}</span>
    </div>
    <div class="mt-4 space-y-4">
      <article v-for="item in liveItems" :key="`${item.timestamp}-${item.title}`" class="relative pl-4">
        <span
          class="absolute left-0 top-1.5 h-2.5 w-2.5 rounded-full"
          :style="{ background: 'var(--accent-primary)', boxShadow: '0 0 6px rgba(14,165,233,0.6)' }"
        />
        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ item.title }}</div>
        <div class="mt-1 text-xs" style="color: var(--text-secondary);">{{ item.detail }}</div>
        <div class="mt-1 flex items-center justify-between text-[11px]" style="color: var(--text-muted);">
          <span>{{ item.timestamp }}</span>
          <span>{{ item.result }}</span>
        </div>
      </article>
    </div>
  </section>
</template>
