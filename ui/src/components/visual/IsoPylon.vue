<script setup lang="ts">
import { computed } from 'vue'
import IsoCube from './IsoCube.vue'
import type { IsoHudData } from '@/composables/useIsoHud'

const props = withDefaults(defineProps<{
  col?: number
  row?: number
  x?: number
  y?: number
  cell?: number
  height?: number
  color?: string
  beacon?: boolean
  label?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  height: 118,
  color: '#64748B',
  beacon: false,
  label: '',
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '输电铁塔',
  accent: props.color,
  badge: props.beacon ? '信标指示' : undefined,
  rows: [
    { label: '塔高', value: `${props.height}` },
    { label: '横担', value: '上下两层' },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div v-hud="hudData" class="iso-pylon" :style="{ left: `${left}px`, top: `${top}px` }">
    <!-- 塔身 -->
    <IsoCube
      :x="0"
      :y="0"
      :width="12"
      :depth="12"
      :height="height"
      :top-color="`${color}33`"
      :front-color="`${color}66`"
      :side-color="`${color}88`"
      :hud="hudData"
    />
    <!-- 下层横担 -->
    <IsoCube
      :x="-16"
      :y="2"
      :z="Math.round(height * 0.42)"
      :width="44"
      :depth="8"
      :height="7"
      :top-color="`${color}44`"
      :front-color="`${color}77`"
      :side-color="`${color}99`"
      :hud="hudData"
    />
    <!-- 上层横担 -->
    <IsoCube
      :x="-16"
      :y="2"
      :z="Math.round(height * 0.72)"
      :width="44"
      :depth="8"
      :height="7"
      :top-color="`${color}44`"
      :front-color="`${color}77`"
      :side-color="`${color}99`"
      :hud="hudData"
    />
    <!-- 顶部避雷针 / 信标 -->
    <IsoCube
      :x="4"
      :y="4"
      :z="height"
      :width="4"
      :depth="4"
      :height="14"
      :top-color="`${color}55`"
      :front-color="`${color}77`"
      :side-color="`${color}99`"
      :beacon="beacon"
      beacon-color="#EF4444"
      :hud="hudData"
    />
    <div v-if="label" class="pylon-label">{{ label }}</div>
  </div>
</template>

<style scoped>
.iso-pylon {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.pylon-label {
  position: absolute;
  left: 50%;
  bottom: -16px;
  transform: translateX(-50%) translateZ(0);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--scene-label);
  text-shadow: 0 1px 4px var(--scene-label-glow);
  white-space: nowrap;
  pointer-events: none;
}
</style>
