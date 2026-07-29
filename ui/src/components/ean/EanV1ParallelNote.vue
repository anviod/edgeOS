<script setup lang="ts">
import { Info } from 'lucide-vue-next'
</script>

<template>
  <div
    class="rounded-xl border p-3 flex items-start gap-2.5"
    style="background: rgba(14,165,233,0.06); border-color: rgba(14,165,233,0.18);"
  >
    <Info class="w-4 h-4 flex-shrink-0 mt-0.5" style="width:16px;height:16px;color:var(--accent-primary);" />
    <div class="text-xs leading-relaxed" style="color: var(--text-secondary);">
      <span class="font-medium" style="color: var(--text-primary);">V1 并行：</span>
      EAN 使用 <code class="font-mono" style="color: var(--accent-primary);">$edgeos/*</code>；
      V1 仍走 <code class="font-mono">edgex/*</code>（MQTT）/ <code class="font-mono">edgex.*</code>（NATS）。
      Topic 无冲突；新功能请只走 EAN，避免同一业务双写。
    </div>
  </div>
</template>
