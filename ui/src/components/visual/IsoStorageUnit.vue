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
  controlH?: number
  color?: string
  status?: 'charge' | 'discharge' | 'idle'
  label?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  width: 176,
  depth: 52,
  height: 30,
  controlH: 14,
  color: '#0EA5E9',
  status: 'idle',
  label: '',
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const statusColor = computed(() => {
  if (props.status === 'charge') return '#F59E0B'
  if (props.status === 'discharge') return '#10B981'
  return '#64748B'
})

const active = computed(() => props.status === 'charge' || props.status === 'discharge')

const bodyTop = computed(() => `${props.color}33`)
const bodyFront = computed(() => `${props.color}55`)
const bodySide = computed(() => `${props.color}77`)
const hatchColor = computed(() => `${props.color}99`)

const statusText = computed(() => props.status === 'charge' ? '充电' : props.status === 'discharge' ? '放电' : '待机')

const hudData = computed<IsoHudData>(() => ({
  title: props.label || '储能柜',
  accent: statusColor.value,
  badge: statusText.value,
  rows: [
    { label: '柜体', value: `${props.width}×${props.depth}×${props.height}` },
    { label: '控制柜', value: `${props.controlH} 层` },
    { label: '状态', value: statusText.value, color: statusColor.value },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="isu"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--bh': `${height}px`,
      '--ch': `${controlH}px`,
    }"
  >
    <!-- 箱体 -->
    <div class="isu-face isu-roof" :style="{ background: bodyTop, borderColor: hatchColor }" />
    <div class="isu-face isu-front" :style="{ background: bodyFront, borderColor: `${color}66` }" />
    <div class="isu-face isu-side" :style="{ background: bodySide }" />

    <!-- 舱口装饰 -->
    <div class="isu-hatch" :style="{ borderColor: hatchColor }" />

    <!-- 顶部控制柜 -->
    <div class="isu-ctrl">
      <div class="isu-face isu-ctrl-roof" :style="{ background: 'rgba(30,41,59,0.72)', borderColor: '#475569' }" />
      <div class="isu-face isu-ctrl-front" :style="{ background: 'rgba(30,41,59,0.9)' }" />
      <div class="isu-face isu-ctrl-side" :style="{ background: 'rgba(30,41,59,0.8)' }" />
    </div>

    <!-- 状态灯 -->
    <div
      class="isu-light"
      :style="{
        background: statusColor,
        boxShadow: active ? `0 0 12px ${statusColor}` : 'none',
        animation: active ? 'isu-blink 1.4s ease-in-out infinite' : 'none',
      }"
    />

    <div class="isu-label" :style="{ color: active ? statusColor : 'var(--scene-label)' }">{{ label }}</div>
  </div>
</template>

<style scoped>
.isu {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.isu-face {
  position: absolute;
}

.isu-roof {
  inset: 0;
  transform: translateZ(var(--bh));
  border: 1px solid;
  box-shadow: inset 0 0 20px rgba(255, 255, 255, 0.06);
}

.isu-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--bh);
  transform-origin: center top;
  transform: rotateX(90deg);
  border-bottom: 1px solid;
  background-image: repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.07) 0 6px, transparent 6px 13px);
}

.isu-side {
  left: 0;
  top: 0;
  width: var(--bh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

.isu-hatch {
  position: absolute;
  left: 14px;
  top: 14px;
  width: calc(100% - 28px);
  height: calc(100% - 28px);
  transform: translateZ(calc(var(--bh) + 0.5px));
  border: 1px solid;
  background: rgba(0, 0, 0, 0.14);
}

.isu-ctrl {
  position: absolute;
  inset: 10px;
  transform: translateZ(var(--bh));
  transform-style: preserve-3d;
}

.isu-ctrl-roof {
  inset: 0;
  transform: translateZ(var(--ch));
  border: 1px solid;
}

.isu-ctrl-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--ch);
  transform-origin: center top;
  transform: rotateX(90deg);
}

.isu-ctrl-side {
  left: 0;
  top: 0;
  width: var(--ch);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

.isu-light {
  position: absolute;
  right: 16px;
  top: 16px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  transform: translateZ(calc(var(--bh) + var(--ch) + 2px));
}

.isu-label {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%) translateZ(calc(var(--bh) + var(--ch) + 5px));
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  white-space: nowrap;
  text-shadow: 0 1px 4px var(--scene-label-glow);
  pointer-events: none;
}

@keyframes isu-blink {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}
</style>
