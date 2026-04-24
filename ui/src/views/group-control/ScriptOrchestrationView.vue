<script setup lang="ts">
import { FileCode2, GitBranch } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'

const { page } = useP3Page('script-orchestration')
</script>

<template>
  <div class="space-y-5">
    <section class="workflow-top rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1fr_1fr]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs workflow-chip">
            <FileCode2 class="h-3.5 w-3.5" />
            DAG 编排工位
          </div>
          <h1 class="mt-3 text-3xl font-semibold workflow-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm workflow-subtitle">{{ page.subtitle }}</p>
          <div class="mt-5">
            <P3StatusStrip :data="page.statusStrip" />
          </div>
        </div>

        <section class="rounded-2xl border p-4 workflow-board">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium workflow-title">工作流 DAG</span>
            <GitBranch class="h-4 w-4 text-sky-500" />
          </div>
          <div class="mt-5 grid gap-3 md:grid-cols-4">
            <div class="workflow-node">审批</div>
            <div class="workflow-node">加载脚本</div>
            <div class="workflow-node">执行节点</div>
            <div class="workflow-node">回滚/归档</div>
          </div>
        </section>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
      <section class="rounded-xl border p-4 workflow-table">
        <h3 class="text-sm font-semibold workflow-title">工作流编排列表</h3>
        <div class="mt-4 space-y-3">
          <article
            v-for="row in page.mainTable.rows"
            :key="row.id"
            class="rounded-xl border p-4"
            style="background: var(--bg-tertiary); border-color: var(--border-color);"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="text-sm font-medium workflow-title">{{ row.workflow }}</div>
              <div class="text-[11px] workflow-subtitle">{{ row.status }}</div>
            </div>
            <div class="mt-2 grid gap-2 text-xs md:grid-cols-3">
              <div><span class="workflow-subtitle">版本</span><div class="mt-1 workflow-title">{{ row.version }}</div></div>
              <div><span class="workflow-subtitle">审批</span><div class="mt-1 workflow-title">{{ row.approval }}</div></div>
              <div><span class="workflow-subtitle">DAG</span><div class="mt-1 workflow-title">{{ row.dag }}</div></div>
            </div>
          </article>
        </div>
      </section>
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
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
}

.workflow-title { color: var(--text-primary); }
.workflow-subtitle { color: var(--text-secondary); }
</style>
