<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { X, Plus, Trash2, ChevronDown, ChevronRight, Shield, Clock, Zap } from 'lucide-vue-next'
import type { MiddlewareConfig, MiddlewareForm } from '@/types/edgex'

const props = defineProps<{
  visible: boolean
  editing?: MiddlewareConfig | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', form: MiddlewareForm): void
}>()

const defaultForm = (): MiddlewareForm => ({
  name: '',
  type: 'mqtt',
  host: '127.0.0.1',
  port: 1883,
  username: 'admin',
  password: 'admin',
  client_id: 'edgeOS_1',
  topics: [
    'edgex/nodes/register',
    'edgex/nodes/heartbeat',
    'edgex/devices/report',
    'edgex/points/report',
    'edgex/data/#',
    'edgex/alerts/#',
    'edgex/responses/#',
  ],
  enabled: true,
  // 高级设置 - MQTT 3.1.1
  mqtt_version: 4,
  ssl: false,
  ca_file: '',
  client_cert_file: '',
  client_key_file: '',
  keep_alive: 30,
  connect_timeout: 10,
  auto_reconnect: true,
  reconnect_interval: 5,
  clean_session: true,
  qos: 1,
})

const form = ref<MiddlewareForm>(defaultForm())
const newTopic = ref('')
const submitting = ref(false)
const showAdvanced = ref(false)

const title = computed(() => props.editing ? '编辑连接' : '添加消息总线')

// 根据 SSL 状态调整端口默认值
watch(() => form.value.ssl, (ssl) => {
  if (form.value.port === 1883 || form.value.port === 8883) {
    form.value.port = ssl ? 8883 : 1883
  }
})

watch(() => props.visible, (val) => {
  if (val) {
    if (props.editing) {
      form.value = {
        name: props.editing.name,
        type: props.editing.type,
        host: props.editing.host,
        port: props.editing.port,
        username: props.editing.username,
        password: props.editing.password,
        client_id: props.editing.client_id,
        topics: [...(props.editing.topics || [])],
        enabled: props.editing.enabled,
        mqtt_version: props.editing.mqtt_version ?? 4,
        ssl: props.editing.ssl ?? false,
        ca_file: props.editing.ca_file ?? '',
        client_cert_file: props.editing.client_cert_file ?? '',
        client_key_file: props.editing.client_key_file ?? '',
        keep_alive: props.editing.keep_alive ?? 30,
        connect_timeout: props.editing.connect_timeout ?? 10,
        auto_reconnect: props.editing.auto_reconnect ?? true,
        reconnect_interval: props.editing.reconnect_interval ?? 5,
        clean_session: props.editing.clean_session ?? true,
        qos: props.editing.qos ?? 1,
      }
    } else {
      form.value = defaultForm()
    }
    newTopic.value = ''
    showAdvanced.value = false
  }
})

function addTopic() {
  const t = newTopic.value.trim()
  if (t && !form.value.topics.includes(t)) {
    form.value.topics.push(t)
  }
  newTopic.value = ''
}

function removeTopic(idx: number) {
  form.value.topics.splice(idx, 1)
}

async function handleSubmit() {
  if (!form.value.name.trim() || !form.value.host.trim()) return
  submitting.value = true
  try {
    emit('submit', { ...form.value })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-all duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-all duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        style="background: rgba(0,0,0,0.7); backdrop-filter: blur(4px);"
        @click.self="emit('close')"
      >
        <Transition
          enter-active-class="transition-all duration-200"
          enter-from-class="opacity-0 scale-95 -translate-y-2"
          enter-to-class="opacity-100 scale-100 translate-y-0"
        >
          <div
            v-if="visible"
            class="w-full max-w-lg rounded-2xl shadow-2xl overflow-hidden"
            style="background: var(--bg-secondary); border: 1px solid var(--border-color);"
          >
            <!-- Header -->
            <div class="flex items-center justify-between px-6 py-4" style="border-bottom: 1px solid var(--border-color);">
              <div>
                <h2 class="text-base font-semibold" style="color: var(--text-primary);">{{ title }}</h2>
                <p class="text-xs mt-0.5" style="color: var(--text-secondary);">配置 MQTT 消息总线参数</p>
              </div>
              <button @click="emit('close')" class="w-8 h-8 flex items-center justify-center rounded-lg transition-colors" style="color: var(--text-muted);">
                <X class="w-4 h-4" style="width:16px;height:16px;" />
              </button>
            </div>

            <!-- Body -->
            <div class="px-6 py-5 space-y-4 max-h-[65vh] overflow-y-auto" style="scrollbar-width:thin; scrollbar-color: var(--border-color) transparent;">
              <!-- Name -->
              <div class="space-y-1.5">
                <label class="text-xs font-medium" style="color: var(--text-secondary);">连接名称 *</label>
                <input
                  v-model="form.name"
                  type="text"
                  placeholder="如：主节点 MQTT"
                  class="w-full rounded-lg px-3 py-2 text-sm outline-none transition-colors"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                  @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                  @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                />
              </div>

              <!-- Host + Port -->
              <div class="flex gap-3">
                <div class="flex-1 space-y-1.5">
                  <label class="text-xs font-medium" style="color: var(--text-secondary);">服务器地址 *</label>
                  <input
                    v-model="form.host"
                    type="text"
                    placeholder="127.0.0.1"
                    class="w-full rounded-lg px-3 py-2 text-sm font-mono outline-none transition-colors"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                    @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                  />
                </div>
                <div class="w-28 space-y-1.5">
                  <label class="text-xs font-medium" style="color: var(--text-secondary);">端口</label>
                  <input
                    v-model.number="form.port"
                    type="number"
                    min="1"
                    max="65535"
                    class="w-full rounded-lg px-3 py-2 text-sm font-mono outline-none transition-colors"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                    @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                  />
                </div>
              </div>

              <!-- Username + Password -->
              <div class="flex gap-3">
                <div class="flex-1 space-y-1.5">
                  <label class="text-xs font-medium" style="color: var(--text-secondary);">用户名</label>
                  <input
                    v-model="form.username"
                    type="text"
                    class="w-full rounded-lg px-3 py-2 text-sm outline-none transition-colors"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                    @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                  />
                </div>
                <div class="flex-1 space-y-1.5">
                  <label class="text-xs font-medium" style="color: var(--text-secondary);">密码</label>
                  <input
                    v-model="form.password"
                    type="password"
                    class="w-full rounded-lg px-3 py-2 text-sm outline-none transition-colors"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                    @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                  />
                </div>
              </div>

              <!-- Client ID -->
              <div class="space-y-1.5">
                <label class="text-xs font-medium" style="color: var(--text-secondary);">Client ID</label>
                <input
                  v-model="form.client_id"
                  type="text"
                  class="w-full rounded-lg px-3 py-2 text-sm font-mono outline-none transition-colors"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                  @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                  @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                />
              </div>

              <!-- Topics -->
              <div class="space-y-2">
                <label class="text-xs font-medium" style="color: var(--text-secondary);">订阅主题</label>
                <div class="flex flex-wrap gap-1.5 rounded-lg p-2 min-h-[48px]"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color);">
                  <span
                    v-for="(topic, idx) in form.topics"
                    :key="idx"
                    class="flex items-center gap-1 rounded px-2 py-0.5 text-xs font-mono"
                    style="background: rgba(14,165,233,0.12); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.2);"
                  >
                    {{ topic }}
                    <button @click="removeTopic(idx)" class="hover:text-red-400 transition-colors ml-0.5">
                      <X class="w-3 h-3" style="width:10px;height:10px;" />
                    </button>
                  </span>
                </div>
                <div class="flex gap-2">
                  <input
                    v-model="newTopic"
                    type="text"
                    placeholder="添加主题 (支持通配符 #/+)"
                    class="flex-1 rounded-lg px-3 py-2 text-xs font-mono outline-none"
                    style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                    @keydown.enter.prevent="addTopic"
                    @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
                    @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
                  />
                  <button
                    @click="addTopic"
                    class="w-9 h-9 flex items-center justify-center rounded-lg transition-colors hover:bg-sky-500/20"
                    style="background: rgba(14,165,233,0.1); border: 1px solid rgba(14,165,233,0.2);"
                  >
                    <Plus class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
                  </button>
                </div>
              </div>

              <!-- Enabled -->
              <div class="flex items-center justify-between rounded-lg px-3 py-2.5"
                style="background: var(--bg-primary); border: 1px solid var(--border-color);">
                <span class="text-sm" style="color: var(--text-primary);">保存后自动连接</span>
                <button
                  @click="form.enabled = !form.enabled"
                  class="relative w-10 h-5 rounded-full transition-all duration-200"
                  :style="form.enabled
                    ? 'background: var(--accent-primary);'
                    : 'background: var(--text-muted);'"
                >
                  <span
                    class="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all duration-200"
                    :style="form.enabled ? 'left: calc(100% - 18px);' : 'left: 2px;'"
                  />
                </button>
              </div>

              <!-- 高级设置折叠面板 -->
              <div class="rounded-xl overflow-hidden" style="border: 1px solid var(--border-color);">
                <!-- Toggle Header -->
                <button
                  @click="showAdvanced = !showAdvanced"
                  class="w-full flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors"
                  style="background: var(--bg-primary); color: var(--text-primary);"
                >
                  <component :is="showAdvanced ? ChevronDown : ChevronRight" class="w-4 h-4" style="color: var(--text-muted); width:14px;height:14px;" />
                  <Shield class="w-4 h-4" style="color: var(--accent-primary); width:14px;height:14px;" />
                  <span>高级设置</span>
                  <span class="text-xs rounded px-1.5 py-0.5 ml-1" style="background: rgba(14,165,233,0.1); color: var(--accent-primary);">MQTT</span>
                  <span v-if="!showAdvanced" class="ml-auto text-xs" style="color: var(--text-muted);">
                    {{ form.mqtt_version === 5 ? 'MQTT 5.0' : 'MQTT 3.1.1' }} · SSL {{ form.ssl ? 'ON' : 'OFF' }} · KA {{ form.keep_alive }}s
                  </span>
                </button>

                <!-- Advanced Settings Panel -->
                <div v-show="showAdvanced" class="px-4 pb-4 space-y-4" style="background: var(--bg-primary);">
                  <!-- MQTT Version + Protocol -->
                  <div class="grid grid-cols-2 gap-3">
                    <div class="space-y-1.5">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">
                        <Zap class="inline w-3 h-3 mr-0.5 -mt-0.5" style="color: var(--text-muted); width:10px;height:10px;" />MQTT 版本
                      </label>
                      <select
                        v-model.number="form.mqtt_version"
                        class="w-full rounded-lg px-3 py-2 text-sm outline-none transition-colors appearance-none"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      >
                        <option :value="4">3.1.1</option>
                        <option :value="5">5.0</option>
                      </select>
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">QoS 等级</label>
                      <select
                        v-model.number="form.qos"
                        class="w-full rounded-lg px-3 py-2 text-sm outline-none transition-colors appearance-none"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      >
                        <option :value="0">0 - 最多一次</option>
                        <option :value="1">1 - 至少一次</option>
                        <option :value="2">2 - 恰好一次</option>
                      </select>
                    </div>
                  </div>

                  <!-- SSL/TLS -->
                  <div class="flex items-center justify-between rounded-lg px-3 py-2.5"
                    style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
                    <div>
                      <div class="text-sm" style="color: var(--text-primary);">启用 SSL/TLS</div>
                      <div class="text-xs mt-0.5" style="color: var(--text-muted);">使用 tls:// 连接（端口默认 8883）</div>
                    </div>
                    <button
                      @click="form.ssl = !form.ssl"
                      class="relative w-10 h-5 rounded-full transition-all duration-200 flex-shrink-0 ml-3"
                      :style="form.ssl
                        ? 'background: #22C55E;'
                        : 'background: var(--text-muted);'"
                    >
                      <span
                        class="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all duration-200"
                        :style="form.ssl ? 'left: calc(100% - 18px);' : 'left: 2px;'"
                      />
                    </button>
                  </div>

                  <!-- Certificate files (shown when SSL enabled) -->
                  <div v-if="form.ssl" class="space-y-3 pl-3 border-l-2" style="border-color: rgba(34,197,94,0.3);">
                    <div class="space-y-1.5">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">CA 证书 (可选)</label>
                      <input
                        v-model="form.ca_file"
                        type="text"
                        placeholder="/etc/ssl/certs/ca-certificates.crt"
                        class="w-full rounded-lg px-3 py-2 text-xs font-mono outline-none transition-colors"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      />
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">客户端证书 (可选)</label>
                      <input
                        v-model="form.client_cert_file"
                        type="text"
                        placeholder="/path/to/client.crt"
                        class="w-full rounded-lg px-3 py-2 text-xs font-mono outline-none transition-colors"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      />
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">客户端私钥 (可选)</label>
                      <input
                        v-model="form.client_key_file"
                        type="text"
                        placeholder="/path/to/client.key"
                        class="w-full rounded-lg px-3 py-2 text-xs font-mono outline-none transition-colors"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      />
                    </div>
                  </div>

                  <!-- Keep Alive + Connect Timeout -->
                  <div class="grid grid-cols-2 gap-3">
                    <div class="space-y-1.5">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">
                        <Clock class="inline w-3 h-3 mr-0.5 -mt-0.5" style="color: var(--text-muted); width:10px;height:10px;" />Keep Alive (秒)
                      </label>
                      <input
                        v-model.number="form.keep_alive"
                        type="number"
                        min="5"
                        max="300"
                        class="w-full rounded-lg px-3 py-2 text-sm font-mono outline-none transition-colors"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      />
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">
                        <Clock class="inline w-3 h-3 mr-0.5 -mt-0.5" style="color: var(--text-muted); width:10px;height:10px;" />连接超时 (秒)
                      </label>
                      <input
                        v-model.number="form.connect_timeout"
                        type="number"
                        min="3"
                        max="60"
                        class="w-full rounded-lg px-3 py-2 text-sm font-mono outline-none transition-colors"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      />
                    </div>
                  </div>

                  <!-- Auto Reconnect + Reconnect Interval -->
                  <div class="space-y-2">
                    <div class="flex items-center justify-between rounded-lg px-3 py-2.5"
                      style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
                      <span class="text-sm" style="color: var(--text-primary);">自动重连</span>
                      <button
                        @click="form.auto_reconnect = !form.auto_reconnect"
                        class="relative w-10 h-5 rounded-full transition-all duration-200 flex-shrink-0 ml-3"
                        :style="form.auto_reconnect
                          ? 'background: var(--accent-primary);'
                          : 'background: var(--text-muted);'"
                      >
                        <span
                          class="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all duration-200"
                          :style="form.auto_reconnect ? 'left: calc(100% - 18px);' : 'left: 2px;'"
                        />
                      </button>
                    </div>

                    <div v-if="form.auto_reconnect" class="space-y-1.5 pl-3 border-l-2" style="border-color: rgba(14,165,233,0.2);">
                      <label class="text-xs font-medium" style="color: var(--text-secondary);">重连间隔 (秒)</label>
                      <input
                        v-model.number="form.reconnect_interval"
                        type="number"
                        min="1"
                        max="60"
                        class="w-full rounded-lg px-3 py-2 text-sm font-mono outline-none transition-colors"
                        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
                      />
                    </div>
                  </div>

                  <!-- Clean Session -->
                  <div class="flex items-center justify-between rounded-lg px-3 py-2.5"
                    style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
                    <div>
                      <div class="text-sm" style="color: var(--text-primary);">Clean Session</div>
                      <div class="text-xs mt-0.5" style="color: var(--text-muted);">断开后清除会话状态（MQTT 3.1.1）</div>
                    </div>
                    <button
                      @click="form.clean_session = !form.clean_session"
                      class="relative w-10 h-5 rounded-full transition-all duration-200 flex-shrink-0 ml-3"
                      :style="form.clean_session
                        ? 'background: var(--accent-primary);'
                        : 'background: var(--text-muted);'"
                    >
                      <span
                        class="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all duration-200"
                        :style="form.clean_session ? 'left: calc(100% - 18px);' : 'left: 2px;'"
                      />
                    </button>
                  </div>
                </div>
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
                :disabled="submitting || !form.name.trim()"
                class="px-5 py-2 rounded-lg text-sm font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                style="background: var(--accent-primary); color: white;"
              >
                {{ submitting ? '保存中...' : '保存连接' }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
