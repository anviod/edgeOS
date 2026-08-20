<script setup lang="ts">
import { computed } from 'vue'
import type { IsoHudData } from '@/composables/useIsoHud'

const props = withDefaults(defineProps<{
  col?: number
  row?: number
  x?: number
  y?: number
  cell?: number
  width?: number
  depth?: number
  height?: number
  color?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  width: 200,
  depth: 368,
  height: 4,
  color: '#0C4A6E',
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

// 将 hex 颜色按比例调暗
function shade(hex: string, factor: number) {
  const h = hex.replace('#', '')
  const num = parseInt(h.length === 3 ? h.split('').map(c => c + c).join('') : h, 16)
  const r = Math.round(((num >> 16) & 0xff) * factor)
  const g = Math.round(((num >> 8) & 0xff) * factor)
  const b = Math.round((num & 0xff) * factor)
  return `rgb(${r},${g},${b})`
}

const frontColor = computed(() => shade(props.color, 0.82))
const sideColor = computed(() => shade(props.color, 0.7))

const hudData = computed<IsoHudData>(() => ({
  title: '水域',
  accent: '#38BDF8',
  badge: '泊位水域',
  rows: [
    { label: '范围', value: `${props.width}×${props.depth}` },
    { label: '水深', value: `${props.height}` },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="iso-water"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--wh': `${height}px`,
      '--wc': color,
    }"
  >
    <div class="wface w-roof" />
    <div class="wface w-front" :style="{ background: frontColor }" />
    <div class="wface w-side" :style="{ background: sideColor }" />
  </div>
</template>

<style scoped>
.iso-water {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.wface {
  position: absolute;
}

.w-roof {
  inset: 0;
  transform: translateZ(var(--wh));
  background:
    linear-gradient(180deg, rgba(125, 211, 252, 0.22) 1px, transparent 2px),
    linear-gradient(90deg, rgba(125, 211, 252, 0.1) 1px, transparent 2px),
    var(--wc);
  background-size: 100% 14px, 28px 100%, 100% 100%;
  border: 1px solid rgba(125, 211, 252, 0.2);
  animation: water-flow 7s linear infinite;
}

.w-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--wh);
  transform-origin: center top;
  transform: rotateX(90deg);
  border-bottom: 1px solid rgba(125, 211, 252, 0.15);
}

.w-side {
  left: 0;
  top: 0;
  width: var(--wh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

@keyframes water-flow {
  0% {
    background-position: 0 0, 0 0, 0 0;
  }
  100% {
    background-position: 0 28px, 28px 0, 0 0;
  }
}
</style>
