<script setup lang="ts">
import { computed } from 'vue'
import { BatteryCharging, Gauge, ShieldCheck } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import DispatchQueueTable from '@/components/p3/DispatchQueueTable.vue'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'
import QualityTrendCard from '@/components/p3/QualityTrendCard.vue'

const { page } = useP3Page('energy-storage')
const primarySite = computed(() => page.value.mainTable.rows[0])
</script>

<template>
  <div class="space-y-5">
    <section class="storage-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[320px_1fr_320px]">
        <div class="rounded-2xl border p-5 storage-gauge">
          <div class="flex items-center justify-between">
            <span class="text-xs uppercase tracking-[0.18em] storage-muted">储能总仓</span>
            <BatteryCharging class="h-5 w-5 text-emerald-500" />
          </div>
          <div class="mt-6 flex items-center justify-center">
            <div class="storage-ring flex h-40 w-40 items-center justify-center rounded-full">
              <div class="text-center">
                <div class="text-xs storage-muted">Average SOC</div>
                <div class="mt-2 text-4xl font-semibold font-mono-num storage-title">68%</div>
              </div>
            </div>
          </div>
          <div class="mt-5 grid grid-cols-2 gap-3 text-xs">
            <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="storage-muted">SOH</div>
              <div class="mt-1 font-mono-num storage-title">93</div>
            </div>
            <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="storage-muted">Power</div>
              <div class="mt-1 font-mono-num storage-title">41.8MW</div>
            </div>
          </div>
        </div>

        <div class="space-y-4">
          <div>
            <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs storage-chip">
              <Gauge class="h-3.5 w-3.5" />
              站级调峰运行面板
            </div>
            <h1 class="mt-3 text-3xl font-semibold storage-title">{{ page.title }}</h1>
            <p class="mt-2 text-sm storage-subtitle">{{ page.subtitle }}</p>
          </div>
          <P3StatusStrip :data="page.statusStrip" />
          <div class="grid gap-4 xl:grid-cols-2">
            <QualityTrendCard :item="page.trends[0]" />
            <QualityTrendCard :item="page.trends[2]" />
          </div>
        </div>

        <div class="rounded-2xl border p-5 storage-right">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-xs uppercase tracking-[0.18em] storage-muted">主站点</div>
              <div class="mt-2 text-lg font-semibold storage-title">{{ primarySite.site }}</div>
            </div>
            <ShieldCheck class="h-5 w-5 text-sky-500" />
          </div>
          <div class="mt-5 space-y-3 text-sm">
            <div class="flex justify-between"><span class="storage-muted">SOC</span><span class="font-mono-num storage-title">{{ primarySite.soc }}</span></div>
            <div class="flex justify-between"><span class="storage-muted">SOH</span><span class="font-mono-num storage-title">{{ primarySite.soh }}</span></div>
            <div class="flex justify-between"><span class="storage-muted">功率</span><span class="font-mono-num storage-title">{{ primarySite.power }}</span></div>
            <div class="flex justify-between"><span class="storage-muted">策略</span><span class="storage-title">{{ primarySite.strategy }}</span></div>
          </div>
          <div class="mt-5 rounded-xl border p-4" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-xs storage-muted">静态控制区</div>
            <div class="mt-3 space-y-2">
              <button class="w-full rounded-lg px-3 py-2 text-sm text-white" style="background: #0EA5E9;">开始充电</button>
              <button class="w-full rounded-lg border px-3 py-2 text-sm storage-title" style="border-color: var(--border-color);">开始放电</button>
              <button class="w-full rounded-lg border px-3 py-2 text-sm text-red-500" style="border-color: rgba(239,68,68,0.3);">停止策略</button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
      <DispatchQueueTable :table="page.mainTable" />
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.storage-hero,
.storage-gauge,
.storage-right {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.storage-hero {
  background:
    radial-gradient(circle at top left, rgba(16, 185, 129, 0.1), transparent 28%),
    var(--bg-secondary);
}

.storage-ring {
  border: 12px solid rgba(16, 185, 129, 0.18);
  box-shadow: inset 0 0 0 1px rgba(16, 185, 129, 0.24);
}

.storage-chip {
  background: rgba(16, 185, 129, 0.08);
  color: #10B981;
}

.storage-title { color: var(--text-primary); }
.storage-subtitle,
.storage-muted { color: var(--text-secondary); }
</style>
