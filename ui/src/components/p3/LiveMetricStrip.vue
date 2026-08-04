<script setup lang="ts">
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'

export interface LiveMetricItem {
  label: string
  value: number
  unit?: string
  decimals?: number
  color?: string
  badge?: string
  pulse?: boolean
}

defineProps<{
  items: LiveMetricItem[]
}>()
</script>

<template>
  <div class="grid gap-2 grid-cols-2 md:grid-cols-3 xl:grid-cols-6">
    <div
      v-for="item in items"
      :key="item.label"
      class="rounded-lg border px-3 py-2"
      :style="{ background: 'var(--bg-tertiary)', borderColor: 'var(--border-color)' }"
    >
      <div class="flex items-center justify-between">
        <span class="text-[11px] uppercase tracking-[0.16em]" style="color: var(--text-muted);">{{ item.label }}</span>
        <span
          v-if="item.pulse"
          class="h-1.5 w-1.5 animate-pulse rounded-full"
          :style="{ background: item.color ?? '#10B981' }"
        />
      </div>
      <div class="mt-1 flex items-baseline gap-1 text-sm">
        <AnimatedNumber
          :value="item.value"
          :decimals="item.decimals ?? 0"
          class="font-mono-num"
          :style="{ color: item.color ?? 'var(--text-primary)', fontSize: '14px', fontWeight: 600 }"
        />
        <span class="text-[11px]" style="color: var(--text-secondary);">{{ item.unit }}</span>
      </div>
      <div v-if="item.badge" class="mt-1">
        <span
          class="rounded-md px-1.5 py-0.5 text-[10px] font-medium"
          :style="{ color: item.color ?? '#0EA5E9', background: `${item.color ?? '#0EA5E9'}1a` }"
        >
          {{ item.badge }}
        </span>
      </div>
    </div>
  </div>
</template>
