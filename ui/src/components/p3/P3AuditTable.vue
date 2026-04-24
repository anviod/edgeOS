<script setup lang="ts">
import type { P3AuditRecord } from '@/types/p3'

defineProps<{
  title?: string
  records: P3AuditRecord[]
}>()
</script>

<template>
  <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ title || '操作审计' }}</h3>
    <div class="mt-4 overflow-x-auto">
      <table class="min-w-full text-xs">
        <thead>
          <tr style="color: var(--text-muted);">
            <th class="px-2 py-2 text-left">User</th>
            <th class="px-2 py-2 text-left">Action</th>
            <th class="px-2 py-2 text-left">Target</th>
            <th class="px-2 py-2 text-left">Timestamp</th>
            <th class="px-2 py-2 text-left">Result</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="record in records"
            :key="`${record.user}-${record.timestamp}-${record.action}`"
            class="border-t"
            :style="{ borderColor: 'var(--border-color)' }"
          >
            <td class="px-2 py-2 font-mono-num" style="color: var(--text-primary);">{{ record.user }}</td>
            <td class="px-2 py-2" style="color: var(--text-secondary);">{{ record.action }}</td>
            <td class="px-2 py-2" style="color: var(--text-secondary);">{{ record.target }}</td>
            <td class="px-2 py-2 font-mono-num" style="color: var(--text-secondary);">{{ record.timestamp }}</td>
            <td class="px-2 py-2" style="color: var(--accent-primary);">{{ record.result }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
