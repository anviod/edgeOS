<script setup lang="ts">
import { computed } from 'vue'
import { Cpu, ShieldAlert } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'

const { page } = useP3Page('power-bms')

const matrix = computed(() =>
  page.value.mainTable.rows.map((row, index) => ({
    id: row.id,
    cluster: String(row.cluster),
    temp: String(row.temp),
    voltage: String(row.voltage),
    balance: String(row.balance),
    status: String(row.status),
    intensity: [0.22, 0.36, 0.58, 0.28][index] ?? 0.2,
  }))
)
</script>

<template>
  <div class="space-y-5">
    <section class="bms-shell rounded-2xl border p-5">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs bms-chip">
            <Cpu class="h-3.5 w-3.5" />
            矩阵诊断视图
          </div>
          <h1 class="mt-3 text-3xl font-semibold bms-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm bms-subtitle">{{ page.subtitle }}</p>
        </div>
        <div class="rounded-xl border px-4 py-3 bms-warning">
          <div class="flex items-center gap-2 text-sm font-medium text-red-500">
            <ShieldAlert class="h-4 w-4" />
            高风险簇 {{ page.kpis[3].value }} 个
          </div>
          <div class="mt-1 text-xs bms-subtitle">温差 / 压差 / 寿命指数联判</div>
        </div>
      </div>
      <div class="mt-5">
        <P3StatusStrip :data="page.statusStrip" />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
      <section class="rounded-2xl border p-4 bms-shell">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold bms-title">电池簇热区矩阵</h3>
          <span class="text-xs bms-subtitle">越红表示风险越高</span>
        </div>
        <div class="mt-4 grid gap-3 md:grid-cols-2">
          <article
            v-for="cell in matrix"
            :key="cell.id"
            class="rounded-xl border p-4"
            :style="{
              background: `linear-gradient(135deg, rgba(239,68,68,${cell.intensity}), rgba(14,165,233,0.05))`,
              borderColor: 'var(--border-color)',
            }"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="text-sm font-medium bms-title">{{ cell.cluster }}</div>
              <span class="text-[11px] bms-subtitle">{{ cell.status }}</span>
            </div>
            <div class="mt-4 grid grid-cols-2 gap-3 text-xs">
              <div class="rounded-lg border px-3 py-2 bms-panel">
                <div class="bms-subtitle">温差</div>
                <div class="mt-1 font-mono-num bms-title">{{ cell.temp }}</div>
              </div>
              <div class="rounded-lg border px-3 py-2 bms-panel">
                <div class="bms-subtitle">压差</div>
                <div class="mt-1 font-mono-num bms-title">{{ cell.voltage }}</div>
              </div>
            </div>
            <div class="mt-3 rounded-lg border px-3 py-2 text-xs bms-panel">
              <div class="bms-subtitle">均衡状态</div>
              <div class="mt-1 bms-title">{{ cell.balance }}</div>
            </div>
          </article>
        </div>
      </section>

      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
      <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      <P3AuditTable :records="page.auditRecords" title="BMS 操作审计" />
    </section>
  </div>
</template>

<style scoped>
.bms-shell,
.bms-panel,
.bms-warning {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.bms-shell {
  background:
    linear-gradient(180deg, rgba(239, 68, 68, 0.06), transparent 24%),
    var(--bg-secondary);
}

.bms-chip {
  background: rgba(239, 68, 68, 0.08);
  color: #EF4444;
}

.bms-title { color: var(--text-primary); }
.bms-subtitle { color: var(--text-secondary); }
</style>
