<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  y?: number
  left?: number
  size?: number
  height?: number
  range?: number
  dur?: number
  delay?: number
  color?: string
}>(), {
  y: 0,
  left: 0,
  size: 10,
  height: 8,
  range: 120,
  dur: 4,
  delay: 0,
  color: '#38BDF8',
})

const top = computed(() => `${props.y}px`)
</script>

<template>
  <div
    class="iso-dot"
    :style="{
      top,
      left: `${left}px`,
      width: `${size}px`,
      height: `${size}px`,
      '--dh': `${height}px`,
      '--ds': `${size}px`,
      '--dc': color,
      '--dr': `${range}px`,
      '--dd': `${dur}s`,
      '--dly': `${delay}s`,
    }"
  >
    <div class="dot-face dot-roof" :style="{ background: `${color}55`, borderColor: color }" />
    <div class="dot-face dot-front" :style="{ background: `${color}99` }" />
    <div class="dot-face dot-side" :style="{ background: `${color}bb` }" />
  </div>
</template>

<style scoped>
.iso-dot {
  position: absolute;
  pointer-events: none;
  transform-style: preserve-3d;
  animation: dot-move var(--dd) linear infinite;
  animation-delay: var(--dly);
}

.dot-face {
  position: absolute;
}

.dot-roof {
  inset: 0;
  transform: translateZ(var(--dh));
  border: 1px solid;
  box-shadow: 0 0 12px var(--dc);
}

.dot-front {
  left: 0;
  top: 100%;
  width: 100%;
  height: var(--dh);
  transform-origin: center top;
  transform: rotateX(90deg);
  box-shadow: 0 0 10px var(--dc);
}

.dot-side {
  left: 0;
  top: 0;
  width: var(--dh);
  height: 100%;
  transform-origin: left center;
  transform: rotateY(-90deg);
  box-shadow: 0 0 8px var(--dc);
}

@keyframes dot-move {
  from {
    left: 0;
  }
  to {
    left: var(--dr);
  }
}
</style>
