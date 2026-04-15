<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api'
import { useToastStore } from '@/stores/toast'
const toast = useToastStore()

const albums = ref<any[]>([])
const loading = ref(false)
const showCreate = ref(false)
const newName = ref('')
const newDesc = ref('')
const newPublic = ref(false)
const editingAlbum = ref<any>(null)
const editName = ref('')
const editDesc = ref('')
const editPublic = ref(false)
const submitting = ref(false)

onMounted(load)

async function load() {
  loading.value = true
  try {
    albums.value = await api.listAlbums()
  } catch (e: any) {
    toast.error(e.message || '加载相册失败')
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!newName.value.trim()) return
  
  submitting.value = true
  try {
    await api.createAlbum(newName.value.trim(), newDesc.value.trim(), newPublic.value)
    showCreate.value = false
    newName.value = ''
    newDesc.value = ''
    newPublic.value = false
    await load()
  } catch (e: any) {
    toast.error(e.message || '创建相册失败')
  } finally {
    submitting.value = false
  }
}

async function updateAlbum() {
  if (!editingAlbum.value || !editName.value.trim()) return
  
  submitting.value = true
  try {
    await api.updateAlbum(editingAlbum.value.id, {
      name: editName.value.trim(),
      description: editDesc.value.trim(),
      is_public: editPublic.value
    })
    closeEdit()
    await load()
  } catch (e: any) {
    toast.error(e.message || '更新相册失败')
  } finally {
    submitting.value = false
  }
}

async function remove(id: number, name: string) {
  if (!confirm(`确定删除相册 "${name}"?\n\n注意:相册中的图片会移到默认相册,但不会删除图片文件。`)) return
  
  try {
    await api.deleteAlbum(id)
    await load()
  } catch (e: any) {
    toast.error(e.message || '删除相册失败')
  }
}

function openEdit(album: any) {
  editingAlbum.value = album
  editName.value = album.name
  editDesc.value = album.description || ''
  editPublic.value = album.is_public
}

function closeEdit() {
  editingAlbum.value = null
  editName.value = ''
  editDesc.value = ''
  editPublic.value = false
}

function fmtDate(ts: number) {
  if (!ts) return '—'
  return new Date(ts * 1000).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function getTotalStats() {
  const totalImages = albums.value.reduce((sum, a) => sum + (a.image_count || 0), 0)
  const publicCount = albums.value.filter(a => a.is_public).length
  return { totalImages, publicCount, totalCount: albums.value.length }
}
</script>

<template>
  <div class="albums-page">
    <!-- 页面头部 -->
    <div class="head">
      <div>
        <h2 class="section-title">相册管理</h2>
        <p class="section-sub">
          共 {{ albums.length }} 个相册 · 
          {{ getTotalStats().totalImages }} 张图片 · 
          {{ getTotalStats().publicCount }} 个公开
        </p>
      </div>
      <button class="btn btn-primary" @click="showCreate = true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        新建相册
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading">
      <div class="loading-spinner"></div>
      <span>加载中…</span>
    </div>
    
    <!-- 空状态 -->
    <div v-else-if="!albums.length" class="empty">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
      </svg>
      <h3>还没有相册</h3>
      <p>创建你的第一个相册,更好地组织图片</p>
      <button class="btn btn-primary" @click="showCreate = true">
        立即创建
      </button>
    </div>
    
    <!-- 相册网格 -->
    <div v-else class="grid">
      <div v-for="a in albums" :key="a.id" class="album-card">
        <div class="album-cover">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          <div class="album-badge" v-if="a.is_public">公开</div>
        </div>
        <div class="album-body">
          <h3 :title="a.name">{{ a.name }}</h3>
          <p class="desc">{{ a.description || '无描述' }}</p>
          <div class="album-meta">
            <span class="meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                <circle cx="8.5" cy="8.5" r="1.5"/>
                <polyline points="21 15 16 10 5 21"/>
              </svg>
              {{ a.image_count }} 张
            </span>
            <span class="meta-item" v-if="a.created_at">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
              {{ fmtDate(a.created_at) }}
            </span>
          </div>
          <div class="album-actions">
            <RouterLink :to="`/gallery?album=${a.id}`" class="btn btn-sm">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              查看
            </RouterLink>
            <button class="btn btn-sm btn-ghost" @click="openEdit(a)">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
              编辑
            </button>
            <button class="btn btn-sm btn-danger" @click="remove(a.id, a.name)">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <polyline points="3 6 5 6 21 6"/>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              </svg>
              删除
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建相册弹窗 -->
    <div v-if="showCreate" class="modal-mask" @click.self="showCreate = false">
      <form class="modal" @submit.prevent="create">
        <div class="modal-head">
          <h3>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            新建相册
          </h3>
        </div>
        <div class="modal-body">
          <label>
            <span class="label-text">相册名称 <span class="required">*</span></span>
            <input 
              class="input" 
              v-model="newName" 
              required 
              placeholder="例如:旅行照片"
              maxlength="100"
            />
          </label>
          <label>
            <span class="label-text">描述</span>
            <input 
              class="input" 
              v-model="newDesc" 
              placeholder="可选,简短描述这个相册"
              maxlength="500"
            />
          </label>
          <label class="checkbox-label">
            <input type="checkbox" v-model="newPublic" />
            <span class="checkbox-text">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              公开相册(游客可访问)
            </span>
          </label>
        </div>
        <div class="modal-foot">
          <button type="button" class="btn" @click="showCreate = false">取消</button>
          <button type="submit" class="btn btn-primary" :disabled="submitting">
            {{ submitting ? '创建中…' : '创建' }}
          </button>
        </div>
      </form>
    </div>

    <!-- 编辑相册弹窗 -->
    <div v-if="editingAlbum" class="modal-mask" @click.self="closeEdit">
      <form class="modal" @submit.prevent="updateAlbum">
        <div class="modal-head">
          <h3>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
            </svg>
            编辑相册
          </h3>
        </div>
        <div class="modal-body">
          <label>
            <span class="label-text">相册名称 <span class="required">*</span></span>
            <input 
              class="input" 
              v-model="editName" 
              required 
              maxlength="100"
            />
          </label>
          <label>
            <span class="label-text">描述</span>
            <input 
              class="input" 
              v-model="editDesc" 
              placeholder="可选"
              maxlength="500"
            />
          </label>
          <label class="checkbox-label">
            <input type="checkbox" v-model="editPublic" />
            <span class="checkbox-text">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              公开相册
            </span>
          </label>
        </div>
        <div class="modal-foot">
          <button type="button" class="btn" @click="closeEdit">取消</button>
          <button type="submit" class="btn btn-primary" :disabled="submitting">
            {{ submitting ? '保存中…' : '保存' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.albums-page {
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
  color: var(--fg-3);
}

.grid { 
  display: grid; 
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); 
  gap: 20px; 
}

.album-card {
  background: var(--bg-2); 
  border: 1px solid var(--border);
  border-radius: var(--radius-lg); 
  overflow: hidden;
  transition: all 0.25s ease;
}

.album-card:hover { 
  border-color: var(--border-2);
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.album-cover {
  aspect-ratio: 16 / 9;
  background: linear-gradient(135deg, var(--bg-3), var(--bg-4));
  display: flex; 
  align-items: center; 
  justify-content: center;
  color: var(--fg-3);
  position: relative;
}

.album-cover svg { 
  width: 48px; 
  height: 48px; 
  stroke: currentColor; 
  fill: none; 
  stroke-width: 1.5;
}

.album-badge {
  position: absolute;
  top: 10px;
  right: 10px;
  background: rgba(82, 196, 26, 0.9);
  color: white;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
  backdrop-filter: blur(4px);
}

.album-body { 
  padding: 16px; 
}

.album-body h3 { 
  font-size: 16px; 
  font-weight: 600; 
  margin: 0 0 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desc { 
  font-size: 13px; 
  color: var(--fg-3); 
  margin: 0 0 12px;
  overflow: hidden; 
  text-overflow: ellipsis; 
  white-space: nowrap;
  min-height: 20px;
}

.album-meta { 
  font-size: 12px; 
  color: var(--fg-3); 
  display: flex; 
  gap: 12px; 
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.album-actions { 
  display: flex; 
  gap: 8px;
  flex-wrap: wrap;
}

.album-actions .btn {
  display: flex;
  align-items: center;
  gap: 5px;
}

/* 模态框样式 */
.modal-head h3 {
  display: flex;
  align-items: center;
  gap: 8px;
}

.modal-body label { 
  display: flex; 
  flex-direction: column; 
  gap: 6px; 
  margin-bottom: 16px; 
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

.checkbox-label {
  flex-direction: row !important;
  align-items: center;
  gap: 10px !important;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  accent-color: var(--accent);
  cursor: pointer;
}

.checkbox-text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: var(--fg-2);
}

@media (max-width: 768px) {
  .head {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .grid {
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 16px;
  }
  
  .album-actions {
    justify-content: flex-start;
  }
}
</style>
