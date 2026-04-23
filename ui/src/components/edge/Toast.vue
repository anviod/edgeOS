<script setup lang="ts">
import { ref, computed } from 'vue'
import { CheckCircle, XCircle, AlertCircle, Info, X } from 'lucide-vue-next'

type ToastType = 'success' | 'error' | 'warning' | 'info'

interface Toast {
  id: string
  type: ToastType
  message: string
  duration?: number
  onRetry?: () => void
}

const toasts = ref<Toast[]>([])

function addToast(toast: Omit<Toast, 'id'>) {
  const id = Date.now().toString()
  const newToast: Toast = {
    id,
    duration: toast.type === 'error' ? 5000 : 2000,
    ...toast,
  }
  
  toasts.value.push(newToast)
  
  setTimeout(() => {
    removeToast(id)
  }, newToast.duration)
}

function removeToast(id: string) {
  const index = toasts.value.findIndex(t => t.id === id)
  if (index > -1) {
    toasts.value.splice(index, 1)
  }
}

const toastIcon = computed(() => (type: ToastType) => {
  switch (type) {
    case 'success': return CheckCircle
    case 'error': return XCircle
    case 'warning': return AlertCircle
    case 'info': return Info
  }
})

const toastConfig = computed(() => (type: ToastType) => {
  switch (type) {
    case 'success': return { border: 'border-l-emerald-500', icon: 'text-emerald-500', text: 'text-emerald-700', darkText: '#10B981' }
    case 'error':   return { border: 'border-l-red-500',     icon: 'text-red-500',     text: 'text-red-700',     darkText: '#EF4444' }
    case 'warning': return { border: 'border-l-amber-500',   icon: 'text-amber-500',   text: 'text-amber-700',   darkText: '#F59E0B' }
    case 'info':    return { border: 'border-l-sky-500',     icon: 'text-sky-500',     text: 'text-sky-700',     darkText: '#0EA5E9' }
  }
})

defineExpose({
  success: (message: string) => addToast({ type: 'success', message }),
  error: (message: string, onRetry?: () => void) => addToast({ type: 'error', message, onRetry }),
  warning: (message: string) => addToast({ type: 'warning', message }),
  info: (message: string) => addToast({ type: 'info', message }),
})
</script>

<template>
  <Teleport to="body">
    <div class="fixed right-4 top-4 z-50 flex flex-col gap-2">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="toast-item flex w-[320px] items-start gap-3 border-l-4 border rounded-lg p-4"
          :class="toastConfig(toast.type).border"
        >
          <component
            :is="toastIcon(toast.type)"
            class="h-5 w-5 flex-shrink-0 mt-0.5"
            :class="toastConfig(toast.type).icon"
          />
          <div class="flex-1 min-w-0">
            <p class="text-sm toast-message">
              {{ toast.message }}
            </p>
            <button
              v-if="toast.onRetry"
              @click="toast.onRetry"
              class="mt-1 text-xs underline hover:no-underline"
              :class="toastConfig(toast.type).text"
            >
              重试
            </button>
          </div>
          <button
            @click="removeToast(toast.id)"
            class="toast-close flex-shrink-0 rounded-md p-0.5 transition-colors"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-item {
  background: var(--bg-secondary);
  border-top-color: var(--border-color);
  border-right-color: var(--border-color);
  border-bottom-color: var(--border-color);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}
.toast-message { color: var(--text-primary); }
.toast-close { color: var(--text-muted); }
.toast-close:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.toast-move {
  transition: transform 0.3s ease;
}
</style>
