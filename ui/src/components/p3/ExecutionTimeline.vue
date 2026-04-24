<script setup lang="ts">
import type { P3TimelineItem } from '@/types/p3'

defineProps<{
  title: string
  items: P3TimelineItem[]
}>()
</script>

<template>
  <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ title }}</h3>
    <div class="mt-4 space-y-4">
      <article v-for="item in items" :key="`${item.timestamp}-${item.title}`" class="relative pl-4">
        <span
          class="absolute left-0 top-1.5 h-2.5 w-2.5 rounded-full"
          style="background: var(--accent-primary);"
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
