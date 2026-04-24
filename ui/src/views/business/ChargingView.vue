<script setup lang="ts">
import { Cable, CarFront, TimerReset } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import DispatchQueueTable from '@/components/p3/DispatchQueueTable.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'
import QualityTrendCard from '@/components/p3/QualityTrendCard.vue'

const { page } = useP3Page('charging')

const sessions = [
  { label: '正在充电', value: 41, icon: Cable },
  { label: '排队车辆', value: 6, icon: CarFront },
  { label: '平均等待', value: '12m', icon: TimerReset },
]
</script>

<template>
  <div class="space-y-5">
    <section class="charging-top rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs charging-chip">
            <Cable class="h-3.5 w-3.5" />
            会话调度视图
          </div>
          <h1 class="mt-3 text-3xl font-semibold charging-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm charging-subtitle">{{ page.subtitle }}</p>
          <div class="mt-5">
            <P3StatusStrip :data="page.statusStrip" />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-3 xl:grid-cols-3">
          <article
            v-for="item in sessions"
            :key="item.label"
            class="rounded-xl border p-4 charging-session"
          >
            <component :is="item.icon" class="h-5 w-5 text-amber-500" />
            <div class="mt-4 text-xs charging-subtitle">{{ item.label }}</div>
            <div class="mt-1 text-2xl font-semibold font-mono-num charging-title">{{ item.value }}</div>
          </article>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[0.92fr_1.08fr]">
      <section class="rounded-2xl border p-4 charging-board">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold charging-title">排队与枪位车道</h3>
          <span class="text-xs charging-subtitle">A 区压力最大</span>
        </div>
        <div class="mt-4 space-y-3">
          <div class="rounded-xl border p-4 lane lane-hot">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium charging-title">A 区快充站</div>
              <div class="text-xs text-amber-500">18/20 占用</div>
            </div>
            <div class="mt-3 flex gap-2">
              <span v-for="n in 10" :key="`a-${n}`" class="lane-slot" :class="n < 9 ? 'lane-slot--busy' : 'lane-slot--idle'" />
            </div>
          </div>
          <div class="rounded-xl border p-4 lane lane-calm">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium charging-title">B 区综合站</div>
              <div class="text-xs text-emerald-500">11/16 占用</div>
            </div>
            <div class="mt-3 flex gap-2">
              <span v-for="n in 10" :key="`b-${n}`" class="lane-slot" :class="n < 6 ? 'lane-slot--busy-blue' : 'lane-slot--idle'" />
            </div>
          </div>
          <div class="rounded-xl border p-4 lane lane-risk">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium charging-title">物流南站</div>
              <div class="text-xs text-red-500">异常订单 2</div>
            </div>
            <div class="mt-3 text-xs charging-subtitle">支付回执延迟，补单队列仍在等待人工确认。</div>
          </div>
        </div>
      </section>

      <div class="grid gap-4 xl:grid-cols-2">
        <QualityTrendCard :item="page.trends[0]" />
        <QualityTrendCard :item="page.trends[1]" />
        <div class="xl:col-span-2">
          <DispatchQueueTable :table="page.mainTable" />
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
      <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
      <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
    </section>
  </div>
</template>

<style scoped>
.charging-top,
.charging-session,
.charging-board,
.lane {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.charging-top {
  background:
    linear-gradient(90deg, rgba(245, 158, 11, 0.08), transparent 45%),
    var(--bg-secondary);
}

.charging-chip {
  background: rgba(245, 158, 11, 0.08);
  color: #F59E0B;
}

.charging-title { color: var(--text-primary); }
.charging-subtitle { color: var(--text-secondary); }

.lane-slot {
  display: inline-block;
  height: 12px;
  flex: 1;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.16);
}

.lane-slot--busy { background: rgba(245, 158, 11, 0.8); }
.lane-slot--busy-blue { background: rgba(14, 165, 233, 0.8); }
.lane-slot--idle { background: rgba(148, 163, 184, 0.16); }

.lane-hot { background: rgba(245, 158, 11, 0.05); }
.lane-calm { background: rgba(14, 165, 233, 0.05); }
.lane-risk { background: rgba(239, 68, 68, 0.05); }
</style>
