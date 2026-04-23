<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { X, Send } from 'lucide-vue-next'
import type { EdgeXPointInfo } from '@/types/edgex'

const props = defineProps<{
  visible: boolean
  point: EdgeXPointInfo | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', pointId: string, value: unknown): void
}>()

const inputValue = ref<string | boolean>('')
const submitting = ref(false)

const isBool = computed(() => {
  const dt = props.point?.data_type?.toLowerCase() || ''
  return dt === 'bool' || dt === 'boolean'
})

const isNumber = computed(() => {
  const dt = props.point?.data_type?.toLowerCase() || ''
  return ['int8','int16','int32','int64','uint8','uint16','uint32','uint64','float32','float64'].includes(dt)
})

watch(() => props.visible, (val) => {
  if (val && props.point) {
    if (isBool.value) {
      inputValue.value = props.point.current_value === true || props.point.current_value === 'true' || props.point.current_value === 1
    } else {
      inputValue.value = props.point.current_value !== undefined && props.point.current_value !== null
        ? String(props.point.current_value)
        : ''
    }
  }
})

async function handleSubmit() {
  if (!props.point) return
  submitting.value = true
  let finalValue: unknown = inputValue.value
  if (isNumber.value) finalValue = Number(inputValue.value)
  try {
    emit('submit', props.point.point_id, finalValue)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-all duration-150"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
    >
      <div
        v-if="visible && point"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        style="background: rgba(0,0,0,0.7); backdrop-filter: blur(4px);"
        @click.self="emit('close')"
      >
        <Transition
          enter-active-class="transition-all duration-200"
          enter-from-class="opacity-0 scale-95"
          enter-to-class="opacity-100 scale-100"
        >
          <div
            v-if="visible"
            class="w-full max-w-md rounded-2xl shadow-2xl overflow-hidden"
            style="background: var(--bg-secondary); border: 1px solid var(--border-color);"
          >
            <!-- Header -->
            <div class="flex items-center justify-between px-6 py-4" style="border-bottom: 1px solid var(--border-color);">
              <div>
                <h2 class="text-base font-semibold" style="color: var(--text-primary);">写入点位值</h2>
                <p class="text-xs mt-0.5 font-mono" style="color: var(--text-secondary);">{{ point.point_id }}</p>
              </div>
              <button @click="emit('close')" class="w-8 h-8 flex items-center justify-center rounded-lg transition-colors" style="color: var(--text-muted);">
                <X class="w-4 h-4" style="width:16px;height:16px;" />
              </button>
            </div>

            <div class="px-6 py-5 space-y-4">
              <!-- Point info -->
              <div class="grid grid-cols-2 gap-3 rounded-xl p-3"
                style="background: var(--bg-primary); border: 1px solid var(--border-color);">
                <div>
                  <div class="text-xs" style="color: var(--text-secondary);">点位名称</div>
                  <div class="text-sm font-medium truncate" style="color: var(--text-primary);">{{ point.point_name }}</div>
                </div>
                <div>
                  <div class="text-xs" style="color: var(--text-secondary);">数据类型</div>
                  <div class="text-sm font-mono" style="color: var(--accent-primary);">{{ point.data_type }}</div>
                </div>
                <div>
                  <div class="text-xs" style="color: var(--text-secondary);">单位</div>
                  <div class="text-sm" style="color: var(--text-primary);">{{ point.units || '—' }}</div>
                </div>
                <div>
                  <div class="text-xs" style="color: var(--text-secondary);">当前值</div>
                  <div class="text-sm font-mono" style="color: #F59E0B;">
                    {{ point.current_value !== undefined ? String(point.current_value) : '—' }}
                  </div>
                </div>
              </div>

              <!-- Input based on type -->
              <div class="space-y-2">
                <label class="text-xs font-medium" style="color: var(--text-secondary);">新值</label>

                <!-- Bool toggle -->
                <div v-if="isBool" class="flex items-center justify-between rounded-xl px-4 py-3"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color);">
                  <span class="text-sm" style="color: var(--text-primary);">
                    {{ inputValue ? 'TRUE（开）' : 'FALSE（关）' }}
                  </span>
                  <button
                    @click="inputValue = !inputValue"
                    class="relative w-12 h-6 rounded-full transition-all duration-200"
                    :style="inputValue ? 'background: #10B981;' : 'background: var(--text-muted);'"
                  >
                    <span
                      class="absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-all duration-200"
                      :style="inputValue ? 'left: calc(100% - 20px);' : 'left: 4px;'"
                    />
                  </button>
                </div>

                <!-- Number input -->
                <input
                  v-else-if="isNumber"
                  v-model="inputValue"
                  type="number"
                  step="any"
                  class="w-full rounded-xl px-4 py-3 text-sm font-mono outline-none transition-colors"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                  @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                  @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                />

                <!-- String input -->
                <input
                  v-else
                  v-model="inputValue"
                  type="text"
                  class="w-full rounded-xl px-4 py-3 text-sm outline-none transition-colors"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                  @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                  @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                />
              </div>

              <!-- Warning -->
              <div class="flex items-start gap-2 rounded-lg p-3" style="background: rgba(245,158,11,0.08); border: 1px solid rgba(245,158,11,0.2);">
                <span class="text-xs leading-relaxed" style="color: #F59E0B;">
                  注意：写入命令将通过 MQTT 发送到目标设备，操作不可撤销，请确认值正确后再提交。
                </span>
              </div>
            </div>

            <!-- Footer -->
            <div class="flex items-center justify-end gap-3 px-6 py-4" style="border-top: 1px solid var(--border-color);">
              <button
                @click="emit('close')"
                class="px-4 py-2 rounded-lg text-sm transition-colors"
                style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-secondary);"
              >取消</button>
              <button
                @click="handleSubmit"
                :disabled="submitting"
                class="flex items-center gap-2 px-5 py-2 rounded-lg text-sm font-medium transition-all disabled:opacity-50"
                style="background: var(--accent-primary); color: white;"
              >
                <Send class="w-3.5 h-3.5" style="width:14px;height:14px;" />
                {{ submitting ? '发送中...' : '发送命令' }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
