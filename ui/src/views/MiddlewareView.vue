<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, RefreshCw, AlertCircle } from 'lucide-vue-next'
import { useMiddlewareStore } from '@/stores/middleware'
import MiddlewareCard from '@/components/edge/MiddlewareCard.vue'
import AddMiddlewareModal from '@/components/edge/AddMiddlewareModal.vue'
import type { MiddlewareConfig, MiddlewareForm } from '@/types/edgex'

const mwStore = useMiddlewareStore()
const showModal = ref(false)
const editingItem = ref<MiddlewareConfig | null>(null)
const connectingIds = ref<Set<string>>(new Set())
const errorMessage = ref('')
const errorTimeout = ref<number | null>(null)

function showError(msg: string) {
  errorMessage.value = msg
  if (errorTimeout.value) clearTimeout(errorTimeout.value)
  errorTimeout.value = window.setTimeout(() => { errorMessage.value = '' }, 5000)
}

onMounted(() => mwStore.fetchList())

function openAdd() {
  editingItem.value = null
  showModal.value = true
}

function openEdit(item: MiddlewareConfig) {
  editingItem.value = item
  showModal.value = true
}

async function handleSubmit(form: MiddlewareForm) {
  try {
    let item
    if (editingItem.value) {
      item = await mwStore.update(editingItem.value.id, form)
    } else {
      item = await mwStore.create(form)
    }

    // 如果启用了自动连接，保存后自动连接
    if (form.enabled) {
      connectingIds.value.add(item.id)
      try {
        await mwStore.connect(item.id)
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err)
        showError(`连接失败: ${msg}`)
      } finally {
        connectingIds.value.delete(item.id)
      }
    }

    showModal.value = false
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err)
    showError(`保存失败: ${msg}`)
  }
}

async function handleConnect(id: string) {
  connectingIds.value.add(id)
  try {
    await mwStore.connect(id)
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err)
    showError(`连接失败: ${msg}`)
  } finally {
    connectingIds.value.delete(id)
  }
}

async function handleDelete(id: string) {
  if (confirm('确认删除该消息总线？')) {
    await mwStore.remove(id)
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Error Toast -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition-all duration-200"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div
        v-if="errorMessage"
        class="flex items-center gap-3 px-4 py-3 rounded-xl error-toast"
      >
        <AlertCircle class="w-4 h-4 flex-shrink-0 error-icon" />
        <span class="text-sm error-message">{{ errorMessage }}</span>
        <button @click="errorMessage = ''" class="ml-auto hover:opacity-80 error-close">
          <span class="text-lg">&times;</span>
        </button>
      </div>
    </Transition>

    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold page-title">消息总线</h1>
        <p class="text-sm mt-1 page-subtitle">管理 MQTT/NATS 消息总线，订阅 EdgeX 数据主题</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="mwStore.fetchList()"
          class="flex items-center justify-center w-9 h-9 rounded-lg transition-colors btn-icon"
          :class="mwStore.loading ? 'animate-spin' : ''"
        >
          <RefreshCw class="w-4 h-4" style="width:16px;height:16px;" />
        </button>
        <button
          @click="openAdd"
          class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-all btn-primary"
        >
          <Plus class="w-4 h-4" style="width:16px;height:16px;" />
          添加连接
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="mwStore.loading" class="flex items-center justify-center py-16">
      <RefreshCw class="w-6 h-6 animate-spin loading-spinner" />
    </div>

    <!-- Empty -->
    <div v-else-if="mwStore.list.length === 0"
      class="flex flex-col items-center justify-center py-20 rounded-xl empty-state"
    >
      <div class="w-16 h-16 rounded-2xl flex items-center justify-center mb-4 empty-icon-wrapper">
        <Plus class="w-7 h-7 empty-icon" />
      </div>
      <p class="text-base font-medium mb-1 empty-title">尚未配置消息总线</p>
      <p class="text-sm mb-5 empty-subtitle">添加 MQTT 连接以开始接收 EdgeX 数据</p>
      <button
        @click="openAdd"
        class="px-5 py-2 rounded-xl text-sm font-medium btn-add"
      >
        + 添加第一个连接
      </button>
    </div>

    <!-- Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <MiddlewareCard
        v-for="mw in mwStore.list"
        :key="mw.id"
        :middleware="mw"
        :connecting="connectingIds.has(mw.id)"
        @connect="handleConnect"
        @edit="openEdit"
        @delete="handleDelete"
      />
    </div>

    <!-- Modal -->
    <AddMiddlewareModal
      :visible="showModal"
      :editing="editingItem"
      @close="showModal = false"
      @submit="handleSubmit"
    />
  </div>
</template>

<style scoped>
.error-toast {
  background: rgba(239,68,68,0.12);
  border: 1px solid rgba(239,68,68,0.3);
}
.error-icon { color: var(--destructive); }
.error-message { color: #FCA5A5; }
.error-close { color: #FCA5A5; }

.page-title { color: var(--text-primary); }
.page-subtitle { color: var(--text-secondary); }

.btn-icon {
  color: var(--text-muted);
  border: 1px solid var(--border-color);
}
.btn-icon:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.btn-primary {
  background: linear-gradient(135deg, var(--accent-primary), #1D4ED8);
  color: white;
}
.btn-primary:hover {
  opacity: 0.9;
}

.loading-spinner { color: var(--accent-primary); }

.empty-state {
  background: var(--bg-secondary);
  border: 1px dashed var(--border-color);
}
.empty-icon-wrapper {
  background: rgba(14,165,233,0.08);
}
.empty-icon { color: var(--accent-primary); }
.empty-title { color: var(--text-primary); }
.empty-subtitle { color: var(--text-secondary); }

.btn-add {
  background: rgba(14,165,233,0.1);
  color: var(--accent-primary);
  border: 1px solid rgba(14,165,233,0.2);
}
.btn-add:hover {
  background: rgba(14,165,233,0.15);
}
</style>
