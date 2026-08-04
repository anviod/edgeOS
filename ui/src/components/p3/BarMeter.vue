<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  value: number
  max?: number
  color?: string
  height?: number
}>(), {
  max: 100,
  color: '#0EA5E9',
  height: 8,
})

const percent = computed(() => Math.max(0, Math.min(100, (props.value / props.max) * 100)))
</script>

<template>
  <div
    class="relative w-full overflow-hidden rounded-full"
    :style="{
      height: `${height}px`,
      background: 'var(--bg-tertiary)',
      border: '1px solid var(--border-color)',
    }"
  >
    <div
      class="absolute inset-y-0 left-0 rounded-full"
      :style="{
        width: `${percent}%`,
        background: `linear-gradient(90deg, ${color}88, ${color})`,
        boxShadow: `0 0 8px ${color}66`,
        transition: 'width 0.8s cubic-bezier(0.4, 0, 0.2, 1)',
      }"
    />
  </div>
</template>
