<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { api } from '@/api'
import { useToastStore } from '@/stores/toast'
const toast = useToastStore()

const tab = ref<'users' | 'settings' | 'storage'>('users')
const users = ref<any[]>([])
const settings = ref<any>({})
const showCreate = ref(false)
const newUser = ref({ username: '', password: '', role: 'user', quota_mb: 0 })
const saving = ref(false)
const saved = ref(false)
const savingTab = ref(false)
const savedTab = ref(false)
const searchUser = ref('')

// 云存储预设
type Provider = 's3' | 'qiniu' | 'aliyun' | 'tencent'
const providerInfo: Record<Provider, { label: string; hint: string; endpointHint: string; regionHint: string; icon: string }> = {
  s3:      { 
    label: '通用 S3', 
    hint: 'AWS S3、MinIO、Cloudflare R2、Backblaze B2 等任何 S3 兼容存储', 
    endpointHint: 's3.us-east-1.amazonaws.com', 
    regionHint: 'us-east-1',
    icon: '☁️'
  },
  qiniu:   { 
    label: '七牛云 Kodo', 
    hint: '七牛对象存储。请在七牛后台开启 S3 兼容协议', 
    endpointHint: 's3-cn-east-1.qiniucs.com', 
    regionHint: 'cn-east-1',
    icon: '📦'
  },
  aliyun:  { 
    label: '阿里云 OSS', 
    hint: '阿里云对象存储', 
    endpointHint: 'oss-cn-hangzhou.aliyuncs.com', 
    regionHint: 'cn-hangzhou',
    icon: '🔶'
  },
  tencent: { 
    label: '腾讯云 COS', 
    hint: '腾讯云对象存储。bucket 名需带 AppID 后缀', 
    endpointHint: 'cos.ap-guangzhou.myqcloud.com', 
    regionHint: 'ap-guangzhou',
    icon: '🔷'
  },
}
const providers: Provider[] = ['s3', 'qiniu', 'aliyun', 'tencent']

const currentProvider = computed<Provider>(() => {
  const d = settings.value.storage_driver as Provider
  return providers.includes(d) ? d : 's3'
})

// 过滤用户
const filteredUsers = computed(() => {
  if (!searchUser.value.trim()) return users.value
  const keyword = searchUser.value.toLowerCase()
  return users.value.filter(u => u.username.toLowerCase().includes(keyword))
})

onMounted(async () => {
  await loadUsers()
  await loadSettings()
})

async function loadUsers() {
  try {
    users.value = await api.admin.listUsers()
  } catch (e: any) {
    toast.error(e.message || '加载用户失败')
  }
}

async function loadSettings() {
  try {
    settings.value = await api.admin.getSettings()
  } catch (e: any) {
    toast.error(e.message || '加载设置失败')
  }
}

async function createUser() {
  if (!newUser.value.username || !newUser.value.password) return
  
  try {
    await api.admin.createUser(newUser.value)
    showCreate.value = false
    newUser.value = { username: '', password: '', role: 'user', quota_mb: 0 }
    await loadUsers()
  } catch (e: any) {
    toast.error(e.message || '创建用户失败')
  }
}

async function toggleDisabled(u: any) {
  try {
    await api.admin.updateUser(u.id, { ...u, disabled: !u.disabled })
    await loadUsers()
  } catch (e: any) {
    toast.error(e.message || '更新用户失败')
  }
}

async function deleteUser(u: any) {
  if (!confirm(`确定删除用户 ${u.username}?\n\n注意:此操作不会删除该用户的图片文件。`)) return
  
  try {
    await api.admin.deleteUser(u.id)
    await loadUsers()
  } catch (e: any) {
    toast.error(e.message || '删除用户失败')
  }
}

async function saveSettings() {
  saving.value = true
  try {
    const payload = { ...settings.value }
    // 秘钥若是掩码则不提交(保留服务端当前值)
    if (payload.storage_s3_secret && String(payload.storage_s3_secret).includes('•')) {
      delete payload.storage_s3_secret
    }
    // 站点设置只保存图片处理和上传相关配置，不覆盖存储驱动
    const settingsPayload: Record<string, any> = {}
    const siteKeys = [
      'allow_anonymous', 'max_size_mb', 'auto_compress', 'quality',
      'convert_webp', 'strip_exif', 'watermark_enabled', 'watermark_text',
      'upload_per_minute', 'thumbnail'
    ]
    for (const key of siteKeys) {
      if (key in payload) {
        settingsPayload[key] = payload[key]
      }
    }
    await api.admin.updateSettings(settingsPayload)
    saved.value = true
    setTimeout(() => (saved.value = false), 2000)
    await loadSettings()
  } catch (e: any) {
    toast.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function saveStorageSettings() {
  savingTab.value = true
  testResult.value = null // 保存时清空测试结果，避免显示两个✓
  try {
    const payload = { ...settings.value }
    // 秘钥若是掩码则不提交(保留服务端当前值)
    if (payload.storage_s3_secret && String(payload.storage_s3_secret).includes('•')) {
      delete payload.storage_s3_secret
    }
    // 存储设置只保存存储相关配置
    const storagePayload: Record<string, any> = {
      storage_driver: payload.storage_driver,
      storage_s3_endpoint: payload.storage_s3_endpoint,
      storage_s3_region: payload.storage_s3_region,
      storage_s3_bucket: payload.storage_s3_bucket,
      storage_s3_access: payload.storage_s3_access,
      storage_s3_ssl: payload.storage_s3_ssl,
      storage_s3_path: payload.storage_s3_path,
      storage_s3_public: payload.storage_s3_public,
      storage_s3_prefix: payload.storage_s3_prefix,
    }
    // 只有非掩码的secret才提交
    if (payload.storage_s3_secret && !String(payload.storage_s3_secret).includes('•')) {
      storagePayload.storage_s3_secret = payload.storage_s3_secret
    }
    await api.admin.updateSettings(storagePayload)
    savedTab.value = true
    setTimeout(() => (savedTab.value = false), 2000)
    await loadSettings()
  } catch (e: any) {
    toast.error(e.message || '保存失败')
  } finally {
    savingTab.value = false
  }
}

const testing = ref(false)
const testResult = ref<{ ok: boolean; message: string } | null>(null)

async function testStorage() {
  testing.value = true
  testResult.value = null
  try {
    const res = await api.admin.testStorage({
      driver: settings.value.storage_driver,
      endpoint: settings.value.storage_s3_endpoint,
      region: settings.value.storage_s3_region,
      bucket: settings.value.storage_s3_bucket,
      access_key: settings.value.storage_s3_access,
      secret_key: settings.value.storage_s3_secret,
      use_ssl: settings.value.storage_s3_ssl,
      path_style: settings.value.storage_s3_path,
      prefix: settings.value.storage_s3_prefix,
    })
    testResult.value = { ok: true, message: res.message || '连接成功' }
  } catch (e: any) {
    testResult.value = { ok: false, message: e.message || '连接失败' }
  } finally {
    testing.value = false
  }
}

function setProvider(p: Provider) {
  if (settings.value.storage_driver === p) return
  // 切换到不同服务商时清空所有凭证字段，避免把七牛的 key 错发到阿里云
  settings.value.storage_driver = p
  const info = providerInfo[p]
  settings.value.storage_s3_endpoint = info.endpointHint
  settings.value.storage_s3_region = info.regionHint
  settings.value.storage_s3_bucket = ''
  settings.value.storage_s3_access = ''
  settings.value.storage_s3_secret = ''
  settings.value.storage_s3_public = ''
  settings.value.storage_s3_prefix = ''
  settings.value.storage_s3_ssl = true
  settings.value.storage_s3_path = false
  testResult.value = null
}

function fmtSize(b: number) {
  if (b === 0) return '0 B'
  if (b < 1024) return b + ' B'
  if (b < 1024 * 1024) return (b / 1024).toFixed(0) + ' KB'
  return (b / 1024 / 1024).toFixed(1) + ' MB'
}

function fmtDate(ts: number) {
  if (!ts) return '—'
  return new Date(ts * 1000).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  })
}

const tabItems = [
  { key: 'users' as const, label: '用户管理', icon: '👥' },
  { key: 'settings' as const, label: '站点设置', icon: '⚙️' },
  { key: 'storage' as const, label: '存储设置', icon: '💾' },
]
</script>

<template>
  <div class="admin-page">
    <!-- 标签页导航 -->
    <div class="tabs">
      <div 
        v-for="t in tabItems" 
        :key="t.key"
        class="tab" 
        :class="{ active: tab === t.key }" 
        @click="tab = t.key"
      >
        <span class="tab-icon">{{ t.icon }}</span>
        <span class="tab-label">{{ t.label }}</span>
      </div>
    </div>

    <!-- 用户管理 -->
    <div v-if="tab === 'users'" class="tab-content">
      <div class="head">
        <div>
          <h2 class="section-title">用户管理</h2>
          <p class="section-sub">共 {{ users.length }} 个用户</p>
        </div>
        <div class="head-actions">
          <div class="search-box">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
              <circle cx="11" cy="11" r="8"/>
              <path d="m21 21-4.35-4.35"/>
            </svg>
            <input 
              class="input" 
              v-model="searchUser" 
              placeholder="搜索用户..."
            />
          </div>
          <button class="btn btn-primary" @click="showCreate = true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
              <line x1="12" y1="5" x2="12" y2="19"/>
              <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            新建用户
          </button>
        </div>
      </div>
      
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>角色</th>
              <th>配额</th>
              <th>已用</th>
              <th>状态</th>
              <th>创建时间</th>
              <th class="actions">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in filteredUsers" :key="u.id">
              <td>{{ u.id }}</td>
              <td class="username-cell">
                <div class="user-avatar">{{ u.username.charAt(0).toUpperCase() }}</div>
                <span>{{ u.username }}</span>
              </td>
              <td>
                <span class="chip role" :class="{ admin: u.role === 'admin' }">
                  {{ u.role === 'admin' ? '👑' : '👤' }} {{ u.role }}
                </span>
              </td>
              <td>{{ u.quota_mb ? u.quota_mb + ' MB' : '♾️ 无限' }}</td>
              <td>
                <div class="usage-bar" v-if="u.quota_mb > 0">
                  <div 
                    class="usage-fill" 
                    :style="{ width: Math.min(100, (u.used_bytes / (u.quota_mb * 1024 * 1024)) * 100) + '%' }"
                    :class="{ warning: (u.used_bytes / (u.quota_mb * 1024 * 1024)) > 0.8 }"
                  ></div>
                </div>
                {{ fmtSize(u.used_bytes) }}
              </td>
              <td>
                <span class="status-badge" :class="{ disabled: u.disabled }">
                  {{ u.disabled ? '🚫 已禁用' : '✅ 正常' }}
                </span>
              </td>
              <td>{{ fmtDate(u.created_at) }}</td>
              <td class="actions-cell">
                <button 
                  class="btn btn-sm" 
                  :class="u.disabled ? 'btn-success' : 'btn-warning'"
                  @click="toggleDisabled(u)"
                >
                  {{ u.disabled ? '启用' : '禁用' }}
                </button>
                <button 
                  v-if="u.id !== 1"
                  class="btn btn-sm btn-danger" 
                  @click="deleteUser(u)"
                >
                  删除
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        
        <div v-if="!filteredUsers.length" class="empty-result">
          没有找到匹配的用户
        </div>
      </div>
    </div>

    <!-- 站点设置 -->
    <div v-else-if="tab === 'settings'" class="tab-content">
      <div class="settings-header">
        <div>
          <h2 class="section-title">站点设置</h2>
          <p class="section-sub">实时生效,无需重启服务</p>
        </div>
        <button class="btn btn-primary" :disabled="saving" @click="saveSettings">
          <svg v-if="saved" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          {{ saved ? '已保存' : saving ? '保存中…' : '保存设置' }}
        </button>
      </div>
      
      <div class="settings-grid">
        <div class="settings-card">
          <h3>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="17 8 12 3 7 8"/>
              <line x1="12" y1="3" x2="12" y2="15"/>
            </svg>
            上传设置
          </h3>
          <div class="field">
            <label>单文件大小上限(MB)
              <input class="input" type="number" v-model.number="settings.max_size_mb" min="1" />
            </label>
          </div>
          <div class="field">
            <label>上传限流(次/分钟,0 = 不限)
              <input class="input" type="number" v-model.number="settings.upload_per_minute" min="0" />
            </label>
          </div>
        </div>
        
        <div class="settings-card">
          <h3>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <circle cx="8.5" cy="8.5" r="1.5"/>
              <polyline points="21 15 16 10 5 21"/>
            </svg>
            图片处理
          </h3>
          <div class="field checkbox">
            <label>
              <input type="checkbox" v-model="settings.auto_compress" />
              <span>自动压缩上传图片</span>
            </label>
          </div>
          <div class="field" v-if="settings.auto_compress">
            <label>压缩质量(1-100)
              <input class="input" type="number" v-model.number="settings.quality" min="1" max="100" />
            </label>
          </div>
          <div class="field checkbox">
            <label>
              <input type="checkbox" v-model="settings.convert_webp" />
              <span>自动转换为 WebP 格式</span>
            </label>
          </div>
          <div class="field checkbox">
            <label>
              <input type="checkbox" v-model="settings.strip_exif" />
              <span>清除 EXIF 元数据</span>
            </label>
          </div>
          <div class="field checkbox">
            <label>
              <input type="checkbox" v-model="settings.thumbnail" />
              <span>生成缩略图</span>
            </label>
          </div>
        </div>
        
        <div class="settings-card">
          <h3>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            </svg>
            安全设置
          </h3>
          <div class="field checkbox">
            <label>
              <input type="checkbox" v-model="settings.watermark_enabled" />
              <span>启用水印功能</span>
            </label>
          </div>
          <div class="field" v-if="settings.watermark_enabled">
            <label>水印文字
              <input class="input" v-model="settings.watermark_text" placeholder="例如:© My Site" />
            </label>
          </div>
        </div>
      </div>
    </div>

    <!-- 存储设置 -->
    <div v-else class="tab-content">
      <div class="settings-header">
        <div>
          <h2 class="section-title">存储后端</h2>
          <p class="section-sub">选择图片实际存放的位置。切换后立即生效,不影响历史图片的访问 URL</p>
        </div>
        <button class="btn btn-primary" :disabled="savingTab" @click="saveStorageSettings">
          <svg v-if="savedTab" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          {{ savedTab ? '已保存' : savingTab ? '保存中…' : '保存存储设置' }}
        </button>
      </div>

      <!-- 存储提供商选择 -->
      <div class="providers">
        <div
          v-for="p in providers"
          :key="p"
          class="provider-card"
          :class="{ active: currentProvider === p }"
          @click="setProvider(p)"
        >
          <div class="provider-icon">{{ providerInfo[p].icon }}</div>
          <div class="provider-label">{{ providerInfo[p].label }}</div>
          <div class="provider-hint">{{ providerInfo[p].hint }}</div>
        </div>
      </div>

      <!-- 存储凭证配置 -->
      <div class="settings-card">
        <div class="card-head">
          <h3>
            {{ providerInfo[currentProvider].icon }} {{ providerInfo[currentProvider].label }} · 凭证配置
          </h3>
        </div>
        <div class="card-body">
          <div class="settings-grid-2">
            <div class="field">
              <label>Endpoint
                <input 
                  class="input" 
                  v-model="settings.storage_s3_endpoint" 
                  :placeholder="providerInfo[currentProvider].endpointHint" 
                />
              </label>
              <p class="field-hint">不带 http(s):// 前缀</p>
            </div>
            <div class="field">
              <label>Region 
                <input 
                  class="input" 
                  v-model="settings.storage_s3_region" 
                  :placeholder="providerInfo[currentProvider].regionHint" 
                />
              </label>
            </div>
            <div class="field">
              <label>Bucket 
                <input class="input" v-model="settings.storage_s3_bucket" placeholder="my-bucket" />
              </label>
              <p class="field-hint" v-if="currentProvider === 'tencent'">
                腾讯云 COS 的 bucket 需带 AppID 后缀,如 my-bucket-1250000000
              </p>
            </div>
            <div class="field">
              <label>Access Key 
                <input class="input" v-model="settings.storage_s3_access" placeholder="AKID..." />
              </label>
            </div>
            <div class="field">
              <label>Secret Key
                <input 
                  class="input" 
                  type="password" 
                  v-model="settings.storage_s3_secret" 
                  :placeholder="settings.storage_s3_secret?.includes('•') ? '（已保存,留空保持不变）' : 'SK...'" 
                />
              </label>
            </div>
            <div class="field">
              <label>对象 Key 前缀(可选)
                <input class="input" v-model="settings.storage_s3_prefix" placeholder="pichub" />
              </label>
            </div>
          </div>
          
          <div class="divider"></div>
          
          <div class="settings-grid-2">
            <div class="field checkbox">
              <label>
                <input type="checkbox" v-model="settings.storage_s3_ssl" />
                <span>启用 HTTPS(推荐)</span>
              </label>
            </div>
            <div class="field checkbox">
              <label>
                <input type="checkbox" v-model="settings.storage_s3_path" />
                <span>Path-Style 访问(MinIO 等自建场景)</span>
              </label>
            </div>
          </div>
          
          <div class="field">
            <label>对外访问前缀(可选)
              <input
                class="input"
                v-model="settings.storage_s3_public"
                placeholder="https://cdn.example.com"
              />
            </label>
            <p class="field-hint">用于生成图片 URL。留空则使用 {endpoint}/{bucket}/{key}</p>
          </div>

          <div class="test-row">
            <button class="btn" :disabled="testing" @click="testStorage">
              {{ testing ? '测试中…' : '测试连接' }}
            </button>
            <div v-if="testResult" class="test-result" :class="testResult.ok ? 'ok' : 'err'">
              {{ testResult.ok ? '✓ ' : '✗ ' }}{{ testResult.message }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建用户弹窗 -->
    <div v-if="showCreate" class="modal-mask" @click.self="showCreate = false">
      <form class="modal" @submit.prevent="createUser">
        <div class="modal-head">
          <h3>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
              <circle cx="8.5" cy="7" r="4"/>
              <line x1="20" y1="8" x2="20" y2="14"/>
              <line x1="23" y1="11" x2="17" y2="11"/>
            </svg>
            新建用户
          </h3>
        </div>
        <div class="modal-body">
          <label>
            <span class="label-text">用户名 <span class="required">*</span></span>
            <input class="input" v-model="newUser.username" required placeholder="请输入用户名" />
          </label>
          <label>
            <span class="label-text">密码 <span class="required">*</span></span>
            <input class="input" type="password" v-model="newUser.password" required placeholder="至少 8 位" />
          </label>
          <label>
            <span class="label-text">角色</span>
            <select class="input" v-model="newUser.role">
              <option value="user">👤 普通用户</option>
              <option value="admin">👑 管理员</option>
            </select>
          </label>
          <label>
            <span class="label-text">存储配额(MB)</span>
            <input class="input" type="number" v-model.number="newUser.quota_mb" min="0" placeholder="0 = 无限" />
            <span class="hint">设置此用户可使用的最大存储空间,0 表示无限制</span>
          </label>
        </div>
        <div class="modal-foot">
          <button type="button" class="btn" @click="showCreate = false">取消</button>
          <button type="submit" class="btn btn-primary">创建用户</button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.admin-page {
  max-width: 1400px;
}

.tabs { 
  display: flex; 
  gap: 2px; 
  border-bottom: 2px solid var(--border); 
  margin-bottom: 24px; 
}

.tab { 
  padding: 12px 18px; 
  font-size: 14px; 
  color: var(--fg-3); 
  cursor: pointer; 
  border-bottom: 2px solid transparent; 
  margin-bottom: -2px;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s;
  font-weight: 500;
}

.tab:hover {
  color: var(--fg);
}

.tab.active { 
  color: var(--accent); 
  border-bottom-color: var(--accent); 
}

.tab-icon {
  font-size: 16px;
}

.head { 
  display: flex; 
  justify-content: space-between; 
  align-items: flex-end; 
  margin-bottom: 20px;
  gap: 16px;
  flex-wrap: wrap;
}

.head-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-box svg {
  position: absolute;
  left: 10px;
  width: 16px;
  height: 16px;
  color: var(--fg-3);
  pointer-events: none;
}

.search-box .input {
  padding-left: 34px;
  width: 240px;
}

.table-wrap {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.table { 
  width: 100%; 
  border-collapse: collapse; 
}

.table th, 
.table td { 
  padding: 12px 16px; 
  text-align: left; 
  font-size: 13px; 
  border-bottom: 1px solid var(--border); 
}

.table th { 
  color: var(--fg-3); 
  font-weight: 600; 
  font-size: 11px; 
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--bg-3);
}

.table th.actions {
  text-align: right;
}

.table tr:last-child td { 
  border: none; 
}

.table tbody tr {
  transition: background 0.2s;
}

.table tbody tr:hover {
  background: var(--bg-3);
}

.username-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
}

.user-avatar {
  width: 32px;
  height: 32px;
  background: var(--accent-soft);
  color: var(--accent);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  flex-shrink: 0;
}

.chip.role {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  background: var(--bg-3);
}

.chip.role.admin {
  background: var(--accent-soft);
  color: var(--accent);
}

.usage-bar {
  width: 100%;
  height: 4px;
  background: var(--bg-3);
  border-radius: 2px;
  margin-bottom: 4px;
  overflow: hidden;
}

.usage-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.usage-fill.warning {
  background: var(--warning);
}

.status-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  background: rgba(82, 196, 26, 0.1);
  color: #52c41a;
}

.status-badge.disabled {
  background: rgba(235, 87, 87, 0.1);
  color: var(--danger);
}

.actions-cell {
  text-align: right;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.btn-success {
  background: #52c41a;
  color: white;
  border-color: #52c41a;
}

.btn-warning {
  background: var(--warning);
  color: white;
  border-color: var(--warning);
}

.empty-result {
  text-align: center;
  padding: 40px;
  color: var(--fg-3);
}

/* 设置页面 */
.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 24px;
  gap: 16px;
  flex-wrap: wrap;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
}

.settings-card {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 20px;
}

.settings-card h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 16px;
  color: var(--fg);
}

.settings-grid-2 {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.field { 
  margin-bottom: 16px; 
}

.field label { 
  display: flex; 
  flex-direction: column; 
  gap: 6px; 
  font-size: 13px; 
  color: var(--fg-2); 
  font-weight: 500;
}

.field.checkbox label {
  flex-direction: row;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}

.field.checkbox input[type="checkbox"] {
  width: 18px;
  height: 18px;
  accent-color: var(--accent);
  cursor: pointer;
}

.field-hint { 
  font-size: 12px; 
  color: var(--fg-3); 
  margin-top: 6px;
}

.divider {
  height: 1px;
  background: var(--border);
  margin: 20px 0;
}

.card-head { 
  padding: 16px 20px; 
  border-bottom: 1px solid var(--border);
  background: var(--bg-3);
}

.card-head h3 { 
  font-size: 15px; 
  font-weight: 600;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-body {
  padding: 20px;
}

/* 存储提供商 */
.providers { 
  display: grid; 
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); 
  gap: 16px; 
  margin-bottom: 24px; 
}

.provider-card {
  background: var(--bg-2);
  border: 2px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 18px;
  cursor: pointer;
  transition: all 0.25s ease;
}

.provider-card:hover { 
  border-color: var(--border-2);
  transform: translateY(-2px);
}

.provider-card.active { 
  border-color: var(--accent); 
  background: var(--accent-soft);
}

.provider-icon {
  font-size: 32px;
  margin-bottom: 10px;
}

.provider-label { 
  font-size: 14px; 
  font-weight: 600; 
  color: var(--fg); 
  margin-bottom: 6px; 
}

.provider-hint {
  font-size: 12px;
  color: var(--fg-3);
  line-height: 1.5;
}

.test-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--border);
  flex-wrap: wrap;
}

.test-result {
  font-size: 13px;
  padding: 6px 12px;
  border-radius: var(--radius);
  flex: 1;
  min-width: 0;
}

.test-result.ok {
  background: rgba(0, 112, 243, 0.08);
  color: var(--success);
  border: 1px solid rgba(0, 112, 243, 0.2);
}

.test-result.err {
  background: rgba(238, 0, 0, 0.06);
  color: var(--danger);
  border: 1px solid rgba(238, 0, 0, 0.2);
}

/* 模态框 */
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

@media (max-width: 1024px) {
  .settings-grid-2 {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .tabs {
    overflow-x: auto;
  }
  
  .tab {
    white-space: nowrap;
  }
  
  .head {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .head-actions {
    width: 100%;
    flex-direction: column;
  }
  
  .search-box .input {
    width: 100%;
  }
  
  .settings-grid {
    grid-template-columns: 1fr;
  }
  
  .providers {
    grid-template-columns: 1fr;
  }
  
  .table-wrap {
    overflow-x: auto;
  }
  
  .table {
    min-width: 800px;
  }
}
</style>
