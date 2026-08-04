<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  value: number
  size?: number
  stroke?: number
  color?: string
  track?: string
  label?: string
  sublabel?: string
  animated?: boolean
}>(), {
  size: 168,
  stroke: 14,
  color: '#10B981',
  track: 'rgba(148,163,184,0.16)',
  label: '',
  sublabel: '',
  animated: true,
})

const radius = computed(() => (props.size - props.stroke) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)
const progress = computed(() => Math.max(0, Math.min(100, props.value)))
const offset = computed(() => circumference.value * (1 - progress.value / 100))

const color = computed(() => {
  if (props.color) return props.color
  if (props.value > 60) return '#10B981'
  if (props.value > 30) return '#F59E0B'
  return '#EF4444'
})
</script>

<template>
  <div class="relative inline-flex items-center justify-center" :style="{ width: `${size}px`, height: `${size}px` }">
    <svg
      :width="size"
      :height="size"
      class="-rotate-90"
      :style="{ filter: `drop-shadow(0 0 10px ${color}55)` }"
    >
      <circle
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        fill="none"
        :stroke="track"
        :stroke-width="stroke"
      />
      <circle
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        fill="none"
        :stroke="color"
        :stroke-width="stroke"
        stroke-linecap="round"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="offset"
        :style="{
          transition: animated ? 'stroke-dashoffset 0.9s cubic-bezier(0.4, 0, 0.2, 1), stroke 0.9s ease' : 'none',
        }"
      />
    </svg>
    <div class="absolute inset-0 flex flex-col items-center justify-center text-center">
      <slot />
      <span v-if="label" class="mt-0.5 text-[11px] tracking-[0.16em]" style="color: var(--text-muted);">{{ label }}</span>
      <span v-if="sublabel" class="text-[10px]" style="color: var(--text-secondary);">{{ sublabel }}</span>
    </div>
  </div>
</template>
