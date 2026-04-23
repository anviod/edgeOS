<script setup lang="ts">
import { ref, computed } from 'vue'
import { cn } from '@/lib/utils'
import StatusIndicator from './StatusIndicator.vue'
import ProtocolBadge from './ProtocolBadge.vue'
import EdgeDeviceCell from './EdgeDeviceCell.vue'
import { formatRelativeTime } from '@/lib/helpers'
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-vue-next'

type Column = {
  key: string
  label: string
  align?: 'left' | 'center' | 'right'
  width?: string
}

type Row = Record<string, any>

const props = defineProps<{
  columns: Column[]
  data: Row[]
  selectedId?: string
  density?: 'compact' | 'normal' | 'comfortable'
}>()

const emit = defineEmits<{
  select: [id: string]
  edit: [row: Row]
  delete: [row: Row]
}>()

const rowHeight = computed(() => {
  switch (props.density) {
    case 'compact': return 'h-8'
    case 'comfortable': return 'h-12'
    default: return 'h-10'
  }
})

const alignClass = (align?: string) => {
  switch (align) {
    case 'center': return 'text-center'
    case 'right': return 'text-right'
    default: return 'text-left'
  }
}

function handleRowClick(row: Row) {
  emit('select', row.id)
}

function handleEdit(row: Row, event: Event) {
  event.stopPropagation()
  emit('edit', row)
}

function handleDelete(row: Row, event: Event) {
  event.stopPropagation()
  emit('delete', row)
}

function renderCell(row: Row, column: Column) {
  const value = row[column.key]

  if (column.key === 'device') {
    return h(EdgeDeviceCell, { device: value })
  }

  if (column.key === 'protocol') {
    return h(ProtocolBadge, { protocol: value })
  }

  if (column.key === 'status') {
    return h('div', { class: 'flex items-center gap-2' }, [
      h(StatusIndicator, { status: value }),
      h('span', { class: 'text-xs', style: 'color: var(--text-secondary);' }, value)
    ])
  }

  if (column.key === 'ip') {
    return h('span', { class: 'font-mono text-xs', style: 'color: var(--text-muted);' }, value)
  }

  if (column.key === 'lastCommunication') {
    return h('span', { class: 'text-xs', style: 'color: var(--text-muted);' }, formatRelativeTime(new Date(value)))
  }

  if (column.key === 'actions') {
    return h('div', { class: 'flex items-center justify-end gap-2' }, [
      h('button', {
        onClick: (e: Event) => handleEdit(row, e),
        class: 'p-1 rounded transition-colors action-btn-edit'
      }, h(Pencil, { class: 'h-4 w-4', style: 'color: var(--text-muted);' })),
      h('button', {
        onClick: (e: Event) => handleDelete(row, e),
        class: 'p-1 rounded transition-colors action-btn-delete'
      }, h(Trash2, { class: 'h-4 w-4', style: 'color: var(--destructive);' }))
    ])
  }

  return value
}
</script>

<template>
  <div class="w-full overflow-x-auto scrollbar-industrial">
    <table class="w-full border-collapse data-table">
      <thead class="table-head">
        <tr class="border-b" style="border-color: var(--border-color);">
          <th
            v-for="column in columns"
            :key="column.key"
            :class="cn(
              'h-10 px-3 text-xs font-semibold table-header-cell',
              alignClass(column.align)
            )"
            :style="{ width: column.width }"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="row in data"
          :key="row.id"
          :class="cn(
            'border-b cursor-pointer transition-colors table-row',
            rowHeight,
            selectedId === row.id && 'border-l-2 bg-sky-50/50 dark:bg-sky-900/10'
          )"
          style="border-color: var(--border-color);"
          @click="handleRowClick(row)"
        >
          <td
            v-for="column in columns"
            :key="column.key"
            :class="cn('px-3 py-0', alignClass(column.align))"
          >
            <component :is="() => renderCell(row, column)" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.data-table { border: 1px solid var(--border-color); border-radius: 8px; overflow: hidden; }
.table-head { background: var(--bg-secondary); }
.table-header-cell { color: var(--text-secondary); }
.table-row { background: var(--bg-primary); }
.table-row:hover { background: var(--bg-hover); }
.action-btn-edit:hover { background: var(--bg-hover); }
.action-btn-edit:hover svg { color: var(--accent-primary) !important; }
.action-btn-delete:hover { background: rgba(239,68,68,0.1); }
.action-btn-delete:hover svg { color: #EF4444 !important; }
</style>