<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { MonitorPlay, Activity, Zap, ShieldCheck } from 'lucide-vue-next'
import { useVisualStore } from '@/stores/visual'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import RingGauge from '@/components/p3/RingGauge.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import VisualScreen from '@/components/visual/VisualScreen.vue'

const store = useVisualStore()
const { time } = useLiveClock()

const BAR_COLORS = ['#0EA5E9', '#22D3EE', '#34D399', '#FBBF24', '#F472B6', '#A78BFA']
const MAX_ALARM = Math.max(...store.alarmSeries)

const alarmBars = computed(() => store.alarmSeries.map((v, i) => ({
  h: Math.round((v / MAX_ALARM) * 100),
  color: BAR_COLORS[i % BAR_COLORS.length],
  label: String(i + 1),
})))

const screenEvents = computed(() =>
  store.events.map(e => ({
    title: e.title,
    subtitle: e.subtitle,
    meta: e.meta,
    status: e.status === 'error' ? 'error' : e.status === 'warning' ? 'warning' : 'resolved',
  }))
)

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <VisualScreen bleed>
    <div class="screen-page">
    <div class="screen-frame">
      <!-- 顶栏 -->
      <header class="screen-header">
        <div class="screen-corner screen-corner-l" />
        <div class="screen-corner screen-corner-r" />
        <div class="flex items-center gap-3">
          <MonitorPlay class="h-5 w-5" style="color: #22D3EE;" />
          <div>
            <h1 class="screen-title">工业可视化大屏</h1>
            <p class="text-[10px] tracking-[0.3em]" style="color: var(--text-secondary);">INDUSTRIAL VISUALIZATION</p>
          </div>
        </div>
        <div class="flex items-center gap-4 text-xs" style="color: var(--text-secondary);">
          <span class="inline-flex items-center gap-1.5" style="color: #34D399;">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
            SYS RUNNING
          </span>
          <span class="screen-clock font-mono-num">{{ time }}</span>
        </div>
      </header>

      <!-- KPI 行 -->
      <section class="screen-kpis">
        <div class="screen-kpi">
          <div class="flex items-center justify-between">
            <span class="screen-kpi-label">总负荷</span>
            <Zap class="h-3.5 w-3.5" style="color: #FBBF24;" />
          </div>
          <div class="screen-kpi-value" style="color: #38BDF8;"><AnimatedNumber :value="store.powerLoad" decimals="1" suffix="kW" /></div>
          <BarMeter :value="(store.powerLoad / 600) * 100" :max="100" color="#38BDF8" :height="5" />
        </div>
        <div class="screen-kpi">
          <div class="flex items-center justify-between">
            <span class="screen-kpi-label">今日能耗</span>
            <Activity class="h-3.5 w-3.5" style="color: #34D399;" />
          </div>
          <div class="screen-kpi-value" style="color: #34D399;"><AnimatedNumber :value="store.totalEnergy" decimals="1" suffix="MWh" /></div>
          <BarMeter :value="((store.totalEnergy % 100) / 100) * 100" :max="100" color="#34D399" :height="5" />
        </div>
        <div class="screen-kpi">
          <div class="flex items-center justify-between">
            <span class="screen-kpi-label">在线设备</span>
            <ShieldCheck class="h-3.5 w-3.5" style="color: #A78BFA;" />
          </div>
          <div class="screen-kpi-value" style="color: #A78BFA;"><AnimatedNumber :value="store.activeDevices" /></div>
          <BarMeter :value="store.deviceOnline" :max="100" color="#A78BFA" :height="5" />
        </div>
        <div class="screen-kpi">
          <div class="flex items-center justify-between">
            <span class="screen-kpi-label">设备在线率</span>
            <Activity class="h-3.5 w-3.5" style="color: #22D3EE;" />
          </div>
          <div class="screen-kpi-value" style="color: #22D3EE;"><AnimatedNumber :value="store.deviceOnline" decimals="1" suffix="%" /></div>
          <BarMeter :value="store.deviceOnline" :max="100" color="#22D3EE" :height="5" />
        </div>
        <div class="screen-kpi">
          <div class="flex items-center justify-between">
            <span class="screen-kpi-label">活跃告警</span>
            <span class="h-1.5 w-1.5 animate-pulse rounded-full" style="background: #F87171;" />
          </div>
          <div class="screen-kpi-value" style="color: #F87171;"><AnimatedNumber :value="store.alarmCount" /></div>
          <BarMeter :value="(store.alarmCount / 12) * 100" :max="100" color="#F87171" :height="5" />
        </div>
      </section>

      <!-- 中部图表 -->
      <section class="screen-grid">
        <article class="screen-panel screen-panel-wide">
          <div class="screen-panel-head">
            <span>负荷趋势 (近 1 小时)</span>
            <span class="screen-tag">LIVE</span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.loadSeries" color="#38BDF8" :height="120" />
          </div>
        </article>

        <article class="screen-panel">
          <div class="screen-panel-head">
            <span>能量水平</span>
            <span class="screen-tag">SOC</span>
          </div>
          <div class="mt-2 flex justify-center">
            <RingGauge :value="store.energyTrend[store.energyTrend.length - 1]" :size="150" color="#22D3EE" label="综合能效">
              <span class="font-mono-num text-2xl font-bold" style="color: #22D3EE;">
                <AnimatedNumber :value="store.energyTrend[store.energyTrend.length - 1]" decimals="1" suffix="%" />
              </span>
            </RingGauge>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.energyTrend" color="#22D3EE" :height="40" />
          </div>
        </article>

        <article class="screen-panel">
          <div class="screen-panel-head">
            <span>通信质量</span>
            <span class="screen-tag">QoS</span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.qualitySeries" color="#34D399" :height="96" />
          </div>
          <div class="mt-2 text-right text-xs font-mono-num" style="color: #34D399;">
            <AnimatedNumber :value="store.qualitySeries[store.qualitySeries.length - 1]" decimals="1" suffix=" / 100" />
          </div>
        </article>

        <article class="screen-panel screen-panel-wide">
          <div class="screen-panel-head">
            <span>告警分布</span>
            <span class="screen-tag">24h</span>
          </div>
          <div class="mt-3 flex h-[110px] items-end gap-1.5 px-1">
            <div
              v-for="bar in alarmBars"
              :key="bar.label"
              class="flex-1 rounded-sm transition-all duration-500"
              :style="{
                height: `${bar.h}%`,
                background: bar.color,
                opacity: 0.85,
                boxShadow: `0 0 10px ${bar.color}66`,
              }"
            />
          </div>
        </article>

        <article class="screen-panel screen-panel-wide">
          <div class="screen-panel-head">
            <span>重点指标</span>
            <span class="screen-tag">TABLE</span>
          </div>
          <div class="screen-table">
            <div class="screen-table-row screen-table-head">
              <span>指标</span>
              <span>当前值</span>
              <span>基准</span>
              <span>状态</span>
            </div>
            <div class="screen-table-row">
              <span>产线节拍</span>
              <span class="font-mono-num" style="color: #38BDF8;">{{ store.lineSpeed }}%</span>
              <span>≥ 80%</span>
              <span style="color: #34D399;">正常</span>
            </div>
            <div class="screen-table-row">
              <span>一次良率</span>
              <span class="font-mono-num" style="color: #34D399;">{{ store.okRate.toFixed(1) }}%</span>
              <span>≥ 97%</span>
              <span style="color: #34D399;">正常</span>
            </div>
            <div class="screen-table-row">
              <span>平均 OEE</span>
              <span class="font-mono-num" style="color: #A78BFA;">{{ store.avgOee }}%</span>
              <span>≥ 85%</span>
              <span style="color: #34D399;">正常</span>
            </div>
            <div class="screen-table-row">
              <span>母线电压</span>
              <span class="font-mono-num" style="color: #38BDF8;">{{ store.gauges.find(g => g.key === 'voltage')?.value.toFixed(1) ?? '—' }}V</span>
              <span>380-420V</span>
              <span style="color: #34D399;">正常</span>
            </div>
            <div class="screen-table-row">
              <span>管道压力</span>
              <span class="font-mono-num" style="color: #FBBF24;">{{ store.gauges.find(g => g.key === 'pressure')?.value.toFixed(2) ?? '—' }}MPa</span>
              <span>≤ 0.8MPa</span>
              <span style="color: #FBBF24;">关注</span>
            </div>
          </div>
        </article>
      </section>

      <!-- 底部事件 -->
      <section class="mt-4">
        <P3EventFeed title="实时事件流" :events="screenEvents" />
      </section>
      </div>
    </div>
  </VisualScreen>
</template>

<style scoped>
.screen-page {
  --sp-page-bg:    #F8FAFC;
  --sp-page-glow:  rgba(14, 165, 233, 0.06);
  --sp-frame-bg:   rgba(255, 255, 255, 0.86);
  --sp-panel-bg:   rgba(255, 255, 255, 0.68);
  --sp-kpi-bg:     linear-gradient(180deg, rgba(14, 165, 233, 0.05), rgba(255, 255, 255, 0.62));
  --sp-title:      var(--text-primary);
  --sp-title-glow: none;
  margin: -24px;
  padding: 24px;
  background:
    radial-gradient(circle at 50% 0%, var(--sp-page-glow), transparent 55%),
    var(--sp-page-bg);
  min-height: calc(100vh - 56px);
}

:global(html.dark) .screen-page {
  --sp-page-bg:    #050B16;
  --sp-page-glow:  rgba(14, 165, 233, 0.09);
  --sp-frame-bg:   rgba(8, 16, 30, 0.92);
  --sp-panel-bg:   rgba(10, 20, 38, 0.6);
  --sp-kpi-bg:     linear-gradient(180deg, rgba(14, 165, 233, 0.08), rgba(8, 16, 30, 0.4));
  --sp-title:      #E2E8F0;
  --sp-title-glow: 0 0 18px rgba(14, 165, 233, 0.45);
}

.screen-frame {
  max-width: 1440px;
  margin: 0 auto;
  border: 1px solid rgba(14, 165, 233, 0.22);
  background: var(--sp-frame-bg);
  padding: 20px;
}

.screen-header {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(14, 165, 233, 0.18);
  padding-bottom: 14px;
}

.screen-corner {
  position: absolute;
  width: 10px;
  height: 10px;
  border: 1px solid #0EA5E9;
}

.screen-corner-l {
  left: -1px;
  top: -1px;
  border-right: none;
  border-bottom: none;
}

.screen-corner-r {
  right: -1px;
  top: -1px;
  border-left: none;
  border-bottom: none;
}

.screen-title {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0.16em;
  color: var(--sp-title);
  text-shadow: var(--sp-title-glow);
}

.screen-clock {
  font-size: 16px;
  color: #38BDF8;
  text-shadow: 0 0 12px rgba(56, 189, 248, 0.5);
}

.screen-kpis {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px;
  margin-top: 16px;
}

.screen-kpi {
  border: 1px solid rgba(14, 165, 233, 0.2);
  background: var(--sp-kpi-bg);
  padding: 12px 14px;
}

.screen-kpi-label {
  font-size: 11px;
  letter-spacing: 0.18em;
  color: var(--text-secondary);
}

.screen-kpi-value {
  margin-top: 6px;
  font-size: 22px;
  font-weight: 700;
  font-family: 'JetBrains Mono', monospace;
}

.screen-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-top: 16px;
}

.screen-panel {
  border: 1px solid rgba(14, 165, 233, 0.18);
  background: var(--sp-panel-bg);
  padding: 14px;
}

.screen-panel-wide {
  grid-column: span 2;
}

.screen-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.12em;
  color: var(--text-secondary);
  border-bottom: 1px solid rgba(14, 165, 233, 0.14);
  padding-bottom: 8px;
}

.screen-tag {
  font-size: 9px;
  color: #0EA5E9;
  border: 1px solid rgba(14, 165, 233, 0.4);
  padding: 1px 6px;
  letter-spacing: 0.2em;
}

.screen-table {
  margin-top: 8px;
}

.screen-table-row {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr 0.8fr 0.6fr;
  gap: 8px;
  font-size: 12px;
  padding: 7px 4px;
  color: var(--text-secondary);
  border-bottom: 1px solid rgba(148, 163, 184, 0.08);
}

.screen-table-head {
  color: var(--text-muted);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

@media (max-width: 1100px) {
  .screen-kpis {
    grid-template-columns: repeat(3, 1fr);
  }
  .screen-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .screen-panel-wide {
    grid-column: span 1;
  }
}
</style>
