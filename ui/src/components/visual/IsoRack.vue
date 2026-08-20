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
  status?: 'normal' | 'hot' | 'fault'
  lights?: number
  label?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  width: 24,
  depth: 40,
  height: 84,
  color: '#0EA5E9',
  status: 'normal',
  lights: 5,
  label: '',
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const statusColor = computed(() => {
  if (props.status === 'fault') return '#EF4444'
  if (props.status === 'hot') return '#F59E0B'
  return '#10B981'
})

const bodyFront = computed(() => `${props.color}66`)
const bodySide = computed(() => `${props.color}88`)
const bodyTop = computed(() => `${props.color}22`)

const lightCols = computed(() => Array.from({ length: props.lights }, (_, i) => i))

const statusText = computed(() => props.status === 'fault' ? '故障' : props.status === 'hot' ? '高热' : '正常')

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '机柜',
  accent: statusColor.value,
  badge: statusText.value,
  rows: [
    { label: '尺寸', value: `${props.width}×${props.depth}×${props.height}` },
    { label: '状态', value: statusText.value, color: statusColor.value },
  ],
  ...(props.hud ?? {}),
}))

function lightColor(i: number) {
  if (props.status === 'fault' && i === 0) return '#EF4444'
  if (props.status === 'hot' && i < 2) return '#F59E0B'
  return '#22C55E'
}
</script>

<template>
  <div
    v-hud="hudData"
    class="iso-rack"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--rh': `${height}px`,
    }"
  >
    <div class="rface r-roof" :style="{ background: bodyTop }" />
    <div
      class="rface r-front"
      :style="{
        background: bodyFront,
        boxShadow: status !== 'normal' ? `0 0 18px ${statusColor}` : 'none',
      }"
    />
    <div class="rface r-side" :style="{ background: bodySide }" />

    <!-- 顶部风扇格栅 -->
    <div class="r-vent" />

    <!-- 状态灯 -->
    <div
      v-for="(n, i) in lightCols"
      :key="i"
      class="r-light"
      :style="{
        left: `${8 + i * (width - 16) / Math.max(1, lights - 1)}px`,
        background: lightColor(i),
        boxShadow: `0 0 6px ${lightColor(i)}`,
      }"
    />

    <div v-if="label" class="r-label">{{ label }}</div>
  </div>
</template>

<style scoped>
.iso-rack {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.rface {
  position: absolute;
}

.r-roof {
  inset: 0;
  transform: translateZ(var(--rh));
  border: 1px solid rgba(255, 255, 255, 0.14);
  box-shadow: inset 0 0 12px rgba(255, 255, 255, 0.05);
}

.r-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--rh);
  transform-origin: center top;
  transform: rotateX(90deg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.r-side {
  left: 0;
  top: 0;
  width: var(--rh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
  border-right: 1px solid rgba(255, 255, 255, 0.08);
}

.r-vent {
  position: absolute;
  left: 4px;
  right: 4px;
  top: 4px;
  height: 10px;
  transform: translateZ(calc(var(--rh) + 0.5px));
  background: repeating-linear-gradient(90deg, rgba(15, 23, 42, 0.6) 0 3px, transparent 3px 6px);
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.r-light {
  position: absolute;
  top: 8px;
  width: 3px;
  height: 3px;
  border-radius: 50%;
  transform: translateZ(calc(var(--rh) + 1px));
  animation: r-light-blink 1.6s ease-in-out infinite;
  pointer-events: none;
}

.r-label {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%) translateZ(calc(var(--rh) + 3px));
  font-size: 8px;
  font-weight: 600;
  letter-spacing: 0.08em;
  white-space: nowrap;
  color: var(--scene-label);
  text-shadow: 0 1px 4px var(--scene-label-glow);
  pointer-events: none;
}

@keyframes r-light-blink {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.35;
  }
}
</style>
