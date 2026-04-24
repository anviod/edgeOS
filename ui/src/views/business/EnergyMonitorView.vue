<script setup lang="ts">
import { Activity, BarChart3 } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import DispatchQueueTable from '@/components/p3/DispatchQueueTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'
import QualityTrendCard from '@/components/p3/QualityTrendCard.vue'
import ScenarioFlowBoard from '@/components/p3/ScenarioFlowBoard.vue'

const { page } = useP3Page('energy-monitoring')
</script>

<template>
  <div class="space-y-5">
    <section class="monitor-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1.15fr_0.85fr]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs monitor-chip">
            <Activity class="h-3.5 w-3.5" />
            能流观察哨
          </div>
          <h1 class="mt-3 text-3xl font-semibold monitor-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm monitor-subtitle">{{ page.subtitle }}</p>
          <div class="mt-5">
            <P3StatusStrip :data="page.statusStrip" />
          </div>
        </div>
        <div class="rounded-2xl border p-4 monitor-rail">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium monitor-title">峰谷观察栏</span>
            <BarChart3 class="h-4 w-4 text-sky-500" />
          </div>
          <div class="mt-4 space-y-3">
            <div
              v-for="(point, index) in page.trends[0].points"
              :key="`rail-${index}`"
              class="flex items-center gap-3"
            >
              <span class="w-10 text-[11px] monitor-subtitle">T{{ index + 1 }}</span>
              <div class="h-2 flex-1 rounded-full" style="background: rgba(148,163,184,0.18);">
                <div class="h-2 rounded-full" :style="{ width: `${point}%`, background: 'rgba(14,165,233,0.75)' }" />
              </div>
              <span class="w-12 text-right text-[11px] font-mono-num monitor-title">{{ point }}</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
      <div class="space-y-4">
        <div class="grid gap-4 xl:grid-cols-2">
          <QualityTrendCard :item="page.trends[1]" />
          <QualityTrendCard :item="page.trends[2]" />
        </div>
        <DispatchQueueTable :table="page.mainTable" />
      </div>
      <div class="space-y-4">
        <ScenarioFlowBoard :title="page.flowTitle" :nodes="page.flowNodes" />
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.monitor-hero,
.monitor-rail {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.monitor-hero {
  background:
    linear-gradient(180deg, rgba(14, 165, 233, 0.06), transparent 28%),
    var(--bg-secondary);
}

.monitor-chip {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.monitor-title { color: var(--text-primary); }
.monitor-subtitle { color: var(--text-secondary); }
</style>
