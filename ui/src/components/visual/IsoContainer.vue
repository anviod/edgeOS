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
  color?: string
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  z: 0,
  width: 44,
  depth: 22,
  height: 22,
  color: '#2563EB',
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const roof = computed(() => `${props.color}dd`)
const front = computed(() => `${props.color}bb`)
const side = computed(() => `${props.color}99`)

const hudData = computed<IsoHudData>(() => ({
  title: '集装箱',
  accent: props.color,
  badge: '标准箱',
  rows: [
    { label: '规格', value: `${props.width}×${props.depth}×${props.height}` },
    { label: '箱色', value: props.color, color: props.color },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div
    v-hud="hudData"
    class="iso-ctn"
    :style="{
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${depth}px`,
      '--ch': `${height}px`,
      transform: `translateZ(${z}px)`,
    }"
  >
    <div class="cf c-roof" :style="{ background: roof, borderColor: `${color}ff` }" />
    <div class="cf c-front" :style="{ background: front }" />
    <div class="cf c-side" :style="{ background: side }" />
    <!-- 顶部锁扣 -->
    <div class="c-lock c-lock--tl" :style="{ background: '#0F172A' }" />
    <div class="c-lock c-lock--tr" :style="{ background: '#0F172A' }" />
  </div>
</template>

<style scoped>
.iso-ctn {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.cf {
  position: absolute;
}

.c-roof {
  inset: 0;
  transform: translateZ(var(--ch));
  border: 1px solid;
  box-shadow: inset 0 0 10px rgba(255, 255, 255, 0.25);
}

.c-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--ch);
  transform-origin: center top;
  transform: rotateX(90deg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.22);
  background-image: repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.14) 0 3px, transparent 3px 11px);
}

.c-side {
  left: 0;
  top: 0;
  width: var(--ch);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
  border-right: 1px solid rgba(255, 255, 255, 0.16);
}

.c-lock {
  position: absolute;
  width: 5px;
  height: 5px;
  transform: translateZ(calc(var(--ch) + 1px));
}

.c-lock--tl {
  left: 6px;
  top: 6px;
}

.c-lock--tr {
  right: 6px;
  top: 6px;
}
</style>
