<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { Braces, Cpu, ArrowLeft, Copy, Check, Clock, Timer } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useGroupControlStore } from '@/stores/groupControl'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'

const { page } = useP3Page('function-execution')
const store = useGroupControlStore()
const { dateTime } = useLiveClock()

const copied = ref(false)

const sampleCode = `{
  "input": "site_power_stream",
  "runtime": "edge-fn:v1.3",
  "result": { "score": 96, "risk": "low" }
}`

async function copySample() {
  try {
    await navigator.clipboard.writeText(sampleCode)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1600)
  } catch {
    copied.value = false
  }
}

function funcColor(status: string) {
  if (status === 'retry') return '#EF4444'
  if (status === 'watch') return '#F59E0B'
  if (status === 'running') return '#0EA5E9'
  return '#10B981'
}

function funcLabel(status: string) {
  if (status === 'retry') return '失败重试'
  if (status === 'watch') return '观察'
  if (status === 'running') return '运行中'
  return '稳定'
}

const sortedFunctions = computed(() => [...store.functions].sort((a, b) => b.latency - a.latency))

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="function-top rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1fr_360px]">
        <div>
          <router-link
            to="/group-control"
            class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
            style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          >
            <ArrowLeft class="h-4 w-4" />
            返回群控管理
          </router-link>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs function-chip">
            <Cpu class="h-3.5 w-3.5" />
            运行时目录
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold function-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #8B5CF6;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 text-sm function-subtitle">{{ page.subtitle }}</p>

          <!-- 函数域实时指标 -->
          <div class="mt-5">
            <LiveMetricStrip
              :items="[
                { label: '函数目录', value: 28, unit: '个', color: '#0EA5E9' },
                { label: '今日执行', value: store.execCount, unit: '次', color: '#10B981', pulse: true },
                { label: '平均耗时', value: store.avgLatency, unit: 'ms', color: '#8B5CF6', pulse: true },
                { label: '失败执行', value: store.retryCount, unit: '次', color: '#EF4444', pulse: true },
                { label: '热启动命中', value: 88, unit: '%', color: '#F59E0B' },
                { label: 'P95 耗时', value: 312, unit: 'ms', color: '#F59E0B' },
              ]"
            />
          </div>
        </div>

        <div class="rounded-2xl border p-4 function-console">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium function-title">输入 / 输出样例</span>
            <div class="flex items-center gap-2">
              <button
                type="button"
                @click="copySample"
                class="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs transition-colors hover:bg-white/5"
                style="color: var(--text-secondary); border: 1px solid var(--border-color);"
              >
                <component :is="copied ? Check : Copy" class="h-3 w-3" />
                {{ copied ? '已复制' : '复制' }}
              </button>
              <Braces class="h-4 w-4 text-violet-500" />
            </div>
          </div>
          <pre class="mt-4 overflow-x-auto rounded-xl p-4 text-xs function-code">{{ sampleCode }}</pre>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1fr_1fr]">
      <div class="space-y-4">
        <section class="rounded-xl border p-4 function-catalog">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold function-title">函数目录卡片</h3>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              实时
            </span>
          </div>
          <div class="mt-4 space-y-3">
            <article
              v-for="fn in sortedFunctions"
              :key="fn.id"
              class="rounded-xl border p-4"
              style="background: var(--bg-tertiary); border-color: var(--border-color);"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-2 font-mono-num text-sm font-medium function-title">
                  <span class="inline-block h-2 w-2 animate-pulse rounded-full" :style="{ background: funcColor(fn.status) }" />
                  {{ fn.name }}
                </div>
                <span class="rounded-md px-2 py-0.5 text-[11px] font-medium" :style="{ color: funcColor(fn.status), background: `${funcColor(fn.status)}1a` }">
                  {{ funcLabel(fn.status) }}
                </span>
              </div>
              <div class="mt-2 flex gap-4 text-xs">
                <span class="function-subtitle">IN: <span style="color: var(--text-primary);">{{ fn.input }}</span></span>
                <span class="function-subtitle">OUT: <span style="color: var(--text-primary);">{{ fn.output }}</span></span>
              </div>
              <div class="mt-3 grid grid-cols-2 gap-3">
                <div>
                  <div class="mb-1 flex justify-between text-[11px]">
                    <span class="function-subtitle">耗时</span>
                    <span class="font-mono-num" :style="{ color: fn.latency > 390 ? '#EF4444' : fn.latency > 300 ? '#F59E0B' : 'var(--text-primary)' }">
                      <AnimatedNumber :value="fn.latency" suffix="ms" />
                    </span>
                  </div>
                  <BarMeter :value="fn.latency" :max="460" :color="fn.latency > 390 ? '#EF4444' : fn.latency > 300 ? '#F59E0B' : '#8B5CF6'" :height="5" />
                </div>
                <div>
                  <div class="mb-1 flex justify-between text-[11px]">
                    <span class="function-subtitle">Quality</span>
                    <span class="font-mono-num" style="color: var(--text-primary);"><AnimatedNumber :value="fn.quality" /></span>
                  </div>
                  <BarMeter :value="fn.quality" :max="100" color="#10B981" :height="5" />
                </div>
              </div>
            </article>
          </div>
        </section>
      </div>

      <div class="space-y-4">
        <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold function-title">P95 耗时趋势</h3>
              <p class="mt-1 text-xs function-subtitle">预热后明显下降</p>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-violet-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-violet-500" />
              实时
            </span>
          </div>
          <div class="mt-3 flex items-center gap-3">
            <Timer class="h-4 w-4 text-violet-500" />
            <span class="font-mono-num text-2xl font-semibold" style="color: #8B5CF6;">
              <AnimatedNumber :value="store.avgLatency" suffix="ms" />
            </span>
            <span class="text-xs function-subtitle">函数平均耗时</span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.latencySeries" color="#8B5CF6" :height="64" />
          </div>
        </article>
        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('function')" />
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
