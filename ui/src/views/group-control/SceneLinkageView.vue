<script setup lang="ts">
import { Sparkles } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import DispatchQueueTable from '@/components/p3/DispatchQueueTable.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'
import ScenarioFlowBoard from '@/components/p3/ScenarioFlowBoard.vue'

const { page } = useP3Page('scenario-linkage')
</script>

<template>
  <div class="space-y-5">
    <section class="scene-top rounded-2xl border p-5">
      <div>
        <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs scene-chip">
          <Sparkles class="h-3.5 w-3.5" />
          ECA 场景引擎
        </div>
        <h1 class="mt-3 text-3xl font-semibold scene-title">{{ page.title }}</h1>
        <p class="mt-2 text-sm scene-subtitle">{{ page.subtitle }}</p>
      </div>
      <div class="mt-5">
        <P3StatusStrip :data="page.statusStrip" />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
      <div class="space-y-4">
        <ScenarioFlowBoard :title="page.flowTitle" :nodes="page.flowNodes" />
        <DispatchQueueTable :table="page.mainTable" />
      </div>
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>

    <P3AuditTable :records="page.auditRecords" title="规则审计日志" />
  </div>
</template>

<style scoped>
.scene-top {
  background:
    linear-gradient(90deg, rgba(16, 185, 129, 0.08), transparent 50%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.scene-chip {
  background: rgba(16, 185, 129, 0.08);
  color: #10B981;
}

.scene-title { color: var(--text-primary); }
.scene-subtitle { color: var(--text-secondary); }
</style>
