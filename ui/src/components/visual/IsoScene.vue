<script setup lang="ts">
import { computed } from 'vue'
import IsoHud from './IsoHud.vue'

const props = withDefaults(defineProps<{
  cols?: number
  rows?: number
  cell?: number
  height?: number
  showGrid?: boolean
  gridColor?: string
  scan?: boolean
  scale?: number
}>(), {
  cols: 6,
  rows: 6,
  cell: 56,
  height: 380,
  showGrid: true,
  gridColor: 'rgba(14, 165, 233, 0.16)',
  scan: true,
  scale: 1,
})

const worldW = computed(() => props.cols * props.cell)
const worldH = computed(() => props.rows * props.cell)

const gridStyle = computed(() => ({
  width: `${worldW.value}px`,
  height: `${worldH.value}px`,
  backgroundImage: props.showGrid
    ? `linear-gradient(${props.gridColor} 1px, transparent 1px), linear-gradient(90deg, ${props.gridColor} 1px, transparent 1px)`
    : 'none',
  backgroundSize: `${props.cell}px ${props.cell}px`,
  backgroundPosition: '0 0',
}))

const corners = ['tl', 'tr', 'bl', 'br'] as const
</script>

<template>
  <div class="iso-scene" :style="{ height: `${height}px` }">
    <div
      class="iso-world"
      :style="{
        width: `${worldW}px`,
        height: `${worldH}px`,
        marginLeft: `${-worldW / 2}px`,
        marginTop: `${-worldH / 2}px`,
        '--scale': scale,
      }"
    >
      <div class="iso-grid" :style="gridStyle" />
      <div class="iso-floor" :style="{ width: `${worldW}px`, height: `${worldH}px` }" />
      <slot />
    </div>

    <!-- 悬停 / 点击 HUD 提示 -->
    <IsoHud />

    <!-- 四角装饰 -->
    <span v-for="c in corners" :key="c" class="iso-corner" :class="`iso-corner--${c}`" />

    <!-- 扫描线 -->
    <div v-if="scan" class="iso-scan" />
  </div>
</template>

<style scoped>
.iso-scene {
  position: relative;
  width: 100%;
  overflow: hidden;
  overflow: clip;
  display: flex;
  align-items: center;
  justify-content: center;
}

.iso-world {
  position: absolute;
  left: 50%;
  top: 50%;
  transform-style: preserve-3d;
  transform: rotateX(60deg) rotateZ(-45deg) scale(var(--scale, 1));
  transform-origin: 50% 50%;
}

.iso-grid {
  position: absolute;
  left: 0;
  top: 0;
  transform-style: preserve-3d;
  opacity: 0.9;
}

.iso-floor {
  position: absolute;
  left: 0;
  top: 0;
  background: radial-gradient(circle at 50% 50%, rgba(14, 165, 233, 0.06), rgba(14, 165, 233, 0.02) 60%, transparent 75%);
  transform-style: preserve-3d;
}

.iso-corner {
  position: absolute;
  width: 14px;
  height: 14px;
  pointer-events: none;
  opacity: 0.7;
}

.iso-corner::before,
.iso-corner::after {
  content: '';
  position: absolute;
  background: #0EA5E9;
}

.iso-corner::before {
  width: 14px;
  height: 2px;
}

.iso-corner::after {
  width: 2px;
  height: 14px;
}

.iso-corner--tl {
  left: 8px;
  top: 8px;
}

.iso-corner--tl::before {
  top: 0;
  left: 0;
}

.iso-corner--tl::after {
  top: 0;
  left: 0;
}

.iso-corner--tr {
  right: 8px;
  top: 8px;
}

.iso-corner--tr::before {
  top: 0;
  right: 0;
}

.iso-corner--tr::after {
  top: 0;
  right: 0;
}

.iso-corner--bl {
  left: 8px;
  bottom: 8px;
}

.iso-corner--bl::before {
  bottom: 0;
  left: 0;
}

.iso-corner--bl::after {
  bottom: 0;
  left: 0;
}

.iso-corner--br {
  right: 8px;
  bottom: 8px;
}

.iso-corner--br::before {
  bottom: 0;
  right: 0;
}

.iso-corner--br::after {
  bottom: 0;
  right: 0;
}

.iso-scan {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, rgba(14, 165, 233, 0.5), transparent);
  box-shadow: 0 0 14px rgba(14, 165, 233, 0.35);
  animation: iso-scan-move 7s ease-in-out infinite;
  pointer-events: none;
}

@keyframes iso-scan-move {
  0%, 10% {
    top: 0;
    opacity: 0;
  }
  20%, 88% {
    opacity: 1;
  }
  100% {
    top: 100%;
    opacity: 0;
  }
}
</style>
