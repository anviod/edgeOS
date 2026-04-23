<script setup lang="ts">
import { computed } from 'vue'
import StatusIndicator from './StatusIndicator.vue'

const props = defineProps<{
  device: {
    device_id: string
    device_name?: string
    status?: string
  }
}>()

const displayName = computed(() => props.device.device_name || props.device.device_id)
const shortId = computed(() => {
  const id = props.device.device_id
  return id.length > 16 ? id.slice(0, 12) + '...' : id
})
</script>

<template>
  <div class="flex items-center gap-2">
    <StatusIndicator v-if="device.status" :status="device.status" />
    <div class="flex flex-col min-w-0">
      <span class="text-xs font-medium truncate" style="color: var(--text-primary);">
        {{ displayName }}
      </span>
      <span class="text-[10px] font-mono truncate" style="color: var(--text-muted);">
        {{ shortId }}
      </span>
    </div>
  </div>
</template>
