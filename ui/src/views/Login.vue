<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 flex items-center justify-center p-4">
    <div 
      class="w-full max-w-md bg-white rounded-2xl shadow-xl p-8 transition-all duration-300"
      :class="{ 
        'translate-y-[-20px] opacity-0': isLoginSuccess,
        'border-l-4 border-red-500 shake': isShaking && ctxData.errorMessage
      }"
    >
      <!-- Header -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center gap-2 px-4 py-2 border-2 border-sky-500 rounded mb-4">
          <span class="text-sky-500 font-bold text-lg tracking-wider">EdgOS</span>
        </div>
        <h1 class="text-2xl font-bold text-slate-800 mb-1">边缘计算核心节点</h1>
        <p class="text-xs text-slate-500 font-mono tracking-wider">Edge Brain Open System</p>
      </div>

      <!-- Login Method Tabs -->
      <div class="flex gap-2 mb-6 bg-slate-100 p-1 rounded-lg">
        <button
          @click="ctxData.loginMethod = 'local'"
          class="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all duration-200"
          :class="ctxData.loginMethod === 'local' 
            ? 'bg-white text-slate-800 shadow-sm' 
            : 'text-slate-500 hover:text-slate-700'"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
          </svg>
          本地登录
        </button>
        <button
          @click="ctxData.loginMethod = 'ldap'"
          class="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all duration-200"
          :class="ctxData.loginMethod === 'ldap' 
            ? 'bg-white text-slate-800 shadow-sm' 
            : 'text-slate-500 hover:text-slate-700'"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
          </svg>
          LDAP 登录
        </button>
      </div>

      <!-- Login Form -->
      <form @submit.prevent="handleLogin" class="space-y-5">
        <!-- Username Input -->
        <div class="space-y-2">
          <label class="block text-xs font-bold text-slate-700">用户标识 / Username</label>
          <div class="relative">
            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
              <svg v-if="ctxData.loginMethod === 'local'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
              </svg>
              <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
              </svg>
            </span>
            <input
              v-model="ctxData.loginForm.userName"
              type="text"
              :placeholder="ctxData.loginMethod === 'ldap' ? 'LDAP 账号 / 邮箱' : '请输入用户名'"
              class="w-full pl-10 pr-4 py-3 border border-slate-200 rounded-lg focus:outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-500/10 transition-all duration-200"
            />
          </div>
        </div>

        <!-- Password Input -->
        <div class="space-y-2">
          <label class="block text-xs font-bold text-slate-700">访问密钥 / Password</label>
          <div class="relative">
            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
              </svg>
            </span>
            <input
              v-model="ctxData.loginForm.password"
              type="password"
              :placeholder="ctxData.loginMethod === 'ldap' ? 'LDAP 域密码' : '请输入密码'"
              class="w-full pl-10 pr-4 py-3 border border-slate-200 rounded-lg focus:outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-500/10 transition-all duration-200"
              @keyup.enter="handleLogin"
            />
          </div>
        </div>

        <!-- Remember & Forgot -->
        <div class="flex items-center justify-between">
          <label class="flex items-center gap-2 text-sm text-slate-600 cursor-pointer">
            <input
              v-model="ctxData.rememberMe"
              type="checkbox"
              class="w-4 h-4 text-sky-500 border-slate-300 rounded focus:ring-sky-500"
            />
            记住访问权限
          </label>
          <a
            v-if="ctxData.loginMethod === 'local'"
            @click="handleForgotPassword"
            class="text-sm text-slate-400 hover:text-sky-500 transition-colors cursor-pointer"
          >
            忘记密码?
          </a>
        </div>

        <!-- Auth Mode Indicator -->
        <div class="flex items-center justify-center gap-2 text-xs text-slate-500 font-mono">
          <span 
            class="w-2 h-2 rounded-full animate-pulse"
            :class="ctxData.loginMethod === 'ldap' ? 'bg-purple-500' : 'bg-green-500'"
          ></span>
          <span>AUTH_MODE: {{ ctxData.loginMethod === 'ldap' ? 'LDAP_DOMAIN' : 'LOCAL_DATABASE' }}</span>
        </div>

        <!-- Error Message -->
        <div v-if="ctxData.errorMessage" class="flex items-center gap-2 px-4 py-3 bg-red-50 border border-red-200 border-l-4 border-l-red-500 rounded-lg text-red-600 text-sm">
          <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span>{{ ctxData.errorMessage }}</span>
        </div>

        <!-- Submit Button -->
        <button
          type="submit"
          :disabled="ctxData.loading || isLoginSuccess"
          class="w-full py-3 px-4 bg-gradient-to-r from-sky-500 to-sky-400 text-white font-medium rounded-lg shadow-lg hover:shadow-xl hover:from-sky-600 hover:to-sky-500 disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none transition-all duration-200 flex items-center justify-center gap-2"
        >
          <svg v-if="isLoginSuccess" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
          </svg>
          <svg v-else-if="ctxData.loading" class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6"/>
          </svg>
          <span v-if="isLoginSuccess">登录成功</span>
          <span v-else-if="ctxData.loading">登录中...</span>
          <span v-else>{{ ctxData.loginMethod === 'ldap' ? '域验证登录' : '立即登录' }}</span>
        </button>
      </form>

      <!-- Footer -->
      <div class="mt-6 text-center">
        <p class="text-xs text-slate-400 font-mono">© {{ new Date().getFullYear() }} {{ ctxData.configInfo.name || '系统' }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { authApi } from '@/api'
import router from '@/router'
import { userStore } from '@/stores/user'
import { configStore } from '@/stores/app'
import { showMessage } from '@/composables/useGlobalState'

// 使用原生 Web Crypto API 进行 SHA256 加密
const sha256Encrypt = async (text) => {
  const encoder = new TextEncoder()
  const data = encoder.encode(text)
  const hash = await crypto.subtle.digest('SHA-256', data)
  const hashArray = Array.from(new Uint8Array(hash))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

const config = configStore()
const users = userStore()

const loginFormRef = ref(null)
const isShaking = ref(false)
const isLoginSuccess = ref(false)

const ctxData = reactive({
  loginForm: {
    userName: '',
    password: '',
  },
  loginMethod: 'local',
  loading: false,
  rememberMe: false,
  configInfo: config.configInfo || {},
  nonce: '',
  errorMessage: ''
})

onMounted(() => {
  const logout = localStorage.getItem('logout')
  if (logout && logout !== '') {
    try {
      const lo = JSON.parse(logout)
      showMessage(lo.message || '您已成功退出登录', lo.type || 'info')
    } catch (error) {
      console.error('解析登出信息失败:', error)
    }
    localStorage.setItem('logout', '')
  }

  loadRememberedAccount()
  getSystemInfo()
  getNonce()
})

const loadRememberedAccount = () => {
  try {
    const saved = localStorage.getItem('rememberedAccount')
    if (saved) {
      const account = JSON.parse(saved)
      ctxData.loginForm.userName = account.userName || ''
      ctxData.rememberMe = true
    }
  } catch (e) {
    console.error('加载保存的账号失败:', e)
  }
}

const getSystemInfo = async () => {
  try {
    const res = await authApi.getSystemInfo()
    if (res.code === '0' && res.data) {
      // 只更新版本号等信息，不更新系统标题
      const newConfigInfo = {
        ...ctxData.configInfo,
        softVer: res.data.softVer
      }
      ctxData.configInfo = newConfigInfo
      config.setConfigInfo(newConfigInfo)
    }
  } catch (error) {
    console.error('获取系统信息失败:', error)
  }
}

const getNonce = async () => {
  try {
    const nonce = await authApi.getNonce()
    ctxData.nonce = nonce
  } catch (error) {
    console.error('获取nonce异常:', error)
    ctxData.nonce = Date.now().toString(36) + Math.random().toString(36).substr(2)
  }
}

const handleLogin = async () => {
  if (!ctxData.loginForm.userName) {
    ctxData.errorMessage = '请输入用户名'
    triggerShake()
    return
  }
  if (!ctxData.loginForm.password) {
    ctxData.errorMessage = '请输入密码'
    triggerShake()
    return
  }
  if (ctxData.loginForm.password.length < 8) {
    ctxData.errorMessage = '密码长度至少8位'
    triggerShake()
    return
  }

  ctxData.loading = true
  ctxData.errorMessage = ''

  try {
    if (!ctxData.nonce) {
      await getNonce()
    }

    let passwordToSend = ''
    if (ctxData.loginMethod === 'ldap') {
      passwordToSend = ctxData.loginForm.password
    } else {
      passwordToSend = await sha256Encrypt(ctxData.loginForm.password + ctxData.nonce)
    }

    const loginData = {
      loginFlag: true,
      loginType: ctxData.loginMethod,
      data: {
        username: ctxData.loginForm.userName,
        password: passwordToSend,
        nonce: ctxData.nonce,
      },
      token: '',
    }

    const res = await authApi.login(loginData)

    if (res.code === '0') {
      await handleLoginSuccess(res)
    } else {
      handleLoginFailure(res)
      triggerShake()
      ctxData.loading = false
    }
  } catch (error) {
    handleLoginError(error)
    triggerShake()
    ctxData.loading = false
  }
}

const triggerShake = () => {
  isShaking.value = true
  setTimeout(() => {
    isShaking.value = false
  }, 500)
}

const handleLoginSuccess = async (res) => {
  try {
    ctxData.errorMessage = ''
    isLoginSuccess.value = true

    const processedPermissions = processPermissions(res.data.permissions)

    // 保存登录信息到 store (store 内部会自动保存 token 到 localStorage)
    users.setLoginInfo(
      { userName: res.data.username },
      processedPermissions,
      res.data.token
    )

    // 记住账号
    if (ctxData.rememberMe) {
      localStorage.setItem('rememberedAccount', JSON.stringify({
        userName: ctxData.loginForm.userName,
        timestamp: Date.now()
      }))
    } else {
      localStorage.removeItem('rememberedAccount')
    }

    showMessage('登录成功')

    ctxData.loading = false
    await new Promise(resolve => setTimeout(resolve, 1000))
    await router.push('/')

  } catch (error) {
    console.error('处理登录成功数据失败:', error)
    ctxData.errorMessage = '处理用户数据失败,请稍后重试'
    ctxData.loading = false
  }
}

const processPermissions = (permissions) => {
  const perms = Array.isArray(permissions) ? [...permissions] : []

  const ensureTerminalGroup = (list) => {
    const edge = list.find(p =>
      p && (p.path === '/ruleEngine' || p.meta?.title === '边缘计算')
    )

    if (edge) {
      edge.children = edge.children || []
      const hasTerminalGroup = edge.children.some(c =>
        c && (c.path === '/terminalGroup' || c.meta?.title === '末端群控')
      )

      if (!hasTerminalGroup) {
        const terminalGroup = {
          path: '/terminalGroup',
          name: 'TerminalGroup',
          meta: { title: '末端群控', icon: 'terminal' }
        }

        const scriptIndex = edge.children.findIndex(c =>
          c && c.meta?.title === '规则脚本'
        )

        if (scriptIndex >= 0) {
          edge.children.splice(scriptIndex + 1, 0, terminalGroup)
        } else {
          edge.children.push(terminalGroup)
        }
      }
    } else {
      list.push({
        name: 'RuleEngine',
        path: '/ruleEngine',
        meta: { title: '边缘计算', icon: 'ruleEngine' },
        children: [{
          path: '/terminalGroup',
          name: 'TerminalGroup',
          meta: { title: '末端群控', icon: 'terminal' }
        }]
      })
    }

    return list
  }

  return ensureTerminalGroup(perms)
}

const handleLoginFailure = (res) => {
  ctxData.errorMessage = res.message || '登录失败，请检查用户名和密码'
  getNonce()
}

const handleLoginError = (error) => {
  console.error('登录错误:', error)

  if (error.code === 'ECONNABORTED' || error.code === 'ERR_NETWORK') {
    ctxData.errorMessage = '网络连接失败，请检查网络后重试'
  } else {
    ctxData.errorMessage = '登录异常，请稍后重试'
  }

  getNonce()
}

const handleForgotPassword = () => {
  showMessage('请联系系统管理员重置密码', 'info')
}
</script>

<style scoped>
.shake {
  animation: shake 0.3s ease-out;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}
</style>

<style scoped>
/* ===== 容器：全屏数据背景 ===== */
.login-container {
  position: fixed;
  inset: 0;
  background: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'JetBrains Mono', monaco, monospace, sans-serif;
}

/* 登录 UI 层 */
.login-scene {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.login-panel {
  width: 580px;
  padding: 32px 60px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  box-shadow: 0 8px 20px -5px rgba(0, 0, 0, 0.05), 0 6px 8px -6px rgba(0, 0, 0, 0.05);
  transition: box-shadow 0.15s ease, border-color 0.15s ease;
  position: relative;
  overflow: hidden;
}

.login-panel::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  border: 2px solid transparent;
  border-radius: 12px;
  pointer-events: none;
  z-index: 1;
}

.login-panel::after {
  content: '';
  position: absolute;
  top: -2px;
  left: -2px;
  right: -2px;
  bottom: -2px;
  border: 2px solid #0ea5e9;
  border-radius: 12px;
  pointer-events: none;
  z-index: 0;
  clip-path: var(--clip-path, polygon(0 100%, 100% 100%, 100% 100%, 0 100%));
  transition: clip-path 0.1s linear, border-color 0.3s ease;
}

.login-panel.countdown-10::after {
  border-color: #ef4444;
  box-shadow: 0 0 10px rgba(239, 68, 68, 0.5);
}

.login-panel:hover {
  box-shadow: 0 15px 30px -5px rgba(0, 0, 0, 0.1), 0 10px 15px -6px rgba(0, 0, 0, 0.1);
  border-color: #cbd5e1;
}

.panel-topbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 8px;
}

.panel-title {
  text-align: center;
  margin-bottom: 24px;
}

.title-main {
  font-size: 20px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: 0.5px;
  margin: 0;
}

.title-sub {
  font-size: 12px;
  color: #64748b;
  letter-spacing: 1.4px;
  font-family: monaco, monospace;
  margin-top: 4px;
}

.panel-header-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.auth-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 20px;
}

/* 其他样式参考前述代码，保持 Arco 组件自定义效果 */
.field { margin-bottom: 20px; }
.label { font-size: 12px; font-weight: 700; color: #475569; margin-bottom: 8px; }

.options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 14px 0 10px;
}

.mode-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: rgba(34, 197, 94, 0.7);
  box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.12);
}

.mode-indicator.is-ldap {
  background: rgba(139, 92, 246, 0.9);
  box-shadow: 0 0 0 4px rgba(139, 92, 246, 0.12);
}

.panel-error {
  margin-top: 12px;
  margin-bottom: 0;
}

.compact-hint {
  margin: 0 0 10px;
  padding: 0;
  border: none;
  background: transparent;
  font-size: 11px;
}

.compact-terminal {
  justify-content: center;
  gap: 8px;
  margin-bottom: 10px;
}

.panel-copyright {
  margin-top: 14px;
  text-align: center;
}

/* 保持 Logo 工业感样式 */
.logo-icon {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 2px solid #0ea5e9;
  border-radius: 2px;
  padding: 6px 12px;
  margin-right: 16px;
}
.logo-icon span { font-weight: 800; color: #0ea5e9; font-size: 16px; }
.logo-icon small { color: #64748b; font-size: 10px; margin-left: 2px; }

.version-tag {
  font-size: 10px;
  font-family: monaco, monospace;
  color: #94a3b8;
  letter-spacing: 1px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 2px;
  padding: 1px 6px;
}

.top-progress {
  width: 88px;
}

.industrial-radio {
  width: 100%;
  display: flex;
}

:deep(.industrial-radio .arco-radio-button) {
  flex: 1;
  justify-content: center;
  border-radius: 8px !important;
  font-weight: 500;
  font-size: 12px;
  white-space: nowrap;
}

/* ===== 表单 ===== */
.custom-form {
  display: flex;
  flex-direction: column;
  gap: 0;
}

:deep(.arco-form-item) {
  margin-bottom: 14px;
}

:deep(.arco-input-wrapper),
:deep(.arco-input-password) {
  border-radius: 8px !important;
  box-shadow: none !important;
  border-color: #cbd5e1 !important;
}

:deep(.arco-input-wrapper:hover),
:deep(.arco-input-password:hover) {
  border-color: #0ea5e9 !important;
}

:deep(.arco-input-wrapper.arco-input-focus),
:deep(.arco-input-password.arco-input-focus) {
  border-color: #0ea5e9 !important;
  box-shadow: 0 0 0 1px rgba(14, 165, 233, 0.15) !important;
}

/* ===== LDAP 提示 ===== */
.ldap-hint {
  font-size: 11px;
  color: #64748b;
  margin-bottom: 14px;
  padding: 8px 10px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-left: 3px solid #0ea5e9;
  border-radius: 2px;
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: monaco, monospace;
}

.remember-check {
  font-size: 13px;
  color: #64748b;
}

:deep(.remember-check .arco-checkbox-label) {
  color: #64748b;
  font-size: 13px;
}

.forgot-link {
  font-size: 13px;
  color: #94a3b8 !important;
}

.forgot-link:hover {
  color: #0ea5e9 !important;
}

/* ===== 错误提示 ===== */
.error-message {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.04);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-left: 3px solid #ef4444;
  border-radius: 2px;
  color: #ef4444;
  font-size: 13px;
  margin-bottom: 14px;
}

/* ===== 提交按钮 ===== */
.login-submit-btn { height: 50px !important; margin-top: 10px; }

:deep(.login-submit-btn.arco-btn-primary) {
  border-radius: 8px !important;
  background: linear-gradient(135deg, #0ea5e9 0%, #38bdf8 100%) !important;
  border: none !important;
  box-shadow: 0 4px 16px rgba(14, 165, 233, 0.3) !important;
  transition: all 0.2s ease !important;
}

:deep(.login-submit-btn.arco-btn-primary:hover) {
  transform: translateY(-1px) !important;
  box-shadow: 0 6px 20px rgba(14, 165, 233, 0.4) !important;
  background: linear-gradient(135deg, #0284c7 0%, #0ea5e9 100%) !important;
}

:deep(.login-submit-btn.arco-btn-primary:active) {
  transform: translateY(0) !important;
  box-shadow: 0 4px 12px rgba(14, 165, 233, 0.3) !important;
}

.terminal-decorator {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-dot { width: 6px; height: 6px; background: #22c55e; animation: blink 1.5s infinite; }

@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

.status-dot.is-ldap {
  background: #0ea5e9;
  box-shadow: 0 0 6px rgba(14, 165, 233, 0.6);
}

.terminal-code {
  font-size: 10px;
  font-family: monaco, monospace;
  color: #94a3b8;
  letter-spacing: 0.5px;
}

.copyright-text {
  font-size: 11px;
  color: #94a3b8;
  font-family: monaco, monospace;
}

/* ===== 动画 ===== */
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}

.shake-animation {
  animation: shake 0.3s ease-out both;
}

.login-card-exit {
  transform: scale(0.95) translateY(-20px);
  opacity: 0;
  transition: all 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  pointer-events: none;
}

/* ===== 响应式 ===== */
@media (max-width: 1199px) {
  .login-panel {
    width: min(580px, 90vw);
  }
}

@media (max-width: 767px) {
  .login-panel {
    width: calc(100vw - 24px);
    padding: 24px;
    border-radius: 12px;
  }

  .panel-topbar,
  .options,
  .auth-row {
    flex-direction: column;
    align-items: stretch;
  }

  .panel-header-side {
    align-items: flex-start;
  }

  .login-submit-btn {
    width: 100%;
  }

  .top-progress {
    width: 82px;
  }
}
</style>
