<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { Zap, Eye, EyeOff, LogIn } from 'lucide-vue-next'
import { authApi } from '@/api/index'
import { userStore } from '@/stores/user'

const router = useRouter()
const showPassword = ref(false)
const loading = ref(false)
const errorMsg = ref('')

const form = reactive({ username: 'admin', password: 'admin' })

async function handleLogin() {
  if (!form.username || !form.password) {
    errorMsg.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  errorMsg.value = ''
  const result = await authApi.login(form.username, form.password).catch(e => {
    errorMsg.value = e?.message || '登录失败，请检查用户名和密码'
    return null
  })
  loading.value = false
  if (result) {
    const store = userStore()
    store.setLoginInfo({ username: result.username }, result.permissions, result.token)
    router.push('/dashboard')
  }
}
</script>

<template>
  <div class="login-container">
    <div class="login-scene">
      <div class="login-panel">
        <!-- Top bar -->
        <div class="panel-topbar">
          <div class="logo-box">
            <div class="logo-icon">
              <span>EdgeOS</span>
            </div>
          </div>
          <div class="panel-header-side">
            <span class="version-tag">VER 0.0.1</span>
          </div>
        </div>

        <!-- Title -->
        <div class="panel-title">
          <div class="title-main">边缘计算主控网关</div>
          <div class="title-sub">Edge Brain Open System</div>
        </div>

        <!-- Form -->
        <form @submit.prevent="handleLogin" class="custom-form">
          <!-- Username -->
          <div class="field">
            <div class="label">用户标识 / Username</div>
            <div class="relative">
              <input
                v-model="form.username"
                type="text"
                autocomplete="username"
                class="login-input w-full px-4 py-3 text-sm outline-none transition-all"
              />
            </div>
          </div>

          <!-- Password -->
          <div class="field">
            <div class="label">访问密钥 / Password</div>
            <div class="relative">
              <input
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                class="login-input w-full px-4 py-3 pr-11 text-sm outline-none transition-all"
              />
              <button
                type="button"
                @click="showPassword = !showPassword"
                class="absolute right-3 top-1/2 -translate-y-1/2 transition-colors login-eye-btn"
              >
                <EyeOff v-if="showPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <!-- Options -->
          <div class="options">
            <div class="remember-check">记住访问权限</div>
          </div>

          <!-- Terminal decorator -->
          <div class="terminal-decorator compact-terminal">
            <span class="status-dot"></span>
            <span class="terminal-code">AUTH_MODE: LOCAL_DATABASE</span>
          </div>

          <!-- Error -->
          <div v-if="errorMsg" class="error-message">
            <span>{{ errorMsg }}</span>
          </div>

          <!-- Submit -->
          <button
            type="submit"
            :disabled="loading"
            class="login-submit-btn w-full flex items-center justify-center gap-2 py-3 text-sm font-semibold transition-all disabled:opacity-60 disabled:cursor-not-allowed mt-2"
          >
            <svg v-if="loading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            <LogIn v-else class="w-4 h-4" />
            {{ loading ? '登录中...' : '立即登录' }}
          </button>

          <!-- Copyright -->
          <div class="copyright-text panel-copyright">© {{ new Date().getFullYear() }} EdgeOS</div>
        </form>
      </div>
    </div>
  </div>
</template>

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
  border-radius: 0;
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
  border-radius: 0;
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
  border-radius: 0;
  pointer-events: none;
  z-index: 0;
  clip-path: polygon(0 100%, 100% 100%, 100% 100%, 0 100%);
  transition: clip-path 0.1s linear, border-color 0.3s ease;
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

/* 其他样式 */
.field { margin-bottom: 20px; }
.label { font-size: 12px; font-weight: 700; color: #475569; margin-bottom: 8px; }

.options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 14px 0 10px;
}

/* 保持 Logo 工业感样式 */
.logo-icon {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 2px solid #0ea5e9;
  border-radius: 0;
  padding: 6px 12px;
  margin-right: 16px;
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

/* ===== 表单 ===== */
.custom-form {
  display: flex;
  flex-direction: column;
  gap: 0;
}

/* ===== 输入框 ===== */
.login-input {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  color: #0f172a;
  transition: border-color 0.15s ease, background-color 0.2s ease;
  border-radius: 0;
  box-shadow: none;
}

.login-input::placeholder { color: #94a3b8; }

.login-input:focus {
  border-color: #0ea5e9;
  outline: none;
  box-shadow: none;
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
  border-radius: 0;
  color: #ef4444;
  font-size: 13px;
  margin-bottom: 14px;
}

/* ===== 提交按钮 ===== */
.login-submit-btn { 
  height: 50px;
  margin-top: 10px;
  background: #0ea5e9;
  color: white;
  border: none;
  border-radius: 0;
  box-shadow: none;
  transition: all 0.2s ease;
}

.login-submit-btn:hover {
  transform: none;
  box-shadow: none;
  background: #0284c7;
}

.login-submit-btn:active {
  transform: none;
  box-shadow: none;
}

.terminal-decorator {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-dot { 
  width: 6px; 
  height: 6px; 
  background: #22c55e; 
  animation: blink 1.5s infinite; 
}

@keyframes blink { 
  0%, 100% { opacity: 1; } 
  50% { opacity: 0.3; } 
}

.terminal-code {
  font-size: 10px;
  font-family: monaco, monospace;
  color: #94a3b8;
  letter-spacing: 0.5px;
}

.compact-terminal {
  justify-content: center;
  gap: 8px;
  margin-bottom: 10px;
}

.remember-check {
  font-size: 13px;
  color: #64748b;
}

.login-eye-btn { 
  color: #94a3b8;
}

.login-eye-btn:hover { 
  color: #0ea5e9;
}

.copyright-text {
  font-size: 11px;
  color: #94a3b8;
  font-family: monaco, monospace;
}

.panel-copyright {
  margin-top: 14px;
  text-align: center;
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
    border-radius: 0;
  }

  .panel-topbar,
  .options {
    flex-direction: column;
    align-items: stretch;
  }

  .panel-header-side {
    align-items: flex-start;
  }

  .login-submit-btn {
    width: 100%;
  }
}
</style>
