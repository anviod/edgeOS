<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  label: string
  value: number
  min?: number
  max?: number
  unit?: string
  color?: string
  size?: number
  decimals?: number
  zone?: boolean
}>(), {
  min: 0,
  max: 100,
  unit: '',
  color: '#0EA5E9',
  size: 168,
  decimals: 1,
  zone: false,
})

const stroke = 12
const radius = computed(() => (props.size - stroke) / 2)
const cx = computed(() => props.size / 2)
const cy = computed(() => props.size / 2)

const startAngle = 135
const sweep = 270

function polar(angleDeg: number, r: number) {
  const rad = ((angleDeg - 90) * Math.PI) / 180
  return {
    x: cx.value + r * Math.cos(rad),
    y: cy.value + r * Math.sin(rad),
  }
}

const trackPath = computed(() => {
  const a0 = polar(startAngle, radius.value)
  const a1 = polar(startAngle + sweep, radius.value)
  const large = sweep > 180 ? 1 : 0
  return `M ${a0.x} ${a0.y} A ${radius.value} ${radius.value} 0 ${large} 1 ${a1.x} ${a1.y}`
})

const pct = computed(() => Math.max(0, Math.min(100, ((props.value - props.min) / (props.max - props.min)) * 100)))

const valuePath = computed(() => {
  const a0 = polar(startAngle, radius.value)
  const a1 = polar(startAngle + (sweep * pct.value) / 100, radius.value)
  const large = (sweep * pct.value) / 100 > 180 ? 1 : 0
  return `M ${a0.x} ${a0.y} A ${radius.value} ${radius.value} 0 ${large} 1 ${a1.x} ${a1.y}`
})

const needleAngle = computed(() => startAngle + (sweep * pct.value) / 100)

const needleEnd = computed(() => polar(needleAngle.value, radius.value - 16))
const needleBase = computed(() => polar(needleAngle.value, 20))

const ticks = computed(() => {
  const out: { angle: number; major: boolean }[] = []
  for (let i = 0; i <= 20; i++) {
    out.push({ angle: startAngle + (sweep * i) / 20, major: i % 5 === 0 })
  }
  return out
})

const zoneColor = computed(() => {
  if (!props.zone) return props.color
  if (pct.value < 70) return '#10B981'
  if (pct.value < 90) return '#F59E0B'
  return '#EF4444'
})
</script>

<template>
  <div class="inline-flex flex-col items-center">
    <div class="relative" :style="{ width: `${size}px`, height: `${size}px` }">
      <svg :width="size" :height="size" class="block">
        <defs>
          <filter id="gauge-glow" x="-40%" y="-40%" width="180%" height="180%">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <path :d="trackPath" fill="none" stroke="rgba(148,163,184,0.18)" :stroke-width="stroke" stroke-linecap="round" />

        <path
          :d="valuePath"
          fill="none"
          :stroke="zoneColor"
          :stroke-width="stroke"
          stroke-linecap="round"
          style="filter: url(#gauge-glow); transition: stroke-dasharray 0.8s ease, stroke 0.6s ease"
        />

        <g v-for="(t, i) in ticks" :key="i">
          <line
            :x1="polar(t.angle, radius.value - (t.major ? 16 : 10)).x"
            :y1="polar(t.angle, radius.value - (t.major ? 16 : 10)).y"
            :x2="polar(t.angle, radius.value - 4).x"
            :y2="polar(t.angle, radius.value - 4).y"
            :stroke="t.major ? 'rgba(148,163,184,0.55)' : 'rgba(148,163,184,0.28)'"
            :stroke-width="t.major ? 2 : 1"
          />
        </g>

        <line
          :x1="needleBase.x"
          :y1="needleBase.y"
          :x2="needleEnd.x"
          :y2="needleEnd.y"
          :stroke="zoneColor"
          stroke-width="3"
          stroke-linecap="round"
          style="filter: url(#gauge-glow); transition: all 0.8s cubic-bezier(0.4, 0, 0.2, 1)"
        />
        <circle :cx="cx" :cy="cy" :r="7" fill="#0F172A" :stroke="zoneColor" stroke-width="2" />
      </svg>

      <div class="absolute inset-0 flex flex-col items-center justify-center pt-8">
        <span class="gauge-value font-mono-num" :style="{ color: zoneColor }">
          {{ value.toFixed(decimals) }}<span class="text-xs">{{ unit }}</span>
        </span>
        <span class="mt-1 text-[10px] uppercase tracking-[0.18em]" style="color: var(--text-muted);">{{ label }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gauge-value {
  font-size: 22px;
  font-weight: 700;
  text-shadow: 0 0 12px rgba(14, 165, 233, 0.35);
}
</style>
