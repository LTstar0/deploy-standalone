<template>
  <!-- Loading state -->
  <div v-if="isLoading" class="loading-screen">
    <div class="spinner"></div>
    <p>正在初始化发布系统...</p>
  </div>

  <!-- Authentication screen -->
  <div v-else-if="!isAuthorized" class="auth-container">
    <div class="auth-card">
      <div class="auth-header">
        <div class="auth-logo">🚀</div>
        <h2>Deploy Platform</h2>
        <p class="subtitle">部署与发布平台 · 身份校验</p>
      </div>

      <form @submit.prevent="handleVerify" class="auth-form">
        <div class="input-group">
          <label for="token">发布授权 Token</label>
          <input
            id="token"
            v-model="tokenInput"
            type="password"
            placeholder="请输入授权 Token"
            :disabled="isVerifying"
            autocomplete="current-password"
            class="token-input"
          />
        </div>

        <div v-if="errorMessage" class="error-message">
          <span class="error-icon">⚠️</span>
          <span>{{ errorMessage }}</span>
        </div>

        <button type="submit" :disabled="isVerifying" class="auth-btn">
          <span v-if="isVerifying" class="btn-spinner"></span>
          <span v-else>验证并登录</span>
        </button>
      </form>

      <div class="auth-footer">
        <p>系统检测不到 DEPLOY_TOKEN 环境变量时</p>
        <p>已在后台控制台打印并生成默认本地 Token</p>
      </div>
    </div>
  </div>

  <!-- Normal app layout -->
  <div v-else class="app-layout">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <h1>🚀 Deploy</h1>
        <div class="subtitle">轻量部署平台</div>
      </div>
      <nav class="sidebar-nav">
        <router-link to="/packages" class="nav-item" active-class="active">
          <span class="icon">📦</span>
          <span>产品包管理</span>
        </router-link>
        <router-link to="/history" class="nav-item" active-class="active">
          <span class="icon">📋</span>
          <span>部署历史</span>
        </router-link>
      </nav>
      <div class="sidebar-footer">
        <button @click="handleLogout" class="logout-btn">
          <span class="icon">🚪</span>
          <span>退出登录</span>
        </button>
      </div>
    </aside>
    <main class="main-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { verifyToken } from './api'

const isAuthorized = ref(false)
const isLoading = ref(true)
const tokenInput = ref('')
const isVerifying = ref(false)
const errorMessage = ref('')

const checkAuth = async () => {
  const token = localStorage.getItem('deploy_token')
  if (!token) {
    isLoading.value = false
    isAuthorized.value = false
    return
  }
  try {
    await verifyToken(token)
    isAuthorized.value = true
  } catch (err) {
    localStorage.removeItem('deploy_token')
    isAuthorized.value = false
  } finally {
    isLoading.value = false
  }
}

const handleVerify = async () => {
  if (!tokenInput.value.trim()) {
    errorMessage.value = '请输入部署 Token'
    return
  }
  isVerifying.value = true
  errorMessage.value = ''
  try {
    await verifyToken(tokenInput.value.trim())
    localStorage.setItem('deploy_token', tokenInput.value.trim())
    isAuthorized.value = true
  } catch (err) {
    errorMessage.value = err.response?.data?.error || 'Token 验证失败，请重试'
  } finally {
    isVerifying.value = false
  }
}

const handleLogout = () => {
  localStorage.removeItem('deploy_token')
  isAuthorized.value = false
  tokenInput.value = ''
  errorMessage.value = ''
}

onMounted(() => {
  checkAuth()
})
</script>

<style scoped>
/* ── Loading Screen ───────────────────────────────────────────────────────── */
.loading-screen {
  position: fixed;
  inset: 0;
  background: #0a0e1a;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  color: #94a3b8;
  font-size: 15px;
  z-index: 1000;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(59, 130, 246, 0.1);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Authentication Layout ────────────────────────────────────────────────── */
.auth-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at center, #111827 0%, #0a0e1a 100%);
  padding: 24px;
}

.auth-card {
  width: 100%;
  max-width: 420px;
  background: rgba(26, 32, 53, 0.6);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.4), 0 0 40px rgba(59, 130, 246, 0.05);
  border-radius: 16px;
  padding: 40px 32px;
  display: flex;
  flex-direction: column;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(15px); }
  to { opacity: 1; transform: translateY(0); }
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;
}

.auth-logo {
  font-size: 44px;
  margin-bottom: 12px;
  display: inline-block;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

.auth-header h2 {
  font-size: 24px;
  font-weight: 800;
  background: linear-gradient(135deg, #3b82f6, #06b6d4);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 6px;
  letter-spacing: -0.5px;
}

.auth-header .subtitle {
  font-size: 13px;
  color: #64748b;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-group label {
  font-size: 13px;
  font-weight: 600;
  color: #94a3b8;
}

.token-input {
  background: #0d1117;
  border: 1px solid #1e293b;
  border-radius: 8px;
  padding: 12px 16px;
  color: #e2e8f0;
  font-size: 15px;
  outline: none;
  transition: all 0.2s;
}

.token-input:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 10px rgba(59, 130, 246, 0.25);
  background: #111827;
}

.auth-btn {
  background: linear-gradient(135deg, #3b82f6, #06b6d4);
  color: white;
  border: none;
  border-radius: 8px;
  padding: 12px;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  box-shadow: 0 4px 20px rgba(59, 130, 246, 0.3);
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.auth-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 24px rgba(59, 130, 246, 0.4);
}

.auth-btn:active:not(:disabled) {
  transform: translateY(1px);
}

.auth-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error-message {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.25);
  border-radius: 8px;
  padding: 10px 14px;
  color: #ef4444;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
  animation: shake 0.3s ease;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-4px); }
  75% { transform: translateX(4px); }
}

.btn-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.2);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.auth-footer {
  margin-top: 32px;
  text-align: center;
  font-size: 11px;
  color: #475569;
  line-height: 1.6;
}

/* ── Sidebar Logout ───────────────────────────────────────────────────────── */
.sidebar-footer {
  padding: 16px;
  border-top: 1px solid var(--border);
}

.logout-btn {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  color: var(--text-dim);
  font-size: 14px;
  font-weight: 500;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.logout-btn:hover {
  background: var(--red-soft);
  color: var(--red);
}
</style>
