<script setup lang="ts">
import { Blocks, Gauge } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import DispatchQueueTable from '@/components/p3/DispatchQueueTable.vue'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'

const { page } = useP3Page('node-scheduling')
</script>

<template>
  <div class="space-y-5">
    <section class="schedule-shell rounded-2xl border p-5">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs schedule-chip">
            <Blocks class="h-3.5 w-3.5" />
            容量与队列调度
          </div>
          <h1 class="mt-3 text-3xl font-semibold schedule-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm schedule-subtitle">{{ page.subtitle }}</p>
        </div>
        <div class="rounded-xl border px-4 py-3 schedule-capacity">
          <div class="flex items-center gap-2 text-sm font-medium schedule-title">
            <Gauge class="h-4 w-4 text-amber-500" />
            当前资源余量 42%
          </div>
          <div class="mt-2 text-xs schedule-subtitle">高负载节点已进入自动迁移策略</div>
        </div>
      </div>
      <div class="mt-5">
        <P3StatusStrip :data="page.statusStrip" />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
      <DispatchQueueTable :table="page.mainTable" />
      <div class="space-y-4">
        <section class="rounded-xl border p-4 schedule-queue">
          <h3 class="text-sm font-semibold schedule-title">待调度任务队列</h3>
          <div class="mt-4 space-y-3">
            <article
              v-for="row in page.mainTable.rows.slice(0, 3)"
              :key="row.id"
              class="rounded-lg border p-3"
              style="background: var(--bg-tertiary); border-color: var(--border-color);"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="text-sm font-medium schedule-title">queue-{{ row.id }}</span>
                <span class="text-[11px] schedule-subtitle">{{ row.policy }}</span>
              </div>
              <div class="mt-2 text-xs schedule-subtitle">目标节点：{{ row.node }} / 当前队列：{{ row.queue }}</div>
              <div class="mt-3 h-2 rounded-full" style="background: rgba(148,163,184,0.18);">
                <div class="h-2 rounded-full" :style="{ width: row.cpu as string, background: 'rgba(245,158,11,0.75)' }" />
              </div>
            </article>
          </div>
        </section>
        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.schedule-shell,
.schedule-capacity,
.schedule-queue {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.schedule-shell {
  background:
    linear-gradient(180deg, rgba(245, 158, 11, 0.05), transparent 25%),
    var(--bg-secondary);
}

.schedule-chip {
  background: rgba(245, 158, 11, 0.08);
  color: #F59E0B;
}

.schedule-title { color: var(--text-primary); }
.schedule-subtitle { color: var(--text-secondary); }
</style>
