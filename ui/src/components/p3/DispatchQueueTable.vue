<script setup lang="ts">
import type { P3TableData, P3TableRow, P3Tone } from '@/types/p3'

defineProps<{
  table: P3TableData
}>()

const accentMap: Record<P3Tone, string> = {
  sky: '#0EA5E9',
  emerald: '#10B981',
  amber: '#F59E0B',
  red: '#EF4444',
  violet: '#8B5CF6',
  slate: '#64748B',
}

function rowAccent(row: P3TableRow): string {
  return row.accent ? accentMap[row.accent] : 'transparent'
}
</script>

<template>
  <section class="rounded-xl border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
    <div class="border-b px-5 py-4" style="border-color: var(--border-color);">
      <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ table.title }}</h3>
      <p class="mt-1 text-xs" style="color: var(--text-secondary);">{{ table.description }}</p>
    </div>
    <div class="overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead style="background: var(--bg-tertiary);">
          <tr>
            <th
              v-for="column in table.columns"
              :key="column.key"
              class="px-4 py-3 text-xs font-semibold"
              :class="{
                'text-left': !column.align || column.align === 'left',
                'text-center': column.align === 'center',
                'text-right': column.align === 'right',
              }"
              style="color: var(--text-secondary);"
            >
              {{ column.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in table.rows"
            :key="row.id"
            class="border-b"
            :style="{ borderColor: 'var(--border-color)' }"
          >
            <td
              v-for="column in table.columns"
              :key="`${row.id}-${column.key}`"
              class="px-4 py-3 align-middle"
              :class="{
                'text-left': !column.align || column.align === 'left',
                'text-center': column.align === 'center',
                'text-right': column.align === 'right',
              }"
            >
              <span
                v-if="column.key === table.columns[0].key"
                class="inline-flex items-center gap-2"
              >
                <span class="inline-block h-2.5 w-2.5 rounded-full" :style="{ background: rowAccent(row) }" />
                <span style="color: var(--text-primary);">{{ row[column.key] }}</span>
              </span>
              <span
                v-else
                :class="{ 'font-mono-num': column.key === 'quality' || column.key === 'latency' || column.key === 'loss' || column.key === 'amount' }"
                style="color: var(--text-secondary);"
              >
                {{ row[column.key] }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
