<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { Blocks, Cpu, FileCode2, Sparkles, Workflow, Clock } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useGroupControlStore } from '@/stores/groupControl'
import { useLiveClock } from '@/composables/useLiveClock'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'

const { page } = useP3Page('group-control')
const store = useGroupControlStore()
const { dateTime } = useLiveClock()

const modules = [
  { label: '节点调度', path: '/group-control/node-scheduling', icon: Blocks, note: '资源池 / 队列 / 重派' },
  { label: '场景联动', path: '/group-control/scenario-linkage', icon: Sparkles, note: 'ECA 规则链' },
  { label: '函数执行', path: '/group-control/function-execution', icon: Cpu, note: '运行时 / 输入输出' },
  { label: '脚本编排', path: '/group-control/script-orchestration', icon: FileCode2, note: 'DAG / 审批 / 回滚' },
]

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="gc-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1.05fr_0.95fr]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs gc-chip">
            <Workflow class="h-3.5 w-3.5" />
            群控指挥面
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold gc-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #0EA5E9;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 text-sm gc-subtitle">{{ page.subtitle }}</p>

          <!-- 群控域实时指标 -->
          <div class="mt-5">
            <LiveMetricStrip
              :items="[
                { label: '在线节点', value: store.onlineNodes, unit: '个', color: '#0EA5E9' },
                { label: '调度任务', value: store.runningTasks, unit: '个', color: '#10B981', pulse: true },
                { label: '场景成功率', value: store.sceneSuccessRate, unit: '%', decimals: 1, color: '#8B5CF6', pulse: true },
                { label: '今日触发', value: store.triggerCount, unit: '次', color: '#F59E0B', pulse: true },
                { label: '失败执行', value: store.failedExecs, unit: '次', color: '#EF4444', pulse: true },
                { label: '函数平均耗时', value: store.avgLatency, unit: 'ms', color: '#10B981' },
              ]"
            />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <router-link
            v-for="module in modules"
            :key="module.path"
            :to="module.path"
            class="gc-module rounded-xl border p-4 transition-all duration-300 hover:-translate-y-0.5"
          >
            <component :is="module.icon" class="h-5 w-5 text-sky-500" />
            <div class="mt-4 text-sm font-medium gc-title">{{ module.label }}</div>
            <div class="mt-1 text-xs gc-subtitle">{{ module.note }}</div>
          </router-link>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-2">
        <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold gc-title">节点调度质量</h3>
              <p class="mt-1 text-xs gc-subtitle">调度成功率保持高位</p>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.taskSeries" color="#0EA5E9" :height="72" />
          </div>
        </article>
        <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold gc-title">场景联动命中</h3>
              <p class="mt-1 text-xs gc-subtitle">规则执行峰值集中 10:00 前后</p>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-violet-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-violet-500" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.triggerSeries" color="#8B5CF6" :height="72" />
          </div>
        </article>
        <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold gc-title">函数执行吞吐</h3>
              <p class="mt-1 text-xs gc-subtitle">高峰期吞吐平稳</p>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-amber-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.execSeries" color="#F59E0B" :height="72" />
          </div>
        </article>
        <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold gc-title">脚本编排成功率</h3>
              <p class="mt-1 text-xs gc-subtitle">主流程稳定，长链路需优化</p>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-sky-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-sky-500" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.successSeries" color="#10B981" :height="72" />
          </div>
        </article>
      </div>
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('overview')" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.gc-hero,
.gc-module {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.gc-hero {
  background:
    radial-gradient(circle at top right, rgba(14, 165, 233, 0.1), transparent 30%),
    var(--bg-secondary);
}

.gc-chip {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.gc-module:hover {
  background: var(--bg-tertiary);
  border-color: rgba(14, 165, 233, 0.3);
}

.gc-title { color: var(--text-primary); }
.gc-subtitle { color: var(--text-secondary); }
</style>
