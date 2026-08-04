<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'

const props = withDefaults(defineProps<{
  points: number[]
  height?: number
  color?: string
  fill?: boolean
}>(), {
  height: 64,
  color: '#0EA5E9',
  fill: true,
})

const canvasRef = ref<HTMLCanvasElement | null>(null)
let raf = 0
let displayPoints: number[] = []
let targetPoints: number[] = []

function hexToRgba(hex: string, alpha: number) {
  const h = hex.replace('#', '')
  const r = parseInt(h.substring(0, 2), 16)
  const g = parseInt(h.substring(2, 4), 16)
  const b = parseInt(h.substring(4, 6), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

function draw() {
  const canvas = canvasRef.value
  if (!canvas) return
  const dpr = window.devicePixelRatio || 1
  const w = canvas.clientWidth || canvas.width / dpr
  const h = props.height
  canvas.width = w * dpr
  canvas.height = h * dpr
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.scale(dpr, dpr)

  const pts = displayPoints
  if (pts.length < 2) return
  const min = Math.min(...pts)
  const max = Math.max(...pts)
  const range = max - min || 1
  const pad = 6

  const stepX = (w - pad * 2) / (pts.length - 1)
  const coords = pts.map((v, i) => ({
    x: pad + i * stepX,
    y: h - pad - ((v - min) / range) * (h - pad * 2),
  }))

  ctx.clearRect(0, 0, w, h)

  // 面积渐变填充
  if (props.fill) {
    const gradient = ctx.createLinearGradient(0, 0, 0, h)
    gradient.addColorStop(0, hexToRgba(props.color, 0.28))
    gradient.addColorStop(1, hexToRgba(props.color, 0.02))
    ctx.beginPath()
    ctx.moveTo(coords[0].x, h - pad)
    coords.forEach(c => ctx.lineTo(c.x, c.y))
    ctx.lineTo(coords[coords.length - 1].x, h - pad)
    ctx.closePath()
    ctx.fillStyle = gradient
    ctx.fill()
  }

  // 曲线
  ctx.beginPath()
  coords.forEach((c, i) => {
    if (i === 0) ctx.moveTo(c.x, c.y)
    else {
      const prev = coords[i - 1]
      const cx = (prev.x + c.x) / 2
      ctx.bezierCurveTo(cx, prev.y, cx, c.y, c.x, c.y)
    }
  })
  ctx.strokeStyle = props.color
  ctx.lineWidth = 2
  ctx.lineJoin = 'round'
  ctx.lineCap = 'round'
  ctx.stroke()

  // 末端发光点
  const last = coords[coords.length - 1]
  ctx.beginPath()
  ctx.arc(last.x, last.y, 4.5, 0, Math.PI * 2)
  ctx.fillStyle = hexToRgba(props.color, 0.25)
  ctx.fill()
  ctx.beginPath()
  ctx.arc(last.x, last.y, 2.5, 0, Math.PI * 2)
  ctx.fillStyle = props.color
  ctx.fill()
}

function easeToward(_now: number) {
  let changed = false
  for (let i = 0; i < displayPoints.length; i++) {
    const target = targetPoints[i] ?? displayPoints[i]
    const next = displayPoints[i] + (target - displayPoints[i]) * 0.18
    if (Math.abs(target - next) > 0.01) changed = true
    displayPoints[i] = next
  }
  draw()
  if (changed) {
    raf = requestAnimationFrame(easeToward)
  }
}

watch(
  () => props.points,
  (newPoints) => {
    if (!newPoints || newPoints.length === 0) return
    if (displayPoints.length === 0) {
      displayPoints = [...newPoints]
      targetPoints = [...newPoints]
      draw()
      return
    }
    // 对齐长度：新序列向右追加，丢弃最左
    targetPoints = [...newPoints]
    while (targetPoints.length < displayPoints.length) targetPoints.unshift(displayPoints[0])
    while (targetPoints.length > displayPoints.length) targetPoints.shift()
    cancelAnimationFrame(raf)
    raf = requestAnimationFrame(easeToward)
  }
)

onMounted(() => {
  displayPoints = [...(props.points ?? [])]
  targetPoints = [...displayPoints]
  draw()
})

onBeforeUnmount(() => cancelAnimationFrame(raf))
</script>

<template>
  <canvas
    ref="canvasRef"
    class="w-full"
    :style="{ height: `${height}px` }"
  />
</template>
