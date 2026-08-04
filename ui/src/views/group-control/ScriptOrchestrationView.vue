<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { FileCode2, GitBranch, ArrowLeft, Clock, ShieldCheck, Loader2 } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useGroupControlStore } from '@/stores/groupControl'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'

const { page } = useP3Page('script-orchestration')
const store = useGroupControlStore()
const { dateTime } = useLiveClock()

function workflowColor(status: string) {
  if (status === 'risk' || status === 'pending') return '#EF4444'
  if (status === 'approving') return '#F59E0B'
  return '#10B981'
}

function workflowLabel(status: string) {
  if (status === 'risk') return '高风险'
  if (status === 'approving') return '审批中'
  if (status === 'pending') return '待执行'
  return '稳定'
}

const sortedWorkflows = computed(() =>
  [...store.workflows].sort((a, b) => {
    const rank = { stable: 0, approving: 1, pending: 2, risk: 3 }
    return (rank[b.status] ?? 0) - (rank[a.status] ?? 0)
  })
)

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="workflow-top rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1fr_1fr]">
        <div>
          <router-link
            to="/group-control"
            class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
            style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          >
            <ArrowLeft class="h-4 w-4" />
            返回群控管理
          </router-link>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs workflow-chip">
            <FileCode2 class="h-3.5 w-3.5" />
            DAG 编排工位
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold workflow-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #0EA5E9;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 text-sm workflow-subtitle">{{ page.subtitle }}</p>

          <!-- 编排域实时指标 -->
          <div class="mt-5">
            <LiveMetricStrip
              :items="[
                { label: '工作流', value: 16, unit: '条', color: '#0EA5E9' },
                { label: '运行中', value: 7, unit: '条', color: '#10B981', pulse: true },
                { label: '审批待处理', value: store.approvingWorkflows, unit: '条', color: '#F59E0B', pulse: true },
                { label: '高风险动作', value: store.riskWorkflows, unit: '条', color: '#EF4444', pulse: true },
                { label: '审批通过率', value: 94, unit: '%', color: '#8B5CF6' },
                { label: '平均执行时长', value: 8.2, unit: '分', color: '#10B981' },
              ]"
            />
          </div>
        </div>

        <section class="rounded-2xl border p-4 workflow-board">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium workflow-title">工作流 DAG</span>
            <div class="flex items-center gap-2">
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
              <GitBranch class="h-4 w-4 text-sky-500" />
            </div>
          </div>
          <div class="mt-5 grid gap-3 md:grid-cols-4">
            <div class="workflow-node">
              <div class="workflow-node-status" style="color: #F59E0B;">
                <ShieldCheck class="h-4 w-4 mx-auto" />
              </div>
              <div class="mt-1">审批</div>
              <div class="workflow-node-sub">L3-L5 门禁</div>
            </div>
            <div class="workflow-node">
              <div class="workflow-node-status" style="color: #0EA5E9;">
                <Loader2 class="h-4 w-4 mx-auto animate-spin" />
              </div>
              <div class="mt-1">加载脚本</div>
              <div class="workflow-node-sub">版本校验</div>
            </div>
            <div class="workflow-node">
              <div class="workflow-node-status" style="color: #10B981;">
                <GitBranch class="h-4 w-4 mx-auto" />
              </div>
              <div class="mt-1">执行节点</div>
              <div class="workflow-node-sub">串 / 并行</div>
            </div>
            <div class="workflow-node">
              <div class="workflow-node-status" style="color: #8B5CF6;">
                <FileCode2 class="h-4 w-4 mx-auto" />
              </div>
              <div class="mt-1">回滚/归档</div>
              <div class="workflow-node-sub">失败保护</div>
            </div>
          </div>
        </section>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
      <section class="rounded-xl border p-4 workflow-table">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold workflow-title">工作流编排列表</h3>
            <p class="mt-1 text-xs workflow-subtitle">关注版本、审批、执行 DAG 与日志结果</p>
          </div>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
            实时
          </span>
        </div>
        <div class="mt-4 space-y-3">
          <article
            v-for="wf in sortedWorkflows"
            :key="wf.id"
            class="rounded-xl border p-4"
            style="background: var(--bg-tertiary); border-color: var(--border-color);"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2 font-mono-num text-sm font-medium workflow-title">
                <span class="inline-block h-2 w-2 animate-pulse rounded-full" :style="{ background: workflowColor(wf.status) }" />
                {{ wf.name }}
              </div>
              <span class="rounded-md px-2 py-0.5 text-[11px] font-medium" :style="{ color: workflowColor(wf.status), background: `${workflowColor(wf.status)}1a` }">
                {{ workflowLabel(wf.status) }}
              </span>
            </div>
            <div class="mt-3 grid gap-3 text-xs md:grid-cols-3">
              <div>
                <span class="workflow-subtitle">版本</span>
                <div class="mt-1 font-mono-num workflow-title">{{ wf.version }}</div>
              </div>
              <div>
                <span class="workflow-subtitle">审批</span>
                <div class="mt-1 font-mono-num" :style="{ color: wf.approval === '高风险' ? '#EF4444' : wf.approval === '待复核' ? '#F59E0B' : 'var(--text-primary)' }">{{ wf.approval }}</div>
              </div>
              <div>
                <span class="workflow-subtitle">DAG</span>
                <div class="mt-1 font-mono-num workflow-title">{{ wf.dag }}</div>
              </div>
            </div>
            <div class="mt-3">
              <div class="mb-1 flex justify-between text-[11px]">
                <span class="workflow-subtitle">Quality</span>
                <span class="font-mono-num workflow-title"><AnimatedNumber :value="wf.quality" /></span>
              </div>
              <BarMeter :value="wf.quality" :max="100" :color="workflowColor(wf.status)" :height="5" />
            </div>
          </article>
        </div>
      </section>
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('workflow')" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>

    <P3AuditTable :records="page.auditRecords" title="编排审计" />
  </div>
</template>

<style scoped>
.workflow-top,
.workflow-board,
.workflow-table,
.workflow-node {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.workflow-top {
  background:
    linear-gradient(90deg, rgba(14, 165, 233, 0.08), rgba(139, 92, 246, 0.06) 65%, transparent),
    var(--bg-secondary);
}

.workflow-chip {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.workflow-node {
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px 12px;
  text-align: center;
  color: var(--text-primary);
  transition: all 0.3s ease;
}

.workflow-node:hover {
  border-color: rgba(14, 165, 233, 0.4);
  background: var(--bg-tertiary);
  transform: translateY(-2px);
}

.workflow-node-sub {
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-muted);
}

.workflow-title { color: var(--text-primary); }
.workflow-subtitle { color: var(--text-secondary); }
</style>
