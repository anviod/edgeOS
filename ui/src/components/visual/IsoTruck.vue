<script setup lang="ts">
import { computed } from 'vue'
import IsoCube from './IsoCube.vue'
import IsoContainer from './IsoContainer.vue'
import type { IsoHudData } from '@/composables/useIsoHud'

const props = withDefaults(defineProps<{
  col?: number
  row?: number
  x?: number
  y?: number
  cell?: number
  width?: number
  depth?: number
  z?: number
  color?: string
  containerColor?: string
  hasContainer?: boolean
  label?: string
  moving?: boolean
  moveDur?: number
  moveFrom?: number
  moveRange?: number
  moveDelay?: number
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  width: 64,
  depth: 20,
  z: 0,
  color: '#0EA5E9',
  containerColor: '#DC2626',
  hasContainer: true,
  label: '',
  moving: false,
  moveDur: 12,
  moveFrom: 200,
  moveRange: 520,
  moveDelay: 0,
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const cabColor = computed(() => `${props.color}cc`)
const cabRoof = computed(() => `${props.color}99`)

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '集卡',
  accent: props.hasContainer ? props.containerColor : props.color,
  badge: props.hasContainer ? '载箱' : '空载',
  rows: [
    { label: '车长', value: `${props.width}` },
    { label: '拖挂板', value: props.hasContainer ? '有箱' : '无箱' },
    { label: '作业', value: props.moving ? '行驶中' : '停驻' },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="iso-truck"
    :class="{ 'iso-truck--moving': moving }"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--td': `${moveDur}s`,
      '--tf': `${moveFrom}px`,
      '--tr': `${moveRange}px`,
      '--tly': `${moveDelay}s`,
      transform: `translateZ(${z}px)`,
    }"
  >
    <!-- 拖车挂板 -->
    <IsoCube :x="16" :y="4" :width="width - 22" :depth="depth - 8" :height="8"
      top-color="rgba(71,85,105,0.6)" front-color="rgba(71,85,105,0.8)" side-color="rgba(71,85,105,0.9)" />

    <!-- 集装箱 -->
    <IsoContainer v-if="hasContainer" :x="24" :y="6" :z="8" :width="30" :depth="14" :height="14" :color="containerColor" :hud="hudData" />

    <!-- 车头 -->
    <IsoCube :x="0" :y="4" :width="14" :depth="depth - 8" :height="16"
      :top-color="cabRoof" :front-color="cabColor" :side-color="cabColor"
      :beacon="true" beacon-color="#F8FAFC" :hud="hudData" />

    <!-- 车轮 -->
    <IsoCube :x="2" :y="2" :width="6" :depth="4" :height="4" top-color="#0F172A" front-color="#0F172A" side-color="#0F172A" :hud="hudData" />
    <IsoCube :x="2" :y="depth - 6" :width="6" :depth="4" :height="4" top-color="#0F172A" front-color="#0F172A" side-color="#0F172A" :hud="hudData" />
    <IsoCube :x="width - 12" :y="2" :width="6" :depth="4" :height="4" top-color="#0F172A" front-color="#0F172A" side-color="#0F172A" :hud="hudData" />
    <IsoCube :x="width - 12" :y="depth - 6" :width="6" :depth="4" :height="4" top-color="#0F172A" front-color="#0F172A" side-color="#0F172A" :hud="hudData" />

    <div v-if="label" class="truck-label">{{ label }}</div>
  </div>
</template>

<style scoped>
.iso-truck {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.iso-truck--moving {
  animation: truck-move var(--td) linear infinite;
  animation-delay: var(--tly);
}

@keyframes truck-move {
  from {
    left: var(--tf);
  }
  to {
    left: var(--tr);
  }
}

.truck-label {
  position: absolute;
  left: 50%;
  bottom: -14px;
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
