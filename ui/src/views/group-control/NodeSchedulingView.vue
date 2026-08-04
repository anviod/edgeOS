<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Blocks, Gauge, ArrowLeft, Clock, Database, ListChecks } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useGroupControlStore } from '@/stores/groupControl'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'

const { page } = useP3Page('node-scheduling')
const store = useGroupControlStore()
const { dateTime } = useLiveClock()

function nodeColor(status: string) {
  if (status === 'watch') return '#F59E0B'
  if (status === 'standby') return '#8B5CF6'
  return '#10B981'
}

function nodeLabel(status: string) {
  if (status === 'watch') return '观察'
  if (status === 'standby') return '可接管'
  return '稳定'
}

const sortedNodes = computed(() => [...store.nodes].sort((a, b) => b.cpu - a.cpu))
const capacity = computed(() => 100 - store.avgCpu)

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="schedule-shell rounded-2xl border p-5">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <router-link
            to="/group-control"
            class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
            style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          >
            <ArrowLeft class="h-4 w-4" />
            返回群控管理
          </router-link>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs schedule-chip">
            <Blocks class="h-3.5 w-3.5" />
            容量与队列调度
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold schedule-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #F59E0B;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 text-sm schedule-subtitle">{{ page.subtitle }}</p>
        </div>
        <div class="rounded-xl border px-4 py-3 schedule-capacity">
          <div class="flex items-center gap-2 text-sm font-medium schedule-title">
            <Gauge class="h-4 w-4 text-amber-500" />
            当前资源余量 <AnimatedNumber :value="capacity" suffix="%" />
          </div>
          <div class="mt-1 text-xs schedule-subtitle">高负载节点已进入自动迁移策略</div>
          <div class="mt-2">
            <BarMeter :value="capacity" :max="100" color="#F59E0B" :height="5" />
          </div>
        </div>
      </div>

      <!-- 调度域实时指标 -->
      <div class="mt-5">
        <LiveMetricStrip
          :items="[
            { label: '在线节点', value: store.onlineNodes, unit: '个', color: '#0EA5E9' },
            { label: '平均 CPU', value: store.avgCpu, unit: '%', color: '#F59E0B', pulse: true },
            { label: '待调度任务', value: store.pendingQueue, unit: '个', color: '#10B981', pulse: true },
            { label: '观察节点', value: store.watchNodes, unit: '个', color: '#EF4444', pulse: true },
            { label: '任务完成率', value: 98, unit: '%', color: '#10B981' },
            { label: '平均延迟', value: 16, unit: 'ms', color: '#8B5CF6' },
          ]"
        />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
      <section class="rounded-xl border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="border-b px-5 py-4" style="border-color: var(--border-color);">
          <h3 class="text-sm font-semibold schedule-title">节点容量队列表</h3>
          <p class="mt-1 text-xs schedule-subtitle">用于查看节点资源与调度队列状态</p>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead style="background: var(--bg-tertiary);">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">节点</th>
                <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">CPU</th>
                <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">内存</th>
                <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">队列</th>
                <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">调度策略</th>
                <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">状态</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="node in sortedNodes"
                :key="node.id"
                class="border-b"
                :style="{ borderColor: 'var(--border-color)' }"
              >
                <td class="px-4 py-3">
                  <span class="inline-flex items-center gap-2">
                    <span class="inline-block h-2.5 w-2.5 rounded-full" :style="{ background: nodeColor(node.status) }" />
                    <span class="font-mono-num" style="color: var(--text-primary);">{{ node.name }}</span>
                  </span>
                </td>
                <td class="px-4 py-3">
                  <div class="flex items-center justify-center gap-2">
                    <span class="font-mono-num text-xs" style="color: var(--text-secondary);">
                      <AnimatedNumber :value="node.cpu" decimals="0" suffix="%" />
                    </span>
                    <BarMeter :value="node.cpu" :max="100" :color="node.cpu > 78 ? '#EF4444' : node.cpu > 65 ? '#F59E0B' : '#10B981'" :height="4" class="w-16" />
                  </div>
                </td>
                <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                  <AnimatedNumber :value="node.memory" decimals="0" suffix="%" />
                </td>
                <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                  <AnimatedNumber :value="node.queue" />
                </td>
                <td class="px-4 py-3 text-left" style="color: var(--text-secondary);">{{ node.policy }}</td>
                <td class="px-4 py-3 text-center">
                  <span
                    class="inline-block rounded-lg px-2.5 py-1 text-xs"
                    :style="{ color: nodeColor(node.status), background: `${nodeColor(node.status)}1a` }"
                  >
                    {{ nodeLabel(node.status) }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <div class="space-y-4">
        <section class="rounded-xl border p-4 schedule-queue">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold schedule-title">待调度任务队列</h3>
            <ListChecks class="h-4 w-4 text-amber-500" />
          </div>
          <div class="mt-4 space-y-3">
            <article
              v-for="node in sortedNodes.slice(0, 3)"
              :key="node.id"
              class="rounded-lg border p-3"
              style="background: var(--bg-tertiary); border-color: var(--border-color);"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="flex items-center gap-1.5 text-sm font-medium schedule-title">
                  <Database class="h-3.5 w-3.5" :style="{ color: nodeColor(node.status) }" />
                  queue-{{ node.id.slice(2) }}
                </span>
                <span class="text-[11px] schedule-subtitle">{{ node.policy }}</span>
              </div>
              <div class="mt-2 text-xs schedule-subtitle">目标节点：{{ node.name }} / 当前队列：<AnimatedNumber :value="node.queue" /></div>
              <div class="mt-3">
                <BarMeter :value="node.cpu" :max="100" color="#F59E0B" :height="5" />
              </div>
            </article>
          </div>
        </section>
        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1fr_1fr]">
      <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold schedule-title">资源热力</h3>
            <p class="mt-1 text-xs schedule-subtitle">华东节点集群负载偏高</p>
          </div>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-amber-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
            实时
          </span>
        </div>
        <div class="mt-3">
          <BarMeter :value="store.avgCpu" :max="100" color="#F59E0B" :height="6" />
          <div class="mt-1 flex justify-between text-[11px]" style="color: var(--text-muted);">
            <span>平均 CPU <AnimatedNumber :value="store.avgCpu" suffix="%" /></span>
            <span>余量 <AnimatedNumber :value="capacity" suffix="%" /></span>
          </div>
        </div>
      </article>
      <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('scheduling')" />
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
