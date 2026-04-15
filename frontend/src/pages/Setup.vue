<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api, setToken } from '@/api'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const username = ref('admin')
const password = ref('')
const confirm = ref('')
const loading = ref(false)
const error = ref('')
const showPassword = ref(false)
const showConfirmPassword = ref(false)

// 表单验证状态
const passwordTouched = ref(false)
const confirmTouched = ref(false)

// 密码强度检测
const passwordStrength = computed(() => {
  const pwd = password.value
  if (!pwd) return { level: 0, text: '', color: '' }
  
  let level = 0
  if (pwd.length >= 8) level++
  if (pwd.length >= 12) level++
  if (/[a-z]/.test(pwd) && /[A-Z]/.test(pwd)) level++
  if (/\d/.test(pwd)) level++
  if (/[^a-zA-Z0-9]/.test(pwd)) level++
  
  if (level <= 2) return { level: 1, text: '弱', color: 'var(--danger)' }
  if (level === 3) return { level: 2, text: '中等', color: 'var(--warning)' }
  if (level === 4) return { level: 3, text: '强', color: '#52c41a' }
  return { level: 4, text: '非常强', color: '#52c41a' }
})

// 验证
const isPasswordValid = computed(() => password.value.length >= 8)
const isConfirmValid = computed(() => confirm.value === password.value && confirm.value !== '')

function validatePassword() {
  return isPasswordValid.value
}

function validateConfirm() {
  return isConfirmValid.value
}

async function submit() {
  error.value = ''
  
  if (!validatePassword()) {
    error.value = '密码至少需要 8 个字符'
    return
  }
  
  if (!validateConfirm()) {
    error.value = '两次输入的密码不一致'
    return
  }
  
  loading.value = true
  try {
    const res = await api.setupInit({
      username: username.value.trim(),
      password: password.value,
    })
    setToken(res.token)
    userStore.user = res.user
    router.push('/upload')
  } catch (e: any) {
    error.value = e.message || '初始化失败,请稍后重试'
  } finally {
    loading.value = false
  }
}

function togglePassword() {
  showPassword.value = !showPassword.value
}

function toggleConfirmPassword() {
  showConfirmPassword.value = !showConfirmPassword.value
}
</script>

<template>
  <div class="setup-wrap">
    <form class="setup-card" @submit.prevent="submit" novalidate>
      <div class="logo">
        <div class="logo-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
        </div>
        <span class="logo-text">PicHub</span>
      </div>
      
      <div class="header">
        <h1>首次使用 · 创建管理员</h1>
        <p class="sub">
          欢迎使用 PicHub 图床。请设置管理员账户,此密码将用于登录管理后台。
        </p>
      </div>
      
      <div class="form-group">
        <label for="username">
          <span class="label-text">管理员用户名</span>
          <span class="required">*</span>
        </label>
        <input
          id="username"
          v-model="username"
          class="input"
          type="text"
          placeholder="请输入用户名"
          required
          autofocus
          minlength="2"
          maxlength="32"
        />
        <span class="hint">2-32 个字符</span>
      </div>
      
      <div class="form-group">
        <label for="password">
          <span class="label-text">密码</span>
          <span class="required">*</span>
        </label>
        <div class="password-wrapper">
          <input
            id="password"
            v-model="password"
            class="input"
            :class="{ 
              'input-error': passwordTouched && !isPasswordValid,
              'input-success': passwordTouched && isPasswordValid 
            }"
            :type="showPassword ? 'text' : 'password'"
            placeholder="至少 8 个字符"
            required
            minlength="8"
            @blur="passwordTouched = true"
          />
          <button
            type="button"
            class="toggle-password"
            @click="togglePassword"
            :aria-label="showPassword ? '隐藏密码' : '显示密码'"
          >
            <svg v-if="!showPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
              <line x1="1" y1="1" x2="23" y2="23"/>
            </svg>
          </button>
        </div>
        
        <!-- 密码强度指示器 -->
        <div v-if="password" class="password-strength">
          <div class="strength-bars">
            <div 
              v-for="i in 4" 
              :key="i" 
              class="strength-bar"
              :class="{ 
                active: i <= passwordStrength.level,
                'weak': passwordStrength.level === 1,
                'medium': passwordStrength.level === 2,
                'strong': passwordStrength.level >= 3
              }"
            ></div>
          </div>
          <span class="strength-text" :style="{ color: passwordStrength.color }">
            {{ passwordStrength.text }}
          </span>
        </div>
        
        <span v-if="passwordTouched && !isPasswordValid" class="validation-msg">
          密码至少需要 8 个字符
        </span>
      </div>
      
      <div class="form-group">
        <label for="confirm">
          <span class="label-text">确认密码</span>
          <span class="required">*</span>
        </label>
        <div class="password-wrapper">
          <input
            id="confirm"
            v-model="confirm"
            class="input"
            :class="{ 
              'input-error': confirmTouched && !isConfirmValid,
              'input-success': confirmTouched && isConfirmValid 
            }"
            :type="showConfirmPassword ? 'text' : 'password'"
            placeholder="请再次输入密码"
            required
            @blur="confirmTouched = true"
            @input="confirmTouched && validateConfirm()"
          />
          <button
            type="button"
            class="toggle-password"
            @click="toggleConfirmPassword"
            :aria-label="showConfirmPassword ? '隐藏密码' : '显示密码'"
          >
            <svg v-if="!showConfirmPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
              <line x1="1" y1="1" x2="23" y2="23"/>
            </svg>
          </button>
        </div>
        <span v-if="confirmTouched && !isConfirmValid" class="validation-msg">
          {{ confirm ? '两次输入的密码不一致' : '请再次输入密码' }}
        </span>
      </div>
      
      <div v-if="error" class="error-alert" role="alert">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <span>{{ error }}</span>
      </div>
      
      <button class="btn btn-primary btn-block" :disabled="loading" :class="{ 'btn-loading': loading }">
        <svg v-if="loading" class="spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" stroke-dasharray="60" stroke-dashoffset="15"/>
        </svg>
        <span>{{ loading ? '创建中…' : '开始使用' }}</span>
      </button>
      
      <p class="footer-hint">
        此页面仅在首次启动时可用。创建后将自动登录并跳转到上传页。
      </p>
    </form>
  </div>
</template>

<style scoped>
.setup-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(135deg, var(--bg) 0%, var(--bg-2) 100%);
}

.setup-card {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: 40px;
  width: 100%;
  max-width: 480px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.1);
  animation: slideUp 0.4s ease-out;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.logo-icon {
  width: 36px;
  height: 36px;
  background: var(--accent);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.logo-icon svg {
  width: 22px;
  height: 22px;
}

.logo-text {
  font-size: 24px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--fg) 0%, var(--fg-2) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.header h1 {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 8px;
  line-height: 1.3;
}

.sub {
  color: var(--fg-3);
  font-size: 14px;
  line-height: 1.6;
  margin: 0;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 500;
  color: var(--fg-2);
}

.required {
  color: var(--danger);
  font-size: 14px;
}

.input {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg);
  border: 1.5px solid var(--border);
  border-radius: var(--radius);
  font-size: 14px;
  color: var(--fg);
  transition: all 0.2s ease;
}

.input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.input-error {
  border-color: var(--danger);
}

.input-error:focus {
  box-shadow: 0 0 0 3px rgba(235, 87, 87, 0.1);
}

.input-success {
  border-color: #52c41a;
}

.password-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.password-wrapper .input {
  padding-right: 44px;
}

.toggle-password {
  position: absolute;
  right: 8px;
  background: none;
  border: none;
  padding: 8px;
  cursor: pointer;
  color: var(--fg-3);
  transition: color 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.toggle-password:hover {
  color: var(--fg);
}

.toggle-password svg {
  width: 18px;
  height: 18px;
}

.password-strength {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
}

.strength-bars {
  display: flex;
  gap: 4px;
  flex: 1;
}

.strength-bar {
  flex: 1;
  height: 4px;
  background: var(--bg-3);
  border-radius: 2px;
  transition: all 0.3s ease;
}

.strength-bar.active.weak {
  background: var(--danger);
}

.strength-bar.active.medium {
  background: var(--warning);
}

.strength-bar.active.strong {
  background: #52c41a;
}

.strength-text {
  font-size: 12px;
  font-weight: 500;
  min-width: 50px;
  text-align: right;
}

.hint {
  font-size: 12px;
  color: var(--fg-3);
}

.validation-msg {
  font-size: 12px;
  color: var(--danger);
  display: flex;
  align-items: center;
  gap: 4px;
}

.error-alert {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: rgba(235, 87, 87, 0.08);
  border: 1px solid rgba(235, 87, 87, 0.3);
  border-radius: var(--radius);
  color: var(--danger);
  font-size: 13px;
  animation: shake 0.4s ease;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-8px); }
  75% { transform: translateX(8px); }
}

.error-alert svg {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.btn-block {
  width: 100%;
  padding: 12px;
  margin-top: 8px;
  font-size: 15px;
  font-weight: 500;
  position: relative;
}

.btn-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.spinner {
  width: 18px;
  height: 18px;
  animation: rotate 0.8s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.footer-hint {
  color: var(--fg-4);
  font-size: 12px;
  text-align: center;
  margin: 8px 0 0;
  line-height: 1.5;
}

@media (max-width: 480px) {
  .setup-card {
    padding: 28px;
  }
  
  .logo-text {
    font-size: 20px;
  }
  
  .header h1 {
    font-size: 20px;
  }
}
</style>
