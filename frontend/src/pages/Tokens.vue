<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api'
import { useToastStore } from '@/stores/toast'
const toast = useToastStore()

const tokens = ref<any[]>([])
const loading = ref(false)
const showCreate = ref(false)
const newName = ref('')
const newDays = ref(0)
const newToken = ref<{ name: string; token: string } | null>(null)
const copiedToken = ref<string | null>(null)

onMounted(load)

async function load() {
  loading.value = true
  try {
    tokens.value = await api.listTokens()
  } catch (e: any) {
    toast.error(e.message || '加载 Token 失败')
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!newName.value.trim()) return
  
  try {
    const res: any = await api.createToken(newName.value.trim(), newDays.value)
    newToken.value = res
    newName.value = ''
    newDays.value = 0
    await load()
  } catch (e: any) {
    toast.error(e.message || '创建 Token 失败')
  }
}

async function remove(id: number, name: string) {
  if (!confirm(`确定删除 Token "${name}"?\n\n删除后该 Token 将立即失效,使用此 Token 的客户端将无法继续上传。`)) return
  
  try {
    await api.deleteToken(id)
    await load()
  } catch (e: any) {
    toast.error(e.message || '删除 Token 失败')
  }
}

function copy(t: string, label: string = 'Token') {
  navigator.clipboard.writeText(t)
  copiedToken.value = label
  setTimeout(() => (copiedToken.value = null), 2000)
}

function fmtDate(ts: number) {
  if (!ts) return '—'
  const date = new Date(ts * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function isExpired(token: any) {
  if (!token.expires_at) return false
  return token.expires_at * 1000 < Date.now()
}

function daysUntilExpiry(token: any) {
  if (!token.expires_at) return null
  const now = Date.now()
  const expiry = token.expires_at * 1000
  const days = Math.ceil((expiry - now) / (1000 * 60 * 60 * 24))
  return days
}

function getTokenStatus(token: any) {
  if (!token.last_used) return { text: '未使用', color: 'var(--fg-3)' }
  if (isExpired(token)) return { text: '已过期', color: 'var(--danger)' }
  
  const days = daysUntilExpiry(token)
  if (days === null) return { text: '永不过期', color: '#52c41a' }
  if (days <= 7) return { text: `${days} 天后过期`, color: 'var(--warning)' }
  return { text: `${days} 天后过期`, color: '#52c41a' }
}
</script>

<template>
  <div class="tokens-page">
    <!-- 页面头部 -->
    <div class="head">
      <div>
        <h2 class="section-title">API 令牌</h2>
        <p class="section-sub">
          用于第三方客户端上传。请妥善保管,泄露后立即删除重建。
        </p>
      </div>
      <button class="btn btn-primary" @click="showCreate = true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        新建 Token
      </button>
    </div>

    <!-- 使用说明 -->
    <div class="usage-card">
      <div class="usage-header">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
        <h3>使用方式</h3>
      </div>
      <div class="usage-body">
        <p>在请求头中添加 <code>X-Token</code> 字段进行认证:</p>
        <pre><code>curl -X POST https://your-domain/api/upload \
  -H "X-Token: YOUR_TOKEN" \
  -F "file=@image.jpg"</code></pre>
        <div class="usage-tips">
          <div class="tip">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
            <span>建议为不同的客户端创建不同的 Token,方便管理和撤销</span>
          </div>
          <div class="tip">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            </svg>
            <span>设置合理的过期时间,提高安全性</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading">
      <div class="loading-spinner"></div>
      <span>加载中…</span>
    </div>
    
    <!-- 空状态 -->
    <div v-else-if="!tokens.length" class="empty">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      <h3>还没有 API Token</h3>
      <p>创建第一个 Token,让第三方客户端可以上传图片</p>
      <button class="btn btn-primary" @click="showCreate = true">
        立即创建
      </button>
    </div>
    
    <!-- Token 列表 -->
    <div v-else class="tokens-table-wrap">
      <table class="tokens-table">
        <thead>
          <tr>
            <th>名称</th>
            <th>Token</th>
            <th>最后使用</th>
            <th>创建时间</th>
            <th>状态</th>
            <th class="actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in tokens" :key="t.id" :class="{ expired: isExpired(t) }">
            <td class="name-cell">
              <div class="name-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                  <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
                </svg>
              </div>
              <span>{{ t.name }}</span>
            </td>
            <td class="token-cell">
              <code class="token-code">{{ t.token.slice(0, 12) }}…{{ t.token.slice(-6) }}</code>
              <button 
                class="btn-copy" 
                @click="copy(t.token, t.id)"
                :class="{ copied: copiedToken === t.id }"
              >
                <svg v-if="copiedToken !== t.id" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
                {{ copiedToken === t.id ? '已复制' : '复制' }}
              </button>
            </td>
            <td class="date-cell">
              {{ fmtDate(t.last_used) }}
              <span v-if="t.last_used" class="date-sub">上次使用</span>
            </td>
            <td class="date-cell">
              {{ fmtDate(t.created_at) }}
            </td>
            <td class="status-cell">
              <span 
                class="status-badge" 
                :style="{ 
                  color: getTokenStatus(t).color,
                  background: getTokenStatus(t).color + '15'
                }"
              >
                {{ getTokenStatus(t).text }}
              </span>
              <div class="expiry-info" v-if="t.expires_at">
                {{ t.expires_at ? fmtDate(t.expires_at) : '永不' }} 过期
              </div>
            </td>
            <td class="actions-cell">
              <button class="btn btn-sm btn-danger" @click="remove(t.id, t.name)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <polyline points="3 6 5 6 21 6"/>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 新建 Token 弹窗 -->
    <div v-if="showCreate" class="modal-mask" @click.self="showCreate = false">
      <form class="modal" @submit.prevent="create">
        <div class="modal-head">
          <h3>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
            </svg>
            新建 API Token
          </h3>
        </div>
        <div class="modal-body">
          <label>
            <span class="label-text">名称 <span class="required">*</span></span>
            <input 
              class="input" 
              v-model="newName" 
              required 
              placeholder="例如:Mac PicGo"
              maxlength="100"
            />
            <span class="hint">用于标识此 Token 的用途或设备</span>
          </label>
          <label>
            <span class="label-text">有效期</span>
            <div class="days-input">
              <input 
                class="input" 
                type="number" 
                v-model.number="newDays" 
                min="0"
                placeholder="0"
              />
              <span class="days-label">天</span>
            </div>
            <span class="hint">0 表示永久有效,建议设置过期时间提高安全性</span>
          </label>
        </div>
        <div class="modal-foot">
          <button type="button" class="btn" @click="showCreate = false">取消</button>
          <button type="submit" class="btn btn-primary">创建</button>
        </div>
      </form>
    </div>

    <!-- Token 创建成功弹窗 -->
    <div v-if="newToken" class="modal-mask" @click.self="newToken = null">
      <div class="modal success-modal">
        <div class="modal-head success-header">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="24" height="24">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
          <h3>Token 已生成</h3>
        </div>
        <div class="modal-body">
          <div class="warning-box">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
            <div>
              <strong>请立即复制 Token!</strong>
              <p>此窗口关闭后将无法再次查看完整内容</p>
            </div>
          </div>
          <code class="token-display">{{ newToken.token }}</code>
        </div>
        <div class="modal-foot">
          <button class="btn btn-primary" @click="copy(newToken.token, 'new')">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
            </svg>
            {{ copiedToken === 'new' ? '已复制 ✓' : '复制 Token' }}
          </button>
          <button class="btn" @click="newToken = null">我已保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tokens-page {
  max-width: 1400px;
}

.head { 
  display: flex; 
  justify-content: space-between; 
  align-items: flex-end; 
  margin-bottom: 24px;
  gap: 16px;
  flex-wrap: wrap;
}

.usage-card {
  background: linear-gradient(135deg, var(--accent-soft), var(--bg-2));
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-lg);
  padding: 20px;
  margin-bottom: 24px;
}

.usage-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  color: var(--accent);
}

.usage-header h3 {
  font-size: 15px;
  font-weight: 600;
  margin: 0;
}

.usage-body p {
  font-size: 13px;
  color: var(--fg-2);
  margin: 0 0 12px;
}

.usage-body pre {
  margin: 0 0 16px;
  padding: 14px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow-x: auto;
}

.usage-body code {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--fg-2);
}

.usage-tips {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tip {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  color: var(--fg-2);
}

.tip svg {
  flex-shrink: 0;
  margin-top: 2px;
  color: #52c41a;
}

.loading, .empty { 
  text-align: center; 
  padding: 80px 20px; 
  color: var(--fg-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty svg {
  width: 72px;
  height: 72px;
  color: var(--fg-3);
  opacity: 0.5;
}

.empty h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--fg-2);
  margin: 0;
}

.empty p {
  font-size: 14px;
  margin: 0;
}

.tokens-table-wrap {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.tokens-table { 
  width: 100%; 
  border-collapse: collapse; 
}

.tokens-table th, 
.tokens-table td { 
  padding: 14px 16px; 
  text-align: left; 
  font-size: 13px; 
  border-bottom: 1px solid var(--border); 
}

.tokens-table th { 
  color: var(--fg-3); 
  font-weight: 600; 
  font-size: 11px; 
  text-transform: uppercase; 
  letter-spacing: 0.05em;
  background: var(--bg-3);
}

.tokens-table th.actions {
  text-align: right;
}

.tokens-table tr:last-child td { 
  border: none; 
}

.tokens-table tr.expired {
  opacity: 0.6;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.name-icon {
  color: var(--accent);
}

.token-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.token-code {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--fg-2);
  background: var(--bg);
  padding: 4px 8px;
  border-radius: 4px;
}

.btn-copy {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 12px;
  color: var(--fg-3);
  cursor: pointer;
  transition: all 0.2s;
}

.btn-copy:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.btn-copy.copied {
  background: #52c41a;
  border-color: #52c41a;
  color: white;
}

.date-cell {
  white-space: nowrap;
}

.date-sub {
  display: block;
  font-size: 11px;
  color: var(--fg-3);
  margin-top: 2px;
}

.status-cell {
  white-space: nowrap;
}

.status-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.expiry-info {
  font-size: 11px;
  color: var(--fg-3);
  margin-top: 4px;
}

.actions-cell {
  text-align: right;
}

.modal-head h3 {
  display: flex;
  align-items: center;
  gap: 8px;
}

.modal-body label { 
  display: flex; 
  flex-direction: column; 
  gap: 6px; 
  margin-bottom: 18px; 
  font-size: 14px; 
  color: var(--fg-2); 
}

.label-text {
  font-weight: 500;
  font-size: 13px;
}

.required {
  color: var(--danger);
}

.hint {
  font-size: 12px;
  color: var(--fg-3);
  margin-top: 4px;
}

.days-input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.days-input .input {
  flex: 1;
}

.days-label {
  color: var(--fg-2);
  font-size: 14px;
}

.warning-box {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px;
  background: rgba(250, 173, 48, 0.1);
  border: 1px solid rgba(250, 173, 48, 0.3);
  border-radius: var(--radius);
  color: var(--warning);
  margin-bottom: 16px;
}

.warning-box svg {
  flex-shrink: 0;
  margin-top: 2px;
}

.warning-box strong {
  display: block;
  margin-bottom: 4px;
  font-size: 14px;
}

.warning-box p {
  margin: 0;
  font-size: 13px;
  opacity: 0.9;
}

.token-display {
  display: block;
  background: var(--bg);
  padding: 14px;
  border-radius: var(--radius);
  word-break: break-all;
  font-family: var(--font-mono);
  font-size: 13px;
  border: 1px solid var(--border);
  line-height: 1.6;
}

.success-header {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #52c41a;
}

.success-header h3 {
  margin: 0;
}

@media (max-width: 1024px) {
  .tokens-table-wrap {
    overflow-x: auto;
  }
  
  .tokens-table {
    min-width: 900px;
  }
}
</style>
