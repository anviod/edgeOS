<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: 'running' | 'standby' | 'fault' | 'unknown'
}>()

const showPulse = computed(() => props.status === 'running' || props.status === 'fault')

const dotColorClass = computed(() => {
  switch (props.status) {
    case 'running': return 'bg-emerald-500'
    case 'standby': return 'bg-amber-500'
    case 'fault': return 'bg-red-500'
    default: return 'bg-neutral-400 dark:bg-neutral-500'
  }
})

const pulseColorClass = computed(() => {
  switch (props.status) {
    case 'running': return 'bg-emerald-400'
    case 'fault': return 'bg-red-400'
    default: return ''
  }
})
</script>

<template>
  <div class="relative flex h-2 w-2">
    <span
      v-if="showPulse"
      class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-75"
      :class="pulseColorClass"
    />
    <span
      class="relative inline-flex h-2 w-2 rounded-full"
      :class="dotColorClass"
    />
  </div>
</template>