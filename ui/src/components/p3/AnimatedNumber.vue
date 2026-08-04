<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'

const props = withDefaults(defineProps<{
  value: number
  decimals?: number
  duration?: number
  prefix?: string
  suffix?: string
}>(), {
  decimals: 0,
  duration: 700,
  prefix: '',
  suffix: '',
})

const display = ref(0)
let from = 0
let to = 0
let startTime = 0
let raf = 0

function step(now: number) {
  const t = Math.min(1, (now - startTime) / props.duration)
  const eased = 1 - Math.pow(1 - t, 3)
  display.value = from + (to - from) * eased
  if (t < 1) {
    raf = requestAnimationFrame(step)
  }
}

function animate(target: number) {
  cancelAnimationFrame(raf)
  from = display.value
  to = target
  startTime = performance.now()
  raf = requestAnimationFrame(step)
}

watch(() => props.value, v => animate(v), { immediate: true })

onMounted(() => {
  display.value = props.value
})
onBeforeUnmount(() => cancelAnimationFrame(raf))
</script>

<template>
  <span class="font-mono-num tabular-nums">{{ prefix }}{{ display.toFixed(decimals) }}{{ suffix }}</span>
</template>
