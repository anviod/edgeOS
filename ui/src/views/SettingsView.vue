<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  Settings,
  Server,
  Radio,
  Info,
  ExternalLink,
  RotateCcw,
  Download,
  Database,
  ShieldCheck,
  Boxes,
  User,
  Cpu,
  HardDrive,
  Activity,
  LayoutGrid,
} from 'lucide-vue-next'
import { systemApi } from '@/api'
import DataManagement from '@/views/components/DataManagement.vue'
import DangerDialog from '@/components/edge/DangerDialog.vue'

const router = useRouter()
const currentUser = ref(localStorage.getItem('username') || 'admin')

const systemInfo = [
  { label: '版本', value: 'v1.0.0', icon: Cpu },
  { label: '框架', value: 'Go Fiber v2 + Vue3', icon: Boxes },
  { label: '存储', value: 'BoltDB', icon: HardDrive },
  { label: '消息协议', value: 'MQTT / NATS', icon: Activity },
]

const mqttDefaults = [
  { label: 'Broker 地址', value: '127.0.0.1:1883' },
  { label: '用户名', value: 'admin' },
  { label: 'Client ID', value: 'edgeOS_1' },
]

const edgeCoreTopics = [
  'edgeCore/nodes/register',
  'edgeCore/nodes/heartbeat',
  'edgeCore/nodes/unregister',
  'edgeCore/devices/report',
  'edgeCore/points/report',
  'edgeCore/data/#',
  'edgeCore/alerts/#',
  'edgeCore/responses/#',
]

const roleBadge = computed(() => 'admin')

// ── 服务重启状态 | Service restart state ──
const showRestartConfirm = ref(false)
const restarting = ref(false)
const restartMessage = ref('')

async function handleRestart() {
  showRestartConfirm.value = false
  restarting.value = true
  restartMessage.value = '正在重启服务，请稍候... | Restarting service, please wait...'
  try {
    await systemApi.restart()
    setTimeout(() => {
      window.location.reload()
    }, 6000)
  } catch {
    restartMessage.value = '重启请求已发送，服务将在数秒内重启 | Restart request sent, service will restart in seconds'
    setTimeout(() => {
      window.location.reload()
    }, 6000)
  }
}

// ── 配置导出状态 | Config export state ──
const exporting = ref(false)
const exportMessage = ref('')

async function handleExport() {
  exporting.value = true
  exportMessage.value = ''
  try {
    await systemApi.exportConfig()
    exportMessage.value = '配置导出成功 | Config exported successfully'
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    exportMessage.value = `导出失败 | Export failed: ${msg}`
  } finally {
    exporting.value = false
    setTimeout(() => { exportMessage.value = '' }, 5000)
  }
}
</script>

<template>
  <div class="settings-page">
    <!-- ── 页面头 | Page header ── -->
    <div class="page-header">
      <div class="page-title-row">
        <div class="page-title-icon">
          <Settings class="w-5 h-5" style="width:20px;height:20px;" />
        </div>
        <div>
          <h1 class="text-xl font-bold" style="color: var(--text-primary);">系统设置</h1>
          <p class="text-sm mt-0.5" style="color: var(--text-secondary);">EdgeOS 平台配置 · 数据库 · 服务管理</p>
        </div>
      </div>
      <div class="user-badge">
        <span class="user-avatar"><User class="w-4 h-4" style="width:16px;height:16px;" /></span>
        <div class="flex flex-col leading-tight">
          <span class="text-sm font-semibold" style="color: var(--text-primary);">{{ currentUser }}</span>
          <span class="text-[10px]" style="color: var(--text-secondary);">ROLE · {{ roleBadge }}</span>
        </div>
      </div>
    </div>

    <!-- ── 概览信息网格 | Overview info grid（全铺开）── -->
    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-5">
      <!-- 系统信息 -->
      <div class="settings-card">
        <div class="card-header">
          <span class="section-icon" style="background: rgba(14,165,233,0.12); color: var(--accent-primary);">
            <Info class="w-4 h-4" style="width:16px;height:16px;" />
          </span>
          <span class="text-sm font-semibold" style="color: var(--text-primary);">系统信息</span>
        </div>
        <div class="card-body divide-y" style="divide-color: var(--border-color);">
          <div v-for="item in systemInfo" :key="item.label" class="info-row">
            <span class="text-xs" style="color: var(--text-secondary);">{{ item.label }}</span>
            <span class="text-xs font-semibold mono" style="color: var(--text-primary);">{{ item.value }}</span>
          </div>
        </div>
      </div>

      <!-- MQTT 默认配置 -->
      <div class="settings-card">
        <div class="card-header">
          <span class="section-icon" style="background: rgba(34,197,94,0.12); color: #22c55e;">
            <Radio class="w-4 h-4" style="width:16px;height:16px;" />
          </span>
          <span class="text-sm font-semibold" style="color: var(--text-primary);">MQTT 默认配置</span>
          <button
            @click="router.push('/middleware')"
            class="ml-auto flex items-center gap-1 text-xs transition-colors hover:text-sky-400"
            style="color: var(--text-secondary);"
          >
            管理 <ExternalLink class="w-3 h-3" style="width:12px;height:12px;" />
          </button>
        </div>
        <div class="card-body divide-y" style="divide-color: var(--border-color);">
          <div v-for="item in mqttDefaults" :key="item.label" class="info-row">
            <span class="text-xs" style="color: var(--text-secondary);">{{ item.label }}</span>
            <span class="text-xs font-mono" style="color: var(--text-primary);">{{ item.value }}</span>
          </div>
        </div>
      </div>

      <!-- edgeCore 订阅主题 -->
      <div class="settings-card">
        <div class="card-header">
          <span class="section-icon" style="background: rgba(99,102,241,0.12); color: #818CF8;">
            <Server class="w-4 h-4" style="width:16px;height:16px;" />
          </span>
          <span class="text-sm font-semibold" style="color: var(--text-primary);">edgeCore 订阅主题</span>
        </div>
        <div class="card-body flex flex-wrap content-start gap-1.5">
          <span
            v-for="topic in edgeCoreTopics"
            :key="topic"
            class="text-[10px] font-mono px-2 py-1 rounded-md whitespace-nowrap"
            style="background: rgba(14,165,233,0.08); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.15);"
          >{{ topic }}</span>
        </div>
      </div>

      <!-- 快捷操作 -->
      <div class="settings-card">
        <div class="card-header">
          <span class="section-icon" style="background: var(--bg-tertiary); color: var(--text-secondary);">
            <LayoutGrid class="w-4 h-4" style="width:16px;height:16px;" />
          </span>
          <span class="text-sm font-semibold" style="color: var(--text-primary);">快捷操作</span>
        </div>
        <div class="card-body flex flex-col gap-2">
          <button
            @click="router.push('/middleware')"
            class="quick-action"
            style="background: rgba(14,165,233,0.1); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.2);"
          >
            <Radio class="w-4 h-4" style="width:16px;height:16px;" />
            <span class="flex flex-col items-start leading-tight">
              <span class="text-sm font-medium">配置中间件</span>
              <span class="text-[10px]" style="opacity:0.8;">消息总线连接管理</span>
            </span>
          </button>
          <button
            @click="router.push('/nodes')"
            class="quick-action"
            style="background: rgba(99,102,241,0.1); color: #818CF8; border: 1px solid rgba(99,102,241,0.2);"
          >
            <Server class="w-4 h-4" style="width:16px;height:16px;" />
            <span class="flex flex-col items-start leading-tight">
              <span class="text-sm font-medium">查看节点</span>
              <span class="text-[10px]" style="opacity:0.8;">边缘节点与设备概览</span>
            </span>
          </button>
        </div>
      </div>
    </div>

    <!-- ── 配置管理 | Config management（全宽）── -->
    <div class="settings-card">
      <div class="card-header">
        <span class="section-icon" style="background: rgba(14,165,233,0.12); color: var(--accent-primary);">
          <Download class="w-4 h-4" style="width:16px;height:16px;" />
        </span>
        <span class="text-sm font-semibold" style="color: var(--text-primary);">配置管理</span>
        <span class="ml-auto text-[10px] uppercase tracking-wider" style="color: var(--text-muted);">Config Backup & Export</span>
      </div>
      <div class="card-body space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-col gap-0.5">
            <span class="text-sm font-medium" style="color: var(--text-primary);">导出配置</span>
            <span class="text-xs" style="color: var(--text-secondary);">包含所有节点、设备和点位映射关系，建议定期备份</span>
          </div>
          <button
            @click="handleExport"
            :disabled="exporting"
            class="action-btn"
            style="background: rgba(14,165,233,0.1); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.2);"
          >
            <Download class="w-4 h-4" style="width:16px;height:16px;" />
            {{ exporting ? '导出中...' : '导出配置' }}
          </button>
        </div>
        <div v-if="exportMessage" class="result-banner" style="background: rgba(14,165,233,0.06); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.15);">
          {{ exportMessage }}
        </div>
      </div>
    </div>

    <!-- ── 数据管理 | Database management（全宽）── -->
    <div class="settings-card">
      <div class="card-header">
        <span class="section-icon" style="background: rgba(245,158,11,0.12); color: #F59E0B;">
          <Database class="w-4 h-4" style="width:16px;height:16px;" />
        </span>
        <span class="text-sm font-semibold" style="color: var(--text-primary);">数据管理</span>
        <span class="ml-auto text-[10px] uppercase tracking-wider" style="color: var(--text-muted);">Database Overview · Config · Runtime</span>
      </div>
      <div class="card-body">
        <DataManagement />
      </div>
    </div>

    <!-- ── 服务管理 | Service management（全宽）── -->
    <div class="settings-card">
      <div class="card-header">
        <span class="section-icon" style="background: rgba(239,68,68,0.12); color: #ef4444;">
          <RotateCcw class="w-4 h-4" style="width:16px;height:16px;" />
        </span>
        <span class="text-sm font-semibold" style="color: var(--text-primary);">服务管理</span>
        <span class="ml-auto text-[10px] uppercase tracking-wider" style="color: var(--text-muted);">Service Control</span>
      </div>
      <div class="card-body space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-col gap-0.5">
            <span class="text-sm font-medium" style="color: var(--text-primary);">服务重启</span>
            <span class="text-xs" style="color: var(--text-secondary);">重启 EdgeOS 服务进程，期间连接将短暂中断</span>
          </div>
          <button
            @click="showRestartConfirm = true"
            :disabled="restarting"
            class="action-btn"
            style="background: rgba(239,68,68,0.1); color: #ef4444; border: 1px solid rgba(239,68,68,0.2);"
          >
            <RotateCcw class="w-4 h-4" style="width:16px;height:16px;" />
            {{ restarting ? '重启中...' : '服务重启' }}
          </button>
        </div>
        <div v-if="restartMessage" class="result-banner" style="background: rgba(239,68,68,0.06); color: #ef4444; border: 1px solid rgba(239,68,68,0.2);">
          {{ restartMessage }}
        </div>
      </div>
    </div>

    <!-- ── 安全提示 | Security hint ── -->
    <div class="security-hint">
      <ShieldCheck class="w-4 h-4 flex-shrink-0" style="width:16px;height:16px;color:#22c55e;" />
      <span class="text-xs" style="color: var(--text-secondary);">
        配置存储于 <span class="mono">data/config.db</span>（bboltDB），无配置文件；修改配置后请重启服务生效。
      </span>
    </div>

    <!-- ── 重启确认弹窗 | Restart confirmation dialog ── -->
    <DangerDialog
      v-model:open="showRestartConfirm"
      title="确认重启服务"
      description="重启期间所有连接将中断，服务将在约 5 秒后自动恢复。确认要继续吗？"
      actionName="重启服务"
      variant="warning"
      @confirm="handleRestart"
    />
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
}

/* ── 页面头 ── */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.page-title-row {
  display: flex;
  align-items: center;
  gap: 14px;
}
.page-title-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(14,165,233,0.15), rgba(14,165,233,0.05));
  border: 1px solid rgba(14,165,233,0.25);
  color: var(--accent-primary);
}
.user-badge {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  border-radius: 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
}
.user-avatar {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: rgba(14,165,233,0.12);
  color: var(--accent-primary);
}

/* ── 通用卡片 ── */
.settings-card {
  border-radius: 0.75rem;
  overflow: hidden;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
}
.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
}
.section-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  flex-shrink: 0;
}
.card-body {
  padding: 14px 16px;
}

/* ── 概览网格项 ── */
.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 9px 2px;
}
.mono {
  font-family: 'JetBrains Mono', monaco, monospace;
}

/* ── 快捷操作 ── */
.quick-action {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  border-radius: 10px;
  transition: all 0.15s ease;
  cursor: pointer;
}
.quick-action:hover { filter: brightness(1.05); }

/* ── 操作按钮 ── */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.15s ease;
  cursor: pointer;
}
.action-btn:hover:not(:disabled) { filter: brightness(1.05); }
.action-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.result-banner {
  padding: 9px 12px;
  border-radius: 10px;
  font-size: 12px;
}

/* ── 安全提示 ── */
.security-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 12px;
  background: rgba(34,197,94,0.05);
  border: 1px solid rgba(34,197,94,0.12);
}
</style>
