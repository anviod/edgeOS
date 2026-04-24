<script setup lang="ts">
import type { P3TableData } from '@/types/p3'

defineProps<{
  table: P3TableData
}>()
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
              :class="column.align === 'right' ? 'text-right' : column.align === 'center' ? 'text-center' : 'text-left'"
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
              class="px-4 py-3"
              :class="column.align === 'right' ? 'text-right' : column.align === 'center' ? 'text-center' : 'text-left'"
            >
              <span
                v-if="column.key === 'status' || column.key === 'invoice' || column.key === 'settlement'"
                class="rounded-lg px-2 py-1 text-xs"
                style="background: var(--bg-tertiary); color: var(--text-primary);"
              >
                {{ row[column.key] }}
              </span>
              <span
                v-else
                :class="{ 'font-mono-num': column.key === 'amount' || column.key === 'quality' }"
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
