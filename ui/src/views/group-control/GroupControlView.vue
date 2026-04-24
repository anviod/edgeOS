<script setup lang="ts">
import { Blocks, Cpu, FileCode2, Sparkles, Workflow } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3StatusStrip from '@/components/p3/P3StatusStrip.vue'
import QualityTrendCard from '@/components/p3/QualityTrendCard.vue'

const { page } = useP3Page('group-control')

const modules = [
  { label: '节点调度', path: '/group-control/node-scheduling', icon: Blocks, note: '资源池 / 队列 / 重派' },
  { label: '场景联动', path: '/group-control/scenario-linkage', icon: Sparkles, note: 'ECA 规则链' },
  { label: '函数执行', path: '/group-control/function-execution', icon: Cpu, note: '运行时 / 输入输出' },
  { label: '脚本编排', path: '/group-control/script-orchestration', icon: FileCode2, note: 'DAG / 审批 / 回滚' },
]
</script>

<template>
  <div class="space-y-5">
    <section class="gc-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1fr_0.95fr]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs gc-chip">
            <Workflow class="h-3.5 w-3.5" />
            群控指挥面
          </div>
          <h1 class="mt-3 text-3xl font-semibold gc-title">{{ page.title }}</h1>
          <p class="mt-2 text-sm gc-subtitle">{{ page.subtitle }}</p>
          <div class="mt-5">
            <P3StatusStrip :data="page.statusStrip" />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <router-link
            v-for="module in modules"
            :key="module.path"
            :to="module.path"
            class="gc-module rounded-xl border p-4"
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
        <QualityTrendCard v-for="trend in page.trends" :key="trend.title" :item="trend" />
      </div>
      <div class="space-y-4">
        <P3EventFeed :title="page.sidePanelTitle" :events="page.sideEvents" />
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
}

.gc-title { color: var(--text-primary); }
.gc-subtitle { color: var(--text-secondary); }
</style>
