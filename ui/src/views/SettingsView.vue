<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Settings, Server, Radio, Info, ExternalLink, Save } from 'lucide-vue-next'

const router = useRouter()
const currentUser = ref(localStorage.getItem('username') || 'admin')

const systemInfo = [
  { label: '版本', value: 'v1.0.0' },
  { label: '框架', value: 'Go Fiber v2 + Vue3' },
  { label: '存储', value: 'BoltDB' },
  { label: '消息协议', value: 'MQTT / NATS' },
]

const mqttDefaults = [
  { label: 'Broker 地址', value: '127.0.0.1:1883' },
  { label: '用户名', value: 'admin' },
  { label: 'Client ID', value: 'edgeOS_1' },
]

const edgexTopics = [
  'edgex/nodes/register',
  'edgex/nodes/heartbeat',
  'edgex/nodes/unregister',
  'edgex/devices/report',
  'edgex/points/report',
  'edgex/data/#',
  'edgex/alerts/#',
  'edgex/responses/#',
]
</script>

<template>
  <div class="space-y-6 max-w-3xl">
    <!-- Header -->
    <div>
      <h1 class="text-xl font-bold" style="color: var(--text-primary);">系统设置</h1>
      <p class="text-sm mt-1" style="color: var(--text-secondary);">EdgeOS 平台配置信息</p>
    </div>

    <!-- System info -->
    <div class="rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="flex items-center gap-2 px-5 py-3.5" style="border-bottom: 1px solid var(--border-color);">
        <Info class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">系统信息</span>
      </div>
      <div class="divide-y" style="divide-color: var(--border-color);">
        <div v-for="item in systemInfo" :key="item.label" class="flex items-center justify-between px-5 py-3">
          <span class="text-sm" style="color: var(--text-secondary);">{{ item.label }}</span>
          <span class="text-sm font-medium" style="color: var(--text-primary);">{{ item.value }}</span>
        </div>
        <div class="flex items-center justify-between px-5 py-3">
          <span class="text-sm" style="color: var(--text-secondary);">当前用户</span>
          <span class="text-sm font-medium" style="color: var(--accent-primary);">{{ currentUser }}</span>
        </div>
      </div>
    </div>

    <!-- MQTT default config -->
    <div class="rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="flex items-center justify-between px-5 py-3.5" style="border-bottom: 1px solid var(--border-color);">
        <div class="flex items-center gap-2">
          <Radio class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
          <span class="text-sm font-semibold" style="color: var(--text-primary);">MQTT 默认配置</span>
        </div>
        <button
          @click="router.push('/middleware')"
          class="flex items-center gap-1 text-xs transition-colors hover:text-sky-400"
          style="color: var(--text-secondary);"
        >
          管理连接 <ExternalLink class="w-3 h-3" style="width:12px;height:12px;" />
        </button>
      </div>
      <div class="divide-y" style="divide-color: var(--border-color);">
        <div v-for="item in mqttDefaults" :key="item.label" class="flex items-center justify-between px-5 py-3">
          <span class="text-sm" style="color: var(--text-secondary);">{{ item.label }}</span>
          <span class="text-sm font-mono" style="color: var(--text-primary);">{{ item.value }}</span>
        </div>
      </div>
    </div>

    <!-- EdgeX topics -->
    <div class="rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="flex items-center gap-2 px-5 py-3.5" style="border-bottom: 1px solid var(--border-color);">
        <Server class="w-4 h-4" style="color: #6366F1; width:16px;height:16px;" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">EdgeX 订阅主题</span>
      </div>
      <div class="px-5 py-4 flex flex-wrap gap-2">
        <span
          v-for="topic in edgexTopics"
          :key="topic"
          class="text-xs font-mono px-2.5 py-1 rounded-lg"
          style="background: rgba(14,165,233,0.08); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.15);"
        >{{ topic }}</span>
      </div>
    </div>

    <!-- Quick actions -->
    <div class="rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="flex items-center gap-2 px-5 py-3.5" style="border-bottom: 1px solid var(--border-color);">
        <Settings class="w-4 h-4" style="color: var(--text-secondary); width:16px;height:16px;" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">快捷操作</span>
      </div>
      <div class="px-5 py-4 flex flex-wrap gap-3">
        <button
          @click="router.push('/middleware')"
          class="flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium transition-all"
          style="background: rgba(14,165,233,0.1); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.2);"
        >
          <Radio class="w-4 h-4" style="width:16px;height:16px;" />
          配置中间件
        </button>
        <button
          @click="router.push('/nodes')"
          class="flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium transition-all"
          style="background: rgba(99,102,241,0.1); color: #818CF8; border: 1px solid rgba(99,102,241,0.2);"
        >
          <Server class="w-4 h-4" style="width:16px;height:16px;" />
          查看节点
        </button>
      </div>
    </div>
  </div>
</template>
