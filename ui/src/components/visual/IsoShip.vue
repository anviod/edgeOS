<script setup lang="ts">
import { computed } from 'vue'
import IsoContainer from './IsoContainer.vue'
import IsoCube from './IsoCube.vue'
import type { IsoHudData } from '@/composables/useIsoHud'

const props = withDefaults(defineProps<{
  col?: number
  row?: number
  x?: number
  y?: number
  cell?: number
  width?: number
  depth?: number
  hullH?: number
  deckH?: number
  color?: string
  label?: string
  moving?: boolean
  moveDur?: number
  moveFrom?: number
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  width: 150,
  depth: 56,
  hullH: 24,
  deckH: 4,
  color: '#1E293B',
  label: '',
  moving: false,
  moveDur: 30,
  moveFrom: -180,
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const deckY = computed(() => props.hullH + props.deckH)

const bowColor = computed(() => `${props.color}99`)

// 甲板集装箱布置（两列，靠艏）
const deckContainers = computed(() => [
  { x: 34, y: 8, color: '#DC2626' },
  { x: 78, y: 8, color: '#2563EB' },
  { x: 34, y: 30, color: '#16A34A' },
  { x: 78, y: 30, color: '#EA580C' },
])

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '集装箱货轮',
  accent: props.color,
  badge: props.moving ? '进港航行' : '靠泊作业',
  rows: [
    { label: '船长', value: `${props.width}` },
    { label: '船宽', value: `${props.depth}` },
    { label: '吃水 / 甲板', value: `${props.hullH} / ${deckY.value}` },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="iso-ship"
    :class="{ 'iso-ship--moving': moving }"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--hh': `${hullH}px`,
      '--dh': `${deckH}px`,
      '--bd': `${deckY}px`,
      '--sd': `${moveDur}s`,
      '--sf': `${moveFrom}px`,
      '--st': `${left}px`,
    }"
  >
    <!-- 船体 -->
    <div class="sf s-hull-roof" :style="{ background: `${color}bb` }" />
    <div class="sf s-hull-front" :style="{ background: `${color}99` }" />
    <div class="sf s-hull-side" :style="{ background: `${color}77` }" />

    <!-- 艏部装饰 -->
    <div class="sf s-bow" :style="{ background: bowColor }" />

    <!-- 甲板 -->
    <div class="sf s-deck-roof" :style="{ background: 'rgba(148,163,184,0.55)' }" />
    <div class="sf s-deck-front" :style="{ background: 'rgba(148,163,184,0.4)' }" />
    <div class="sf s-deck-side" :style="{ background: 'rgba(148,163,184,0.3)' }" />

    <!-- 舱室（桥楼） -->
    <IsoCube
      :x="6"
      :y="10"
      :width="24"
      :depth="36"
      :height="30"
      :z="deckY"
      top-color="rgba(226,232,240,0.5)"
      front-color="rgba(226,232,240,0.7)"
      side-color="rgba(203,213,225,0.8)"
      :hud="hudData"
    />
    <!-- 烟囱 -->
    <IsoCube
      :x="8"
      :y="16"
      :width="12"
      :depth="20"
      :height="16"
      :z="deckY + 30"
      top-color="rgba(15,23,42,0.8)"
      front-color="rgba(15,23,42,0.9)"
      side-color="rgba(15,23,42,0.7)"
      :hud="hudData"
    />

    <!-- 甲板集装箱 -->
    <IsoContainer
      v-for="(c, i) in deckContainers"
      :key="i"
      :x="c.x"
      :y="c.y"
      :z="deckY"
      :color="c.color"
      :hud="hudData"
    />

    <div v-if="label" class="s-label">{{ label }}</div>
  </div>
</template>

<style scoped>
.iso-ship {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.iso-ship--moving {
  animation: ship-io var(--sd) ease-in-out infinite;
}

@keyframes ship-io {
  0%, 100% {
    left: var(--sf);
  }
  30%, 62% {
    left: var(--st);
  }
  85% {
    left: var(--sf);
  }
}

.sf {
  position: absolute;
}

.s-hull-roof {
  inset: 0;
  transform: translateZ(var(--hh));
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.s-hull-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--hh);
  transform-origin: center top;
  transform: rotateX(90deg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.s-hull-side {
  left: 0;
  top: 0;
  width: var(--hh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
  border-right: 1px solid rgba(255, 255, 255, 0.08);
}

.s-bow {
  position: absolute;
  right: 0;
  top: 20%;
  width: 6px;
  height: 60%;
  transform: translateZ(var(--hh));
}

.s-deck-roof {
  inset: 0;
  transform: translateZ(var(--bd));
  border: 1px solid rgba(226, 232, 240, 0.4);
}

.s-deck-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--dh);
  transform-origin: center top;
  transform: rotateX(90deg);
}

.s-deck-side {
  left: 0;
  top: 0;
  width: var(--dh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

.s-label {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%) translateZ(calc(var(--bd) + 66px));
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  color: var(--scene-label);
  text-shadow: 0 1px 5px var(--scene-label-glow);
  white-space: nowrap;
  pointer-events: none;
}
</style>
