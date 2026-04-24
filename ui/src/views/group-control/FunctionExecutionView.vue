<script setup lang="ts">
import { Braces, Cpu } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'
import QualityTrendCard from '@/components/p3/QualityTrendCard.vue'

const { page } = useP3Page('function-execution')
</script>

<template>
  <div class="space-y-5">
    <section class="function-top rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1fr_360px]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs function-chip">
            <Cpu class="h-3.5 w-3.5" />
            运行时目录
          </div>
          <h1 class="mt-3 text-3xl font-semibold function-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm function-subtitle">{{ page.subtitle }}</p>
          <div class="mt-5">
            <P3StatusStrip :data="page.statusStrip" />
          </div>
        </div>

        <div class="rounded-2xl border p-4 function-console">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium function-title">输入 / 输出样例</span>
            <Braces class="h-4 w-4 text-violet-500" />
          </div>
          <pre class="mt-4 overflow-x-auto rounded-xl p-4 text-xs function-code">{
  "input": "site_power_stream",
  "runtime": "edge-fn:v1.3",
  "result": { "score": 96, "risk": "low" }
}</pre>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1fr_1fr]">
      <div class="space-y-4">
        <section class="rounded-xl border p-4 function-catalog">
          <h3 class="text-sm font-semibold function-title">函数目录卡片</h3>
          <div class="mt-4 space-y-3">
            <article
              v-for="row in page.mainTable.rows"
              :key="row.id"
              class="rounded-xl border p-4"
              style="background: var(--bg-tertiary); border-color: var(--border-color);"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="text-sm font-medium function-title">{{ row.function }}</div>
                <div class="text-[11px] function-subtitle">{{ row.status }}</div>
              </div>
              <div class="mt-2 text-xs function-subtitle">IN: {{ row.input }}</div>
              <div class="mt-1 text-xs function-subtitle">OUT: {{ row.output }}</div>
              <div class="mt-3 h-2 rounded-full" style="background: rgba(148,163,184,0.18);">
                <div class="h-2 rounded-full" :style="{ width: row.quality as string + '%', background: 'rgba(139,92,246,0.72)' }" />
              </div>
            </article>
          </div>
        </section>
      </div>

      <div class="space-y-4">
        <div class="grid gap-4 xl:grid-cols-2">
          <QualityTrendCard :item="page.trends[0]" />
          <QualityTrendCard :item="page.trends[1]" />
        </div>
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.function-top,
.function-console,
.function-catalog {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.function-top {
  background:
    linear-gradient(180deg, rgba(139, 92, 246, 0.08), transparent 30%),
    var(--bg-secondary);
}

.function-chip {
  background: rgba(139, 92, 246, 0.08);
  color: #8B5CF6;
}

.function-code {
  background: #0f172a;
  color: #cbd5e1;
}

.function-title { color: var(--text-primary); }
.function-subtitle { color: var(--text-secondary); }
</style>
