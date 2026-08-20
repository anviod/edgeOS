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
  tilt?: boolean
  label?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  width: 120,
  depth: 44,
  height: 8,
  color: '#0EA5E9',
  tilt: true,
  label: '',
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const topColor = computed(() => `${props.color}2e`)
const frontColor = computed(() => `${props.color}55`)
const sideColor = computed(() => `${props.color}77`)

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '光伏组件',
  accent: props.color,
  badge: props.tilt ? '受光倾斜' : '平置',
  rows: [
    { label: '尺寸', value: `${props.width}×${props.depth}` },
    { label: '倾角', value: props.tilt ? '10°' : '0°' },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="iso-solar"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--sh': `${height}px`,
      '--tilt': tilt ? '10deg' : '0deg',
      '--w': `${width}px`,
      '--d': `${depth}px`,
    }"
  >
    <div class="sface s-roof" :style="{ background: topColor }" />
    <div class="sface s-front" :style="{ background: frontColor }" />
    <div class="sface s-side" :style="{ background: sideColor }" />
    <div v-if="label" class="s-label">{{ label }}</div>
  </div>
</template>

<style scoped>
.iso-solar {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.sface {
  position: absolute;
}

.s-roof {
  inset: 0;
  transform: translateZ(var(--sh)) rotateX(var(--tilt));
  transform-origin: center bottom;
  border: 1px solid rgba(255, 255, 255, 0.18);
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.12) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.12) 1px, transparent 1px);
  background-size: 12px 12px;
}

.s-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--sh);
  transform-origin: center top;
  transform: rotateX(90deg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.s-side {
  left: 0;
  top: 0;
  width: var(--sh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

.s-label {
  position: absolute;
  left: 50%;
  bottom: 2px;
  transform: translateX(-50%);
  font-size: 8px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--scene-label);
  text-shadow: 0 1px 4px var(--scene-label-glow);
  white-space: nowrap;
  pointer-events: none;
}
</style>
