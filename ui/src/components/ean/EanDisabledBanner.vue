<script setup lang="ts">
import { AlertCircle, BookOpen } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

withDefaults(
  defineProps<{
    /** 紧凑模式（子页顶部条） */
    compact?: boolean
  }>(),
  { compact: false },
)

const router = useRouter()
</script>

<template>
  <div
    class="rounded-xl border flex gap-3"
    :class="compact ? 'p-3 items-center' : 'p-4 items-start'"
    style="background: rgba(245,158,11,0.08); border-color: rgba(245,158,11,0.2);"
  >
    <AlertCircle
      class="flex-shrink-0"
      :class="compact ? 'w-4 h-4' : 'w-5 h-5'"
      :style="{ width: compact ? '16px' : '20px', height: compact ? '16px' : '20px', color: '#F59E0B' }"
    />
    <div class="flex-1 min-w-0">
      <p class="text-sm font-medium" style="color: #F59E0B;">EAN Bus 未启用</p>
      <p v-if="!compact" class="text-xs mt-0.5" style="color: var(--text-secondary);">
        配置保存在 data/config.db（bbolt，无配置文件）。请在系统配置中设置
        <code class="font-mono">ean.enabled=true</code>，配置
        <code class="font-mono">ean.mqtt</code> / <code class="font-mono">ean.nats</code> 后重启服务。
        未启用时除 <code class="font-mono">/api/ean/health</code> 外的 <code class="font-mono">/api/ean/*</code> 返回 503。
      </p>
      <p v-else class="text-xs mt-0.5 truncate" style="color: var(--text-secondary);">
        配置 ean.enabled 与传输层后重启；可查看联合调试帮助。
      </p>
    </div>
    <button
      type="button"
      class="flex items-center gap-1 text-xs px-2.5 py-1.5 rounded-lg flex-shrink-0 transition-colors hover:bg-white/5"
      style="color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.25);"
      @click="router.push('/ean/debug')"
    >
      <BookOpen class="w-3.5 h-3.5" style="width:14px;height:14px;" />
      联调帮助
    </button>
  </div>
</template>
