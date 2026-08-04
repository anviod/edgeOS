<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Download, Receipt, ArrowLeft, Clock, FileDown, RefreshCw } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useBusinessStore } from '@/stores/business'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'

const { page } = useP3Page('ledger')
const store = useBusinessStore()
const { dateTime } = useLiveClock()

const exporting = ref<string | null>(null)

function exportLedger(kind: 'daily' | 'abnormal') {
  const name = kind === 'daily' ? 'daily' : 'abnormal'
  exporting.value = name
  setTimeout(() => {
    const rows = store.ledgerEntries
    const header = kind === 'daily' ? '台账号,金额,开票状态,结算状态,Quality,状态' : '台账号,金额,开票状态,结算状态,Quality,异常'
    const body = rows
      .map(e => [e.no, e.amount, e.invoice, e.settlement, e.quality, e.status === 'error' ? '异常' : '正常'].join(','))
      .join('\n')
    const csv = `${header}\n${body}`
    const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `ledger-${kind}-${new Date().toISOString().slice(0, 10)}.csv`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    exporting.value = null
  }, 1200)
}

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="ledger-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1fr_320px]">
        <div>
          <router-link
            to="/business-center"
            class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
            style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          >
            <ArrowLeft class="h-4 w-4" />
            返回业务中心
          </router-link>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs ledger-chip">
            <Receipt class="h-3.5 w-3.5" />
            财务台账工作台
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold ledger-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #6366F1;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 text-sm ledger-subtitle">{{ page.subtitle }}</p>

          <!-- 账务域实时指标 -->
          <div class="mt-5">
            <LiveMetricStrip
              :items="[
                { label: '应收账单', value: 418, unit: '笔', color: '#0EA5E9' },
                { label: '待对账金额', value: store.pendingAmount, unit: '万元', decimals: 1, color: '#F59E0B', pulse: true },
                { label: '已结算率', value: store.settleRate, unit: '%', decimals: 1, color: '#10B981', pulse: true },
                { label: '异常条目', value: 3, unit: '项', color: '#EF4444' },
                { label: '开票完成率', value: 88, unit: '%', color: '#8B5CF6' },
                { label: '结算准确率', value: 97, unit: '%', color: '#10B981' },
              ]"
            />
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
              <div class="mt-2 flex items-center gap-1.5 text-[11px] text-emerald-500">
                <RefreshCw class="h-3 w-3 animate-spin" />
                数据实时同步中
              </div>
            </div>
            <button
              class="flex w-full items-center justify-center gap-2 rounded-lg bg-sky-500 px-3 py-2 text-white transition-all hover:bg-sky-600 disabled:opacity-60"
              :disabled="exporting === 'daily'"
              @click="exportLedger('daily')"
            >
              <FileDown class="h-4 w-4" />
              {{ exporting === 'daily' ? '导出中…' : '导出财务日报' }}
            </button>
            <button
              class="flex w-full items-center justify-center gap-2 rounded-lg border px-3 py-2 ledger-title transition-all hover:bg-white/5 disabled:opacity-60"
              style="border-color: var(--border-color);"
              :disabled="exporting === 'abnormal'"
              @click="exportLedger('abnormal')"
            >
              <FileDown class="h-4 w-4 text-red-400" />
              {{ exporting === 'abnormal' ? '导出中…' : '导出异常账单清单' }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
      <div class="space-y-4">
        <section class="rounded-xl border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="border-b px-5 py-4" style="border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-sm font-semibold ledger-title">台账状态表</h3>
                <p class="mt-1 text-xs ledger-subtitle">{{ page.mainTable.description }}</p>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full text-sm">
              <thead style="background: var(--bg-tertiary);">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">台账号</th>
                  <th class="px-4 py-3 text-right text-xs font-semibold" style="color: var(--text-secondary);">金额</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">开票状态</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">结算状态</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">Quality</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">状态</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="entry in store.ledgerEntries"
                  :key="entry.id"
                  class="border-b"
                  :style="{ borderColor: 'var(--border-color)' }"
                >
                  <td class="px-4 py-3 font-mono-num" style="color: var(--text-primary);">{{ entry.no }}</td>
                  <td class="px-4 py-3 text-right font-mono-num" style="color: var(--text-secondary);">
                    ¥<AnimatedNumber :value="entry.amount" decimals="0" />
                  </td>
                  <td class="px-4 py-3 text-center">
                    <span class="rounded-lg px-2 py-1 text-xs" style="background: var(--bg-tertiary); color: var(--text-primary);">
                      {{ entry.invoice }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-center">
                    <span class="rounded-lg px-2 py-1 text-xs" style="background: var(--bg-tertiary); color: var(--text-primary);">
                      {{ entry.settlement }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                    <AnimatedNumber :value="entry.quality" />
                  </td>
                  <td class="px-4 py-3 text-center">
                    <span
                      class="rounded-lg px-2 py-1 text-xs"
                      :style="{
                        color: entry.status === 'error' ? '#EF4444' : entry.status === 'watch' ? '#F59E0B' : '#10B981',
                        background: entry.status === 'error' ? 'rgba(239,68,68,0.1)' : entry.status === 'watch' ? 'rgba(245,158,11,0.1)' : 'rgba(16,185,129,0.1)',
                      }"
                    >
                      {{ entry.status === 'error' ? '异常' : entry.status === 'watch' ? '观察' : '正常' }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
        <P3AuditTable :records="page.auditRecords" title="财务审计日志" />
      </div>
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('ledger')" />
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
