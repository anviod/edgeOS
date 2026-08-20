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
  speed?: number
  itemCount?: number
  hud?: IsoHudData
}>(), {
  col: 0,
  row: 0,
  x: 0,
  y: 0,
  cell: 56,
  width: 224,
  depth: 20,
  height: 10,
  color: '#0EA5E9',
  speed: 5,
  itemCount: 5,
  hud: undefined,
})

const left = computed(() => props.x || props.col * props.cell)
const top = computed(() => props.y || props.row * props.cell)

const PALETTE = ['#38BDF8', '#34D399', '#FBBF24', '#A78BFA', '#F472B6', '#22D3EE']

const items = computed(() =>
  Array.from({ length: props.itemCount }, (_, i) => ({
    x: (props.width / props.itemCount) * i + 6,
    top: 4,
    color: PALETTE[i % PALETTE.length],
    delay: (props.speed / props.itemCount) * i,
  }))
)

const beltStyle = computed(() => ({
  left: `${left.value}px`,
  top: `${top.value}px`,
  width: `${props.width}px`,
  height: `${props.depth}px`,
  '--bh': `${props.height}px`,
  '--belt': `${props.width - 24}px`,
  '--dur': `${props.speed}s`,
  '--belt-color': props.color,
}))

const hudData = computed<IsoHudData>(() => ({
  title: '输送带',
  accent: props.color,
  rows: [
    { label: '长度', value: `${props.width}` },
    { label: '节拍', value: `${props.speed}s` },
    { label: '在线工件', value: `${props.itemCount} 件` },
  ],
  ...(props.hud ?? {}),
}))
</script>

<template>
  <div v-hud="hudData" class="iso-conveyor" :style="beltStyle">
    <div class="cface c-roof" :style="{ background: `${color}26`, borderColor: `${color}55` }" />
    <div class="cface c-front" :style="{ background: `${color}44` }" />
    <div class="cface c-side" :style="{ background: `${color}33` }" />

    <div
      v-for="(it, i) in items"
      :key="i"
      class="c-item"
      :style="{
        left: `${it.x}px`,
        top: `${it.top}px`,
        '--idly': `${it.delay}s`,
      }"
    >
      <div class="c-item-roof" :style="{ background: it.color }" />
      <div class="c-item-front" :style="{ background: `${it.color}cc` }" />
      <div class="c-item-side" :style="{ background: `${it.color}99` }" />
    </div>
  </div>
</template>

<style scoped>
.iso-conveyor {
  position: absolute;
  transform-style: preserve-3d;
  cursor: help;
}

.cface {
  position: absolute;
}

.c-roof {
  inset: 0;
  transform: translateZ(var(--bh));
  border: 1px solid;
  box-shadow: inset 0 0 14px rgba(255, 255, 255, 0.05);
}

.c-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--bh);
  transform-origin: center top;
  transform: rotateX(90deg);
}

.c-side {
  left: 0;
  top: 0;
  width: var(--bh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

.c-item {
  position: absolute;
  width: 12px;
  height: 12px;
  transform-style: preserve-3d;
  animation: belt-left var(--dur) linear infinite;
  animation-delay: var(--idly);
}

.c-item-roof {
  position: absolute;
  inset: 0;
  transform: translateZ(10px);
  border: 1px solid rgba(255, 255, 255, 0.5);
}

.c-item-front {
  position: absolute;
  left: 0;
  top: 100%;
  width: 100%;
  height: 10px;
  transform-origin: center top;
  transform: rotateX(90deg);
}

.c-item-side {
  position: absolute;
  left: 0;
  top: 0;
  width: 10px;
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
}

@keyframes belt-left {
  from {
    left: 6px;
  }
  to {
    left: var(--belt);
  }
}
</style>
