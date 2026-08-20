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
  jibLen?: number
  backLen?: number
  height?: number
  depth?: number
  color?: string
  trolleyDur?: number
  label?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  jibLen: 150,
  backLen: 34,
  height: 110,
  depth: 16,
  color: '#F59E0B',
  trolleyDur: 9,
  label: '',
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const cTop = computed(() => `${props.color}55`)
const cFront = computed(() => `${props.color}88`)
const cSide = computed(() => `${props.color}aa`)
const legColor = computed(() => 'rgba(30, 41, 59, 0.9)')

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '岸桥',
  accent: props.color,
  badge: '装卸作业',
  rows: [
    { label: '前伸距', value: `${props.jibLen}` },
    { label: '后伸距', value: `${props.backLen}` },
    { label: '总高', value: `${props.height}` },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="qc"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${jibLen + backLen}px`,
      height: `${depth}px`,
      '--qc-jib': `${jibLen}px`,
      '--qc-back': `${backLen}px`,
      '--qc-h': `${height}px`,
      '--qc-d': `${depth}px`,
      '--qc-td': `${trolleyDur}s`,
    }"
  >
    <!-- 行走轨道 -->
    <IsoCube :x="-3" :y="-4" :width="backLen" :depth="4" :height="4"
      top-color="rgba(71,85,105,0.5)" front-color="rgba(71,85,105,0.7)" side-color="rgba(71,85,105,0.8)" :hud="hudData" />

    <!-- 门架塔身 -->
    <IsoCube :x="-3" :y="0" :width="6" :depth="depth" :height="height"
      :top-color="legColor" :front-color="legColor" :side-color="legColor" :hud="hudData" />
    <IsoCube :x="backLen - 3" :y="0" :width="6" :depth="depth" :height="height"
      :top-color="legColor" :front-color="legColor" :side-color="legColor" :hud="hudData" />

    <!-- 顶部横梁 -->
    <IsoCube :x="-3" :y="0" :width="backLen" :depth="depth" :height="6" :z="height"
      :top-color="cTop" :front-color="cFront" :side-color="cSide" :hud="hudData" />

    <!-- 前臂悬梁（伸向水域） -->
    <IsoCube :x="-jibLen" :y="0" :width="jibLen" :depth="depth" :height="6" :z="height"
      :top-color="cTop" :front-color="cFront" :side-color="cSide"
      :beacon="true" beacon-color="#EF4444" :hud="hudData" />

    <!-- 平衡重 -->
    <IsoCube :x="backLen - 16" :y="0" :width="12" :depth="depth" :height="20" :z="height"
      top-color="rgba(100,116,139,0.7)" front-color="rgba(100,116,139,0.85)" side-color="rgba(71,85,105,0.9)" :hud="hudData" />

    <!-- 小车 + 吊具 -->
    <div class="qc-trolley">
      <div class="qc-cable" />
      <div class="qc-hoist">
        <div class="h-face h-roof" :style="{ background: cTop }" />
        <div class="h-face h-front" :style="{ background: cFront }" />
        <div class="h-face h-side" :style="{ background: cSide }" />
      </div>
      <span class="qc-light" />
    </div>

    <div v-if="label" class="qc-label">{{ label }}</div>
  </div>
</template>

<style scoped>
.qc {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.qc-trolley {
  position: absolute;
  left: 0;
  top: calc((var(--qc-d) - 12px) / 2);
  width: 12px;
  height: 12px;
  transform-style: preserve-3d;
  transform: translateZ(var(--qc-h));
  animation: qc-trolley-move var(--qc-td) ease-in-out infinite alternate;
}

@keyframes qc-trolley-move {
  from {
    left: 0;
  }
  to {
    left: calc(-1 * var(--qc-jib) + 12px);
  }
}

.qc-cable {
  position: absolute;
  left: 5px;
  top: 6px;
  width: 2px;
  height: calc(var(--qc-h) - 26px);
  background: linear-gradient(180deg, rgba(226, 232, 240, 0.8), rgba(226, 232, 240, 0.4));
  transform: translateZ(calc(-1 * (var(--qc-h) - 26px) / 2 - 4px));
}

.qc-hoist {
  position: absolute;
  left: 1px;
  top: 1px;
  width: 10px;
  height: 10px;
  transform-style: preserve-3d;
  transform: translateZ(calc(-1 * (var(--qc-h) - 26px)));
}

.h-face {
  position: absolute;
}

.h-roof {
  inset: 0;
  transform: translateZ(16px);
  border: 1px solid rgba(255, 255, 255, 0.5);
}

.h-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: 16px;
  transform-origin: center top;
  transform: rotateX(90deg);
}

.h-side {
  left: 0;
  top: 0;
  width: 16px;
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

.qc-light {
  position: absolute;
  right: -4px;
  top: -4px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #EF4444;
  box-shadow: 0 0 8px #EF4444;
  transform: translateZ(2px);
  animation: qc-blink 1.1s ease-in-out infinite;
}

.qc-label {
  position: absolute;
  left: 50%;
  bottom: -14px;
  transform: translateX(-50%);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.1em;
  color: var(--scene-label);
  text-shadow: 0 1px 4px var(--scene-label-glow);
  white-space: nowrap;
  pointer-events: none;
}

@keyframes qc-blink {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.25;
  }
}
</style>
