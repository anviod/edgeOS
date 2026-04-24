<script setup lang="ts">
import { ref } from 'vue'
import type { P3ActionPreset, P3Tone } from '@/types/p3'

const props = defineProps<{
  title: string
  actions: P3ActionPreset[]
}>()

const active = ref<P3ActionPreset | null>(props.actions[0] ?? null)

const toneMap: Record<P3Tone, { color: string; bg: string; border: string }> = {
  sky: { color: '#0EA5E9', bg: 'rgba(14,165,233,0.08)', border: 'rgba(14,165,233,0.24)' },
  emerald: { color: '#10B981', bg: 'rgba(16,185,129,0.08)', border: 'rgba(16,185,129,0.24)' },
  amber: { color: '#F59E0B', bg: 'rgba(245,158,11,0.08)', border: 'rgba(245,158,11,0.24)' },
  red: { color: '#EF4444', bg: 'rgba(239,68,68,0.08)', border: 'rgba(239,68,68,0.24)' },
  violet: { color: '#8B5CF6', bg: 'rgba(139,92,246,0.08)', border: 'rgba(139,92,246,0.24)' },
  slate: { color: '#64748B', bg: 'rgba(100,116,139,0.08)', border: 'rgba(100,116,139,0.24)' },
}

function styleByTone(tone: P3Tone) {
  return toneMap[tone]
}
</script>

<template>
  <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ title }}</h3>
    <div class="mt-4 space-y-3">
      <button
        v-for="action in actions"
        :key="action.label"
        class="w-full rounded-xl border px-4 py-3 text-left"
        :style="{
          color: styleByTone(action.tone).color,
          background: styleByTone(action.tone).bg,
          borderColor: styleByTone(action.tone).border,
        }"
        @click="active = action"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="text-sm font-medium">{{ action.label }}</span>
          <span class="rounded-md px-2 py-1 text-[11px]" style="background: rgba(255,255,255,0.12); color: inherit;">{{ action.level }}</span>
        </div>
        <p class="mt-1 text-xs" style="color: var(--text-secondary);">{{ action.description }}</p>
      </button>
    </div>

    <div v-if="active" class="mt-4 rounded-xl border p-4" style="background: var(--bg-tertiary); border-color: var(--border-color);">
      <div class="text-xs uppercase tracking-[0.16em]" style="color: var(--text-muted);">模拟确认区</div>
      <div class="mt-2 text-sm font-medium" style="color: var(--text-primary);">{{ active.label }}</div>
      <div class="mt-1 text-xs" style="color: var(--text-secondary);">{{ active.description }}</div>
      <div v-if="active.level === 'L2'" class="mt-3 rounded-lg px-3 py-2 text-sm" style="background: rgba(14,165,233,0.08); color: #0EA5E9;">Toast: 变更已记录，待提交</div>
      <div v-else-if="active.level === 'L3'" class="mt-3 text-sm" style="color: var(--text-primary);">Confirm: 请核对执行目标与边界。</div>
      <div v-else-if="active.level === 'L4'" class="mt-3 text-sm" style="color: var(--text-primary);">Confirm + 倒计时: 2s 后可继续。</div>
      <div v-else-if="active.level === 'L5'" class="mt-3">
        <div class="text-sm" style="color: var(--text-primary);">输入确认词后才可执行。</div>
        <input
          class="mt-3 w-full rounded-lg border px-3 py-2 text-sm"
          :placeholder="`输入 ${active.confirmText || 'CONFIRM'} 确认`"
          style="background: var(--bg-secondary); border-color: var(--border-color); color: var(--text-primary);"
        />
      </div>
    </div>
  </section>
</template>
