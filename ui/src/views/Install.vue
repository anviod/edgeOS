<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { Eye, EyeOff } from 'lucide-vue-next'
import { installApi } from '@/api/install'
import type { InstallPayload } from '@/api/install'

type Step = 'node' | 'account' | 'installing' | 'completed'

const current = ref<Step>('node')
const loading = ref(false)
const errorMsg = ref('')
const generalErrors = ref<string[]>([])
const statusText = ref('')
const redirectCountdown = ref(5)
const showPassword = ref(false)
const showConfirmPassword = ref(false)
let redirectTimer: ReturnType<typeof setInterval> | null = null

const form = reactive({
  node_type: 'primary' as 'primary' | 'secondary' | 'collector',
  node_id: 'node-001',
  primary_node_id: 'node-primary',
  listen: ':8000',
  username: 'admin',
  password: '',
  confirm_password: '',
  ean_enabled: false,
  ean_planner_id: 'edgeos-planner',
})

const nodeTypeOptions = [
  { value: 'primary', label: '主节点', desc: 'Primary Queen · 全局调度与决策' },
  { value: 'secondary', label: '备用节点', desc: 'Secondary Queen · 故障自动切换' },
  { value: 'collector', label: '采集节点', desc: 'Edge Collector · 设备接入与协议转换' },
]

const showPrimaryNode = computed(() => form.node_type === 'secondary')
const hasSpecialChar = computed(() => /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?~`]/.test(form.password))
const passwordOk = computed(() => form.password.length >= 8)
const passwordMixOk = computed(() => {
  const upper = /[A-Z]/.test(form.password)
  const lower = /[a-z]/.test(form.password)
  const num = /[0-9]/.test(form.password)
  return [upper, lower, num, hasSpecialChar.value].filter(Boolean).length >= 3
})

function validateStep(step: Step): boolean {
  generalErrors.value = []
  if (step === 'node') {
    if (!form.node_id.trim()) generalErrors.value.push('节点 ID 不能为空')
    if (showPrimaryNode.value && !form.primary_node_id.trim()) generalErrors.value.push('备用节点必须配置主节点 ID')
    return generalErrors.value.length === 0
  }
  if (step === 'account') {
    if (!form.username.trim()) generalErrors.value.push('管理员用户名不能为空')
    if (!passwordOk.value) generalErrors.value.push('密码长度至少 8 位')
    if (!passwordMixOk.value) generalErrors.value.push('密码需包含大写字母、小写字母、数字、特殊符号中的至少三种')
    if (form.password !== form.confirm_password) generalErrors.value.push('两次输入的密码不一致')
    return generalErrors.value.length === 0
  }
  return true
}

function nextStep() {
  const target: Step = current.value === 'node' ? 'account' : 'node'
  if (validateStep(current.value)) {
    current.value = target
  }
}

function prevStep() {
  current.value = 'node'
}

async function handleStart() {
  if (!validateStep('account')) return
  loading.value = true
  errorMsg.value = ''
  try {
    const payload: InstallPayload = {
      node: {
        node_id: form.node_id.trim(),
        node_type: form.node_type,
        primary_node_id: showPrimaryNode.value ? form.primary_node_id.trim() : undefined,
        listen: form.listen.trim() || ':8000',
      },
      user: {
        username: form.username.trim(),
        password: form.password,
        role: 'admin',
      },
      ean: {
        enabled: form.ean_enabled,
        planner_id: form.ean_planner_id.trim() || undefined,
      },
      middlewares: [],
    }
    await installApi.start(payload)
    current.value = 'installing'
    statusText.value = '配置已写入数据库，系统正在重启…'
    startCountdown()
  } catch (err) {
    errorMsg.value = (err as Error)?.message || '初始化失败'
  } finally {
    loading.value = false
  }
}

function startCountdown() {
  redirectCountdown.value = 6
  if (redirectTimer) clearInterval(redirectTimer)
  redirectTimer = setInterval(() => {
    redirectCountdown.value--
    if (redirectCountdown.value <= 0) {
      if (redirectTimer) clearInterval(redirectTimer)
      current.value = 'completed'
      setTimeout(() => {
        window.location.href = '/login'
      }, 800)
    }
  }, 1000)
}

function stepClass(step: Step): string {
  const order: Step[] = ['node', 'account', 'installing']
  const idx = order.indexOf(step)
  const cur = order.indexOf(current.value)
  return idx < cur ? 'done' : idx === cur ? 'active' : ''
}

onMounted(async () => {
  try {
    const st = await installApi.status()
    // 已有数据库配置：不得再次进入安装引导
    if (st.is_installed) {
      window.location.href = '/login'
      return
    }
  } catch {
    // 状态检查失败时保守处理：仍展示安装引导
  }
})

onUnmounted(() => {
  if (redirectTimer) clearInterval(redirectTimer)
})
</script>

<template>
  <div class="install-container">
    <div class="install-scene">
      <div class="install-panel">
        <!-- Top bar -->
        <div class="panel-topbar">
          <div class="logo-box">
            <div class="logo-icon"><span>EdgeOS</span></div>
            <div class="panel-header-side">
              <span class="version-tag">INSTALL WIZARD</span>
            </div>
          </div>
        </div>

        <!-- Title -->
        <div class="panel-title">
          <div class="title-main">系统初始化安装</div>
          <div class="title-sub">Edge Brain Open System · First-time Setup</div>
        </div>

        <!-- Step indicator -->
        <div class="step-indicator">
          <div class="step" :class="stepClass('node')">
            <span class="step-number">1</span>
            <span class="step-label">节点配置</span>
          </div>
          <div class="step-divider" :class="{ done: current !== 'node' }"></div>
          <div class="step" :class="stepClass('account')">
            <span class="step-number">2</span>
            <span class="step-label">管理员账户</span>
          </div>
          <div class="step-divider" :class="{ done: current === 'installing' || current === 'completed' }"></div>
          <div class="step" :class="{ active: current === 'installing' || current === 'completed' }">
            <span class="step-number">3</span>
            <span class="step-label">初始化</span>
          </div>
        </div>

        <!-- Step: node config -->
        <div v-if="current === 'node'" class="install-body">
          <div class="field">
            <div class="label">节点类型 / Node Type</div>
            <div class="node-type-grid">
              <button
                v-for="opt in nodeTypeOptions"
                :key="opt.value"
                type="button"
                class="node-type-card"
                :class="{ selected: form.node_type === opt.value }"
                @click="form.node_type = opt.value"
              >
                <div class="node-type-title">{{ opt.label }}</div>
                <div class="node-type-desc">{{ opt.desc }}</div>
              </button>
            </div>
          </div>

          <div class="field">
            <div class="label">节点 ID / Node ID</div>
            <input v-model="form.node_id" type="text" class="install-input" placeholder="如 node-001" />
          </div>

          <div v-if="showPrimaryNode" class="field">
            <div class="label">主节点 ID / Primary Node ID</div>
            <input v-model="form.primary_node_id" type="text" class="install-input" placeholder="如 node-primary" />
          </div>

          <div class="field">
            <div class="label">Web 服务端口 / Listen Port</div>
            <input v-model="form.listen" type="text" class="install-input" placeholder="如 :8000" />
            <div class="form-hint">留空使用默认 :8000</div>
          </div>

          <div v-if="generalErrors.length > 0" class="error-list">
            <div v-for="(e, i) in generalErrors" :key="i" class="error-message">{{ e }}</div>
          </div>

          <button type="button" class="install-submit-btn" @click="nextStep">
            下一步：管理员账户
          </button>
        </div>

        <!-- Step: account config -->
        <div v-else-if="current === 'account'" class="install-body">
          <div class="field">
            <div class="label">管理员用户名 / Username</div>
            <input v-model="form.username" type="text" class="install-input" placeholder="默认 admin" />
          </div>

          <div class="field">
            <div class="label">管理员密码 / Password</div>
            <div class="password-wrap">
              <input
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                class="install-input"
                placeholder="至少 8 位，包含三种字符组合"
              />
              <button
                type="button"
                class="password-eye"
                :title="showPassword ? '隐藏密码' : '显示密码'"
                @click="showPassword = !showPassword"
                @mousedown.prevent
              >
                <EyeOff v-if="showPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
            <div class="password-rules">
              <span class="rule" :class="{ pass: passwordOk }">≥8位</span>
              <span class="rule" :class="{ pass: /[A-Z]/.test(form.password) }">大写</span>
              <span class="rule" :class="{ pass: /[a-z]/.test(form.password) }">小写</span>
              <span class="rule" :class="{ pass: /[0-9]/.test(form.password) }">数字</span>
              <span class="rule" :class="{ pass: hasSpecialChar }">符号</span>
              <span class="rule" :class="{ pass: passwordMixOk }">三选</span>
            </div>
          </div>

          <div class="field">
            <div class="label">确认密码 / Confirm Password</div>
            <div class="password-wrap">
              <input
                v-model="form.confirm_password"
                :type="showConfirmPassword ? 'text' : 'password'"
                class="install-input"
                placeholder="再次输入密码"
              />
              <button
                type="button"
                class="password-eye"
                :title="showConfirmPassword ? '隐藏密码' : '显示密码'"
                @click="showConfirmPassword = !showConfirmPassword"
                @mousedown.prevent
              >
                <EyeOff v-if="showConfirmPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <div class="field">
            <div class="label ean-toggle-label">
              <span>EAN 2.0 协调层</span>
              <label class="switch">
                <input v-model="form.ean_enabled" type="checkbox" />
                <span class="slider"></span>
              </label>
            </div>
            <div v-if="form.ean_enabled" class="field">
              <input v-model="form.ean_planner_id" type="text" class="install-input" placeholder="Planner ID，默认 edgeos-planner" />
            </div>
            <div class="form-hint">EAN 启用后需在消息总线中配置 MQTT/NATS broker</div>
          </div>

          <div v-if="generalErrors.length > 0" class="error-list">
            <div v-for="(e, i) in generalErrors" :key="i" class="error-message">{{ e }}</div>
          </div>
          <div v-if="errorMsg" class="error-message">{{ errorMsg }}</div>

          <div class="form-actions">
            <button type="button" class="install-submit-btn ghost" @click="prevStep">上一步</button>
            <button type="button" class="install-submit-btn" :disabled="loading" @click="handleStart">
              {{ loading ? '正在初始化…' : '开始初始化安装' }}
            </button>
          </div>
        </div>

        <!-- Step: installing -->
        <div v-else-if="current === 'installing'" class="install-body installing">
          <div class="installing-icon">
            <svg class="spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2a10 10 0 0 1 10 10" />
            </svg>
          </div>
          <div class="installing-title">正在初始化系统</div>
          <div class="installing-sub">{{ statusText }}</div>
          <div class="installing-note">
            配置已写入 data/config.db，服务将自动重启（systemd 会自动拉起）。<br />
            如未自动重启，请手动重启服务后使用新账户登录。
          </div>
          <div class="countdown">自动跳转登录页：{{ redirectCountdown }}s</div>
        </div>

        <!-- Step: completed -->
        <div v-else class="install-body installing">
          <div class="complete-icon">✓</div>
          <div class="installing-title">初始化完成</div>
          <div class="installing-sub">系统将跳转到登录页面</div>
        </div>

        <!-- Footer -->
        <div class="copyright-text">© {{ new Date().getFullYear() }} EdgeOS · 配置存储于 bboltDB (data/config.db)</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ===== 容器：全屏数据背景 ===== */
.install-container {
  position: fixed;
  inset: 0;
  background: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'JetBrains Mono', monaco, monospace, sans-serif;
  overflow-y: auto;
}

.install-scene {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  width: 100%;
  min-height: 100%;
  padding: 40px 16px;
}

.install-panel {
  width: 620px;
  max-width: 100%;
  padding: 32px 48px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 0;
  box-shadow: 0 8px 20px -5px rgba(0, 0, 0, 0.05), 0 6px 8px -6px rgba(0, 0, 0, 0.05);
}

.panel-topbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.logo-box { display: flex; align-items: center; gap: 12px; }
.logo-icon {
  display: inline-flex;
  align-items: center;
  border: 2px solid #0ea5e9;
  border-radius: 0;
  padding: 6px 12px;
}
.logo-icon span { font-weight: 800; color: #0ea5e9; font-size: 16px; }

.version-tag {
  font-size: 10px;
  font-family: monaco, monospace;
  color: #94a3b8;
  letter-spacing: 1px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 0;
  padding: 1px 6px;
}

.panel-title { text-align: center; margin: 12px 0 24px; }
.title-main { font-size: 20px; font-weight: 600; color: #0f172a; letter-spacing: 0.5px; }
.title-sub { font-size: 12px; color: #64748b; letter-spacing: 1.4px; margin-top: 4px; }

/* ===== Step indicator ===== */
.step-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 28px;
}
.step { display: flex; align-items: center; gap: 8px; color: #94a3b8; }
.step-number {
  width: 24px; height: 24px;
  display: flex; align-items: center; justify-content: center;
  border: 1px solid #cbd5e1;
  border-radius: 0;
  font-size: 12px;
  font-weight: 700;
  color: #94a3b8;
  background: #f8fafc;
}
.step.active .step-number { background: #0ea5e9; border-color: #0ea5e9; color: #fff; }
.step.done .step-number { background: #0f172a; border-color: #0f172a; color: #fff; }
.step-label { font-size: 12px; font-weight: 600; color: #64748b; }
.step.active .step-label { color: #0ea5e9; }
.step-divider { flex: 0 0 40px; height: 1px; background: #e2e8f0; }
.step-divider.done { background: #0ea5e9; }

/* ===== Form ===== */
.install-body { display: flex; flex-direction: column; gap: 16px; }
.field { display: flex; flex-direction: column; gap: 8px; }
.label { font-size: 12px; font-weight: 700; color: #475569; }
.ean-toggle-label { display: flex; align-items: center; justify-content: space-between; }
.form-hint { font-size: 11px; color: #94a3b8; }

.install-input {
  width: 100%;
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  border-radius: 0;
  color: #0f172a;
  padding: 10px 12px;
  font-size: 13px;
  box-shadow: none;
  outline: none;
  transition: border-color 0.15s ease;
}
.install-input::placeholder { color: #94a3b8; }
.install-input:focus { border-color: #0ea5e9; }

.password-wrap {
  position: relative;
  display: flex;
  align-items: center;
}
.password-wrap .install-input {
  padding-right: 42px;
}
.password-eye {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: none;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  transition: color 0.15s ease;
}
.password-eye:hover { color: #0ea5e9; }

.node-type-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.node-type-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  border-radius: 0;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background-color 0.15s ease;
  color: #0f172a;
}
.node-type-card:hover { border-color: #0ea5e9; }
.node-type-card.selected { border-color: #0ea5e9; background: rgba(14, 165, 233, 0.06); }
.node-type-title { font-size: 13px; font-weight: 700; }
.node-type-desc { font-size: 10px; color: #64748b; line-height: 1.4; }

.password-rules { display: flex; gap: 8px; flex-wrap: wrap; }
.rule { font-size: 10px; color: #94a3b8; border: 1px solid #e2e8f0; padding: 2px 6px; }
.rule.pass { color: #0ea5e9; border-color: #0ea5e9; }

/* ===== Switch ===== */
.switch { position: relative; display: inline-block; width: 40px; height: 22px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider {
  position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0;
  background: #e2e8f0; border-radius: 0; transition: 0.2s;
}
.slider:before {
  position: absolute; content: ''; height: 16px; width: 16px; left: 3px; bottom: 3px;
  background: #fff; border-radius: 0; transition: 0.2s;
}
.switch input:checked + .slider { background: #0ea5e9; }
.switch input:checked + .slider:before { transform: translateX(18px); }

/* ===== Errors ===== */
.error-list { display: flex; flex-direction: column; gap: 6px; }
.error-message {
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.04);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-left: 3px solid #ef4444;
  color: #ef4444;
  font-size: 12px;
}

/* ===== Actions ===== */
.form-actions { display: flex; gap: 12px; }
.install-submit-btn {
  flex: 1;
  height: 48px;
  margin-top: 6px;
  background: #0ea5e9;
  color: #fff;
  border: none;
  border-radius: 0;
  box-shadow: none;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s ease;
  font-family: inherit;
}
.install-submit-btn:hover { background: #0284c7; }
.install-submit-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.install-submit-btn.ghost { flex: 0 0 120px; background: #f8fafc; color: #475569; border: 1px solid #cbd5e1; }
.install-submit-btn.ghost:hover { background: #f1f5f9; }

/* ===== Installing / completed ===== */
.install-body.installing { align-items: center; text-align: center; padding: 16px 0; }
.installing-icon { width: 48px; height: 48px; color: #0ea5e9; }
.spinner { width: 48px; height: 48px; animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.installing-title { font-size: 18px; font-weight: 700; color: #0f172a; }
.installing-sub { font-size: 13px; color: #64748b; }
.installing-note { font-size: 12px; color: #94a3b8; line-height: 1.8; max-width: 420px; }
.countdown { font-size: 12px; color: #0ea5e9; font-weight: 600; }
.complete-icon {
  width: 56px; height: 56px;
  display: flex; align-items: center; justify-content: center;
  background: #0ea5e9; color: #fff; font-size: 28px; font-weight: 700;
}

.copyright-text {
  margin-top: 20px;
  text-align: center;
  font-size: 11px;
  color: #94a3b8;
  font-family: monaco, monospace;
}

@media (max-width: 767px) {
  .install-panel { padding: 24px; }
  .node-type-grid { grid-template-columns: 1fr; }
}
</style>
