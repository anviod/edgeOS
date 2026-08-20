<script setup lang="ts">
import { computed } from 'vue'
import { AlertTriangle } from 'lucide-vue-next'

const props = defineProps<{
  alarmCount?: number
}>()

const active = computed(() => (props.alarmCount ?? 0) > 0)
const display = computed(() => Math.min(props.alarmCount ?? 0, 99))
</script>

<template>
  <div
    class="hm-alarm"
    :class="{ 'hm-alarm--active': active }"
    :title="active ? `${alarmCount} 条活跃告警` : '暂无告警'"
  >
    <div class="hm-alarm-icon">
      <AlertTriangle class="h-[15px] w-[15px]" />
      <span v-if="active" class="hm-alarm-ring" />
    </div>
    <div class="hm-alarm-body">
      <span class="hm-alarm-value">{{ display }}</span>
      <span class="hm-alarm-label">告警</span>
    </div>
  </div>
</template>

<style scoped>
.hm-alarm {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 14px 0 9px;
  border-radius: 999px;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  white-space: nowrap;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.hm-alarm--active {
  border-color: rgba(239, 68, 68, 0.38);
  background: linear-gradient(180deg, rgba(239, 68, 68, 0.1), rgba(239, 68, 68, 0.03));
  animation: hm-breathe 2.4s ease-in-out infinite;
}

.hm-alarm-icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  color: var(--text-muted);
  background: var(--border-subtle);
  transition: color 0.2s ease, background 0.2s ease;
}

.hm-alarm--active .hm-alarm-icon {
  color: #EF4444;
  background: rgba(239, 68, 68, 0.14);
}

.hm-alarm-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid rgba(239, 68, 68, 0.5);
  animation: hm-ring 1.6s ease-out infinite;
  pointer-events: none;
}

@keyframes hm-ring {
  0% {
    transform: scale(0.8);
    opacity: 0.8;
  }
  100% {
    transform: scale(1.9);
    opacity: 0;
  }
}

@keyframes hm-breathe {
  0%, 100% {
    box-shadow: 0 0 0 rgba(239, 68, 68, 0);
  }
  50% {
    box-shadow: 0 0 14px rgba(239, 68, 68, 0.18);
  }
}

.hm-alarm-body {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.hm-alarm-value {
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  font-variant-numeric: tabular-nums;
  font-size: 15px;
  font-weight: 800;
  line-height: 1;
  color: var(--text-secondary);
  transition: color 0.2s ease;
}

.hm-alarm--active .hm-alarm-value {
  color: #EF4444;
}

.hm-alarm-label {
  font-size: 10px;
  letter-spacing: 0.16em;
  color: var(--text-muted);
}
</style>