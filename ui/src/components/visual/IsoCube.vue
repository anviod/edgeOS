<script setup lang="ts">
import { computed } from 'vue'
import type { IsoHudData } from '@/composables/useIsoHud'

const props = withDefaults(defineProps<{
  col?: number
  row?: number
  x?: number
  y?: number
  cell?: number
  z?: number
  width?: number
  depth?: number
  height?: number
  topColor?: string
  frontColor?: string
  sideColor?: string
  glow?: boolean
  beacon?: boolean
  beaconColor?: string
  label?: string
  labelColor?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  z: 0,
  width: 56,
  depth: 56,
  height: 40,
  topColor: 'rgba(14, 165, 233, 0.22)',
  frontColor: 'rgba(14, 165, 233, 0.42)',
  sideColor: 'rgba(7, 108, 168, 0.55)',
  glow: false,
  beacon: false,
  beaconColor: '#F59E0B',
  label: '',
  labelColor: 'var(--scene-label)',
  hud: undefined,
})

const left = computed(() => (props.x || props.col * props.cell))
const top = computed(() => (props.y || props.row * props.cell))

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '等距设备',
  accent: props.frontColor,
  badge: props.glow ? '发光' : props.beacon ? '信标' : undefined,
  rows: [
    { label: '尺寸', value: `${props.width}×${props.depth}×${props.height}` },
    { label: '抬升', value: `z ${props.z}` },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="iso-cube"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--h': `${height}px`,
      '--w': `${width}px`,
      transform: `translateZ(${z}px)`,
    }"
  >
    <div class="iso-face iso-face-roof" :style="{ background: topColor }" />
    <div
      class="iso-face iso-face-front"
      :style="{
        background: frontColor,
        boxShadow: glow ? `0 0 26px ${frontColor}` : 'none',
      }"
    />
    <div
      class="iso-face iso-face-side"
      :style="{
        background: sideColor,
        boxShadow: glow ? `0 0 18px ${sideColor}` : 'none',
      }"
    />
    <div v-if="beacon" class="iso-beacon" :style="{ background: beaconColor, boxShadow: `0 0 10px ${beaconColor}` }" />
    <div v-if="label" class="iso-label" :style="{ color: labelColor }">{{ label }}</div>
  </div>
</template>

<style scoped>
.iso-cube {
  position: absolute;
  transform-style: preserve-3d;
  transform-origin: center center;
  cursor: help;
}

.iso-face {
  position: absolute;
}

.iso-face-roof {
  inset: 0;
  transform: translateZ(var(--h));
  border: 1px solid rgba(255, 255, 255, 0.16);
  box-shadow: inset 0 0 18px rgba(255, 255, 255, 0.06);
}

.iso-face-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--h);
  transform-origin: center top;
  transform: rotateX(90deg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.iso-face-side {
  left: 0;
  top: 0;
  width: var(--h);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
  border-right: 1px solid rgba(255, 255, 255, 0.08);
}

.iso-label {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  transform: translateZ(calc(var(--h) + 3px));
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.08em;
  white-space: nowrap;
  text-shadow: 0 1px 4px var(--scene-label-glow);
  pointer-events: none;
}

.iso-beacon {
  position: absolute;
  right: -5px;
  top: -5px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  transform: translateZ(calc(var(--h) + 1px));
  animation: iso-beacon-blink 1s ease-in-out infinite;
  pointer-events: none;
}

@keyframes iso-beacon-blink {
  0%, 100% {
    opacity: 1;
    transform: translateZ(calc(var(--h) + 1px)) scale(1);
  }
  50% {
    opacity: 0.35;
    transform: translateZ(calc(var(--h) + 1px)) scale(0.7);
  }
}
</style>
