<script setup lang="ts">
import StatusIndicator from '@/components/edge/StatusIndicator.vue'
import type { P3FlowNode } from '@/types/p3'

defineProps<{
  title: string
  nodes: P3FlowNode[]
}>()
</script>

<template>
  <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ title }}</h3>
    <div class="mt-4 flex flex-col gap-3 lg:flex-row lg:items-center">
      <template v-for="(node, index) in nodes" :key="`${node.title}-${index}`">
        <div class="flex-1 rounded-xl border px-4 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
          <div class="flex items-center justify-between gap-2">
            <div>
              <p class="text-sm font-medium" style="color: var(--text-primary);">{{ node.title }}</p>
              <p class="mt-1 text-xs" style="color: var(--text-secondary);">{{ node.subtitle }}</p>
            </div>
            <StatusIndicator :status="node.status" />
          </div>
        </div>
        <div v-if="index < nodes.length - 1" class="hidden text-center lg:block" style="color: var(--text-muted);">→</div>
      </template>
    </div>
  </section>
</template>
