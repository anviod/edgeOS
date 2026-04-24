<script setup lang="ts">
import { Download, Receipt } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import BusinessKpiStrip from '@/components/p3/BusinessKpiStrip.vue'
import LedgerStatusTable from '@/components/p3/LedgerStatusTable.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'

const { page } = useP3Page('ledger')
</script>

<template>
  <div class="space-y-5">
    <section class="ledger-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1fr_320px]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs ledger-chip">
            <Receipt class="h-3.5 w-3.5" />
            财务台账工作台
          </div>
          <h1 class="mt-3 text-3xl font-semibold ledger-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm ledger-subtitle">{{ page.subtitle }}</p>
          <div class="mt-5">
            <P3StatusStrip :data="page.statusStrip" />
          </div>
        </div>
        <div class="rounded-2xl border p-5 ledger-export">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium ledger-title">报表导出区</span>
            <Download class="h-4 w-4 text-sky-500" />
          </div>
          <div class="mt-4 space-y-3 text-sm">
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="ledger-subtitle text-xs">日报草稿</div>
              <div class="mt-1 ledger-title">已生成，可导出 PDF / XLSX</div>
            </div>
            <button class="w-full rounded-lg px-3 py-2 bg-sky-500 text-white">导出财务日报</button>
            <button class="w-full rounded-lg border px-3 py-2 ledger-title" style="border-color: var(--border-color);">导出异常账单清单</button>
          </div>
        </div>
      </div>
    </section>

    <BusinessKpiStrip :items="page.kpis" />

    <section class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
      <div class="space-y-4">
        <LedgerStatusTable :table="page.mainTable" />
        <P3AuditTable :records="page.auditRecords" title="财务审计日志" />
      </div>
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.ledger-hero,
.ledger-export {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.ledger-hero {
  background:
    linear-gradient(90deg, rgba(99, 102, 241, 0.07), transparent 55%),
    var(--bg-secondary);
}

.ledger-chip {
  background: rgba(99, 102, 241, 0.08);
  color: #6366F1;
}

.ledger-title { color: var(--text-primary); }
.ledger-subtitle { color: var(--text-secondary); }
</style>
