<script setup lang="ts">
import { BarChart3, BatteryCharging, Cable, Cpu, Receipt, Zap } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import BusinessKpiStrip from '@/components/p3/BusinessKpiStrip.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'
import QualityTrendCard from '@/components/p3/QualityTrendCard.vue'

const { page } = useP3Page('business-center')

const modules = [
  { label: '储能管理', path: '/business-center/energy-storage', icon: BatteryCharging, note: '调峰 / SOC / SOH' },
  { label: '电源BMS', path: '/business-center/power-bms', icon: Cpu, note: '温差 / 压差 / 均衡' },
  { label: '充电管理', path: '/business-center/charging', icon: Cable, note: '枪位 / 排队 / 订单' },
  { label: '能耗监测', path: '/business-center/energy-monitoring', icon: BarChart3, note: '能流 / 峰谷 / 异常' },
  { label: '账务台账', path: '/business-center/ledger', icon: Receipt, note: '账单 / 发票 / 对账' },
]
</script>

<template>
  <div class="space-y-5">
    <section class="business-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1.25fr_0.75fr]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs business-pill">
            <Zap class="h-3.5 w-3.5" />
            业务经营驾驶舱
          </div>
          <h1 class="mt-3 text-3xl font-semibold business-title">{{ page.title }}</h1>
          <p class="mt-2 max-w-3xl text-sm business-subtitle">{{ page.subtitle }}</p>
          <div class="mt-5">
            <P3StatusStrip :data="page.statusStrip" />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-1">
          <router-link
            v-for="module in modules"
            :key="module.path"
            :to="module.path"
            class="business-module rounded-xl border p-4 transition-colors"
          >
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-sm font-medium business-title">{{ module.label }}</div>
                <div class="mt-1 text-xs business-subtitle">{{ module.note }}</div>
              </div>
              <component :is="module.icon" class="h-5 w-5 business-icon" />
            </div>
          </router-link>
        </div>
      </div>
    </section>

    <BusinessKpiStrip :items="page.kpis" />

    <section class="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-2">
        <QualityTrendCard
          v-for="trend in page.trends"
          :key="trend.title"
          :item="trend"
          class="business-card"
        />
      </div>
      <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
    </section>
  </div>
</template>

<style scoped>
.business-hero {
  background:
    linear-gradient(90deg, rgba(14, 165, 233, 0.08), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.business-pill {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.business-title {
  color: var(--text-primary);
}

.business-subtitle {
  color: var(--text-secondary);
}

.business-module {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.business-module:hover {
  border-color: rgba(14, 165, 233, 0.3);
  background: var(--bg-tertiary);
}

.business-icon {
  color: #0EA5E9;
}
</style>
