<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { X } from 'lucide-vue-next'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  title: string
  description?: string
  actionName?: string
  actionDescription?: string
  variant?: 'danger' | 'warning' | 'info'
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const countdown = ref(2)
let timer: NodeJS.Timeout | null = null

onMounted(() => {
  if (open.value) {
    startCountdown()
  }
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})

watch(open, (newVal) => {
  if (newVal) {
    startCountdown()
  } else {
    if (timer) {
      clearInterval(timer)
    }
  }
})

function startCountdown() {
  countdown.value = 2
  timer = setInterval(() => {
    if (countdown.value > 0) {
      countdown.value--
    } else {
      if (timer) clearInterval(timer)
    }
  }, 1000)
}

function handleConfirm() {
  if (countdown.value > 0) return
  emit('confirm')
  open.value = false
}

function handleCancel() {
  emit('cancel')
  open.value = false
}

function handleOverlayClick(event: MouseEvent) {
  if (event.target === event.currentTarget) {
    handleCancel()
  }
}

const variantBorderClass = computed(() => {
  switch (props.variant) {
    case 'danger': return 'border-red-400/40'
    case 'warning': return 'border-amber-400/40'
    default: return ''
  }
})

const buttonVariantClasses = computed(() => {
  switch (props.variant) {
    case 'danger':
      return 'bg-red-600 hover:bg-red-700 text-white'
    case 'warning':
      return 'bg-amber-600 hover:bg-amber-700 text-white'
    default:
      return 'bg-sky-600 hover:bg-sky-700 text-white'
  }
})
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click="handleOverlayClick"
      >
        <Transition name="slide-up">
          <div
            v-if="open"
            class="dialog-panel relative w-[400px] border rounded-xl p-0 overflow-hidden"
            :class="variantBorderClass"
          >
            <div class="dialog-header flex items-center justify-between border-b p-4">
              <h3 class="text-lg font-semibold dialog-title">
                {{ title }}
              </h3>
              <button
                @click="handleCancel"
                class="dialog-close-btn flex h-7 w-7 items-center justify-center rounded-lg transition-colors"
              >
                <X class="h-4 w-4" />
              </button>
            </div>

            <div class="p-4">
              <p class="text-sm dialog-desc">
                {{ description || `此操作将${actionDescription || actionName}，是否继续？` }}
              </p>
            </div>

            <div class="dialog-footer flex items-center justify-end gap-2 border-t p-4">
              <button
                @click="handleCancel"
                class="dialog-cancel-btn rounded-lg border px-4 py-2 text-sm transition-colors"
              >
                取消
              </button>
              <button
                @click="handleConfirm"
                :disabled="countdown > 0"
                class="rounded-lg px-4 py-2 text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                :class="buttonVariantClasses"
              >
                {{ countdown > 0 ? `确认 (${countdown})` : '确认' }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.dialog-panel {
  background: var(--bg-secondary);
  border-color: var(--border-color);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}
.dialog-header {
  border-color: var(--border-color);
}
.dialog-title { color: var(--text-primary); }
.dialog-desc  { color: var(--text-secondary); }
.dialog-close-btn {
  color: var(--text-muted);
}
.dialog-close-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.dialog-footer {
  border-color: var(--border-color);
}
.dialog-cancel-btn {
  color: var(--text-primary);
  border-color: var(--border-color);
  background: transparent;
}
.dialog-cancel-btn:hover {
  background: var(--bg-hover);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.2s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  opacity: 0;
  transform: translateY(20px);
}
</style>
