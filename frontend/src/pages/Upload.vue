<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { api } from '@/api'
import { useMetaStore } from '@/stores/meta'
import { useToastStore } from '@/stores/toast'

const metaStore = useMetaStore()
const toast = useToastStore()

type LinkFormat = 'url' | 'markdown' | 'html' | 'bbcode' | 'thumb'

interface UploadResult {
  key: string
  filename: string
  url: string
  thumb?: string
  markdown: string
  html: string
  bbcode: string
  width: number
  height: number
  size: number
}

interface UploadQueueItem {
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  error?: string
  result?: UploadResult
}

const stats = ref<any>({})
const dragging = ref(false)
const uploading = ref(false)
const uploadQueue = ref<UploadQueueItem[]>([])
const errors = ref<string[]>([])
const format = ref<LinkFormat>('url')
const copied = ref<string | null>(null)

const options = ref({
  compress: true,
  watermark: false,
  thumbnail: true,
  webp: false,
})

const fileInput = ref<HTMLInputElement>()
const dragCounter = ref(0)

onMounted(async () => {
  try {
    stats.value = await api.stats()
  } catch {}
  window.addEventListener('paste', onPaste)
})

onUnmounted(() => {
  window.removeEventListener('paste', onPaste)
})

function onPaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items
  if (!items) return
  
  const files: File[] = []
  for (const it of items) {
    if (it.kind === 'file') {
      const f = it.getAsFile()
      if (f && f.type.startsWith('image/')) {
        // 为粘贴的文件生成一个名称
        if (!f.name) {
          Object.defineProperty(f, 'name', {
            value: `pasted_${Date.now()}.png`,
            writable: false
          })
        }
        files.push(f)
      }
    }
  }
  
  if (files.length) {
    addToQueue(files)
  }
}

function pickFile() {
  fileInput.value?.click()
}

function onFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  if (target.files?.length) {
    addToQueue(Array.from(target.files))
    target.value = ''
  }
}

function onDragEnter(e: DragEvent) {
  e.preventDefault()
  dragCounter.value++
  dragging.value = true
}

function onDragLeave(e: DragEvent) {
  e.preventDefault()
  dragCounter.value--
  if (dragCounter.value === 0) {
    dragging.value = false
  }
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  dragging.value = false
  dragCounter.value = 0
  
  const files = Array.from(e.dataTransfer?.files || [])
  const imageFiles = files.filter(f => f.type.startsWith('image/'))
  
  if (imageFiles.length) {
    addToQueue(imageFiles)
  } else {
    errors.value = ['请上传图片文件']
  }
}

function addToQueue(files: File[]) {
  const newItems: UploadQueueItem[] = files.map(file => ({
    file,
    progress: 0,
    status: 'pending'
  }))
  
  uploadQueue.value = [...newItems, ...uploadQueue.value].slice(0, 50)
  processQueue()
}

async function processQueue() {
  if (uploading.value) return
  
  uploading.value = true
  
  try {
    const pendingItems = uploadQueue.value.filter(item => item.status === 'pending')
    
    for (const item of pendingItems) {
      item.status = 'uploading'
      item.progress = 0
      
      try {
        const res = await api.upload([item.file], p => {
          item.progress = p
        })
        
        if (res.uploaded.length > 0) {
          item.status = 'success'
          item.result = res.uploaded[0]
          item.progress = 100
        }
        
        if (res.errors?.length) {
          item.status = 'error'
          item.error = res.errors[0]
        }
      } catch (e: any) {
        item.status = 'error'
        item.error = e.message || '上传失败'
      }
    }
    
    // 更新统计
    try {
      stats.value = await api.stats()
    } catch {}
    
    // 清理错误
    setTimeout(() => {
      errors.value = []
    }, 5000)
    
  } finally {
    uploading.value = false
  }
}

function formatLink(item: UploadResult) {
  switch (format.value) {
    case 'markdown': return item.markdown
    case 'html': return item.html
    case 'bbcode': return item.bbcode
    case 'thumb': return item.thumb || item.url
    default: return item.url
  }
}

function copyAll() {
  const successItems = uploadQueue.value.filter(item => item.status === 'success' && item.result)
  const text = successItems.map(item => formatLink(item.result!)).join('\n')
  
  navigator.clipboard.writeText(text)
  copied.value = 'all'
  setTimeout(() => (copied.value = null), 2000)
}

function copyOne(item: UploadResult) {
  navigator.clipboard.writeText(formatLink(item))
  copied.value = item.key
  setTimeout(() => (copied.value = null), 2000)
}

function formatSize(b: number) {
  if (b < 1024) return b + ' B'
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  return (b / 1024 / 1024).toFixed(1) + ' MB'
}

function clearCompleted() {
  uploadQueue.value = uploadQueue.value.filter(item => item.status !== 'success')
}

function retryFailed() {
  const failedItems = uploadQueue.value.filter(item => item.status === 'error')
  failedItems.forEach(item => {
    item.status = 'pending'
    item.progress = 0
    item.error = undefined
  })
  processQueue()
}

const successCount = computed(() => uploadQueue.value.filter(item => item.status === 'success').length)
const errorCount = computed(() => uploadQueue.value.filter(item => item.status === 'error').length)
const totalCount = computed(() => uploadQueue.value.length)
const overallProgress = computed(() => {
  if (totalCount.value === 0) return 0
  const total = uploadQueue.value.reduce((sum, item) => sum + item.progress, 0)
  return Math.round(total / totalCount.value)
})

const formatOptions = [
  { key: 'url' as LinkFormat, label: 'URL' },
  { key: 'markdown' as LinkFormat, label: 'Markdown' },
  { key: 'html' as LinkFormat, label: 'HTML' },
  { key: 'bbcode' as LinkFormat, label: 'BBCode' },
  { key: 'thumb' as LinkFormat, label: '缩略图' },
]
</script>

<template>
  <div>
    <!-- 统计卡片 -->
    <div class="stats">
      <div class="stat">
        <div class="stat-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-label">总图片</div>
          <div class="stat-value">{{ stats.total || 0 }}</div>
          <div class="stat-delta" v-if="stats.today">+{{ stats.today }} 今日</div>
        </div>
      </div>
      
      <div class="stat">
        <div class="stat-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-label">已用空间</div>
          <div class="stat-value">{{ formatSize(stats.size_bytes || 0) }}</div>
        </div>
      </div>
      
      <div class="stat">
        <div class="stat-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-label">访问次数</div>
          <div class="stat-value">{{ stats.views || 0 }}</div>
        </div>
      </div>
      
      <div class="stat">
        <div class="stat-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-label">相册数量</div>
          <div class="stat-value">{{ stats.album_count || 0 }}</div>
        </div>
      </div>
    </div>

    <div class="upload-head">
      <div>
        <h2 class="section-title">上传图片</h2>
        <p class="section-sub">拖拽、粘贴或点击上传。支持 JPG / PNG / WebP / GIF</p>
      </div>
      <div v-if="metaStore.loaded" class="storage-tag" :title="'当前图片存放于 ' + metaStore.driverLabel()">
        <span class="dot"></span>
        存储: {{ metaStore.driverIcon() }} {{ metaStore.driverLabel() }}
      </div>
    </div>

    <!-- 上传区域 -->
    <div
      class="upload-zone"
      :class="{ dragging, uploading }"
      @dragenter="onDragEnter"
      @dragleave="onDragLeave"
      @dragover="onDragOver"
      @drop="onDrop"
      @click="pickFile"
    >
      <div class="upload-content">
        <svg class="upload-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="17 8 12 3 7 8"/>
          <line x1="12" y1="3" x2="12" y2="15"/>
        </svg>
        
        <template v-if="uploading">
          <h3>上传中… {{ overallProgress }}%</h3>
          <div class="progress-bar">
            <div :style="{ width: overallProgress + '%' }"></div>
          </div>
          <p>正在上传 {{ successCount }}/{{ totalCount }} 张图片</p>
        </template>
        
        <template v-else-if="dragging">
          <h3>松手上传</h3>
          <p>将文件拖到此处即可上传</p>
        </template>
        
        <template v-else>
          <h3>拖拽图片到此处,或点击选择</h3>
          <p>
            支持 <kbd>Ctrl</kbd>+<kbd>V</kbd> 粘贴 · 可同时选择多张
          </p>
        </template>
      </div>
      
      <input 
        ref="fileInput" 
        type="file" 
        multiple 
        accept="image/*" 
        @change="onFileChange" 
        hidden 
      />
    </div>

    <!-- 上传选项 -->
    <div class="options-row">
      <div 
        class="chip" 
        :class="{ active: options.compress }" 
        @click="options.compress = !options.compress"
      >
        <svg v-if="options.compress" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        自动压缩
      </div>
      <div 
        class="chip" 
        :class="{ active: options.watermark }" 
        @click="options.watermark = !options.watermark"
      >
        <svg v-if="options.watermark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        添加水印
      </div>
      <div
        class="chip"
        :class="{ active: options.thumbnail }"
        @click="options.thumbnail = !options.thumbnail"
      >
        <svg v-if="options.thumbnail" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        生成缩略图
      </div>
      <div
        class="chip"
        :class="{ active: options.webp }"
        @click="options.webp = !options.webp"
      >
        <svg v-if="options.webp" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        转 WebP
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="errors.length" class="error-box">
      <div class="error-header">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <span>上传错误</span>
      </div>
      <div v-for="(err, index) in errors" :key="index" class="error-item">{{ err }}</div>
    </div>

    <!-- 上传结果 -->
    <div v-if="totalCount > 0" class="results-card">
      <div class="results-header">
        <div class="results-tabs">
          <div 
            v-for="opt in formatOptions" 
            :key="opt.key"
            class="link-tab" 
            :class="{ active: format === opt.key }" 
            @click="format = opt.key"
          >
            {{ opt.label }}
          </div>
        </div>
        <div class="results-actions">
          <span class="results-count">
            成功 {{ successCount }} 张
            <span v-if="errorCount > 0" class="error-count">失败 {{ errorCount }} 张</span>
          </span>
          <button v-if="errorCount > 0" class="btn btn-sm" @click="retryFailed">重试失败</button>
          <button class="btn btn-sm" @click="clearCompleted">清空</button>
          <button class="btn btn-sm btn-primary" @click="copyAll">
            {{ copied === 'all' ? '已复制 ✓' : '复制全部' }}
          </button>
        </div>
      </div>
      
      <div class="results-list">
        <div v-for="item in uploadQueue.filter(i => i.status === 'success' && i.result)" :key="item.result!.key" class="link-row">
          <img :src="item.result!.thumb || item.result!.url" :alt="item.result!.filename" />
          <div class="link-info">
            <div class="fn" :title="item.result!.filename">{{ item.result!.filename }}</div>
            <code>{{ formatLink(item.result!) }}</code>
          </div>
          <div class="meta">
            {{ item.result!.width }}×{{ item.result!.height }}
            <span class="dot">·</span>
            {{ formatSize(item.result!.size) }}
          </div>
          <button class="btn btn-sm" @click="copyOne(item.result!)">
            {{ copied === item.result!.key ? '已复制 ✓' : '复制' }}
          </button>
        </div>
        
        <!-- 失败项 -->
        <div v-for="item in uploadQueue.filter(i => i.status === 'error')" :key="item.file.name" class="link-row error-row">
          <div class="error-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <circle cx="12" cy="12" r="10"/>
              <line x1="15" y1="9" x2="9" y2="15"/>
              <line x1="9" y1="9" x2="15" y2="15"/>
            </svg>
          </div>
          <div class="link-info">
            <div class="fn" :title="item.file.name">{{ item.file.name }}</div>
            <div class="error-msg">{{ item.error }}</div>
          </div>
          <button class="btn btn-sm" @click="item.status = 'pending'; processQueue()">重试</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.stats { 
  display: grid; 
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); 
  gap: 14px; 
  margin-bottom: 28px; 
}

.stat {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 14px;
  transition: all 0.2s ease;
}

.stat:hover {
  border-color: var(--border-2);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.stat-icon {
  width: 44px;
  height: 44px;
  background: var(--accent-soft);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
  flex-shrink: 0;
}

.stat-icon svg {
  width: 22px;
  height: 22px;
}

.stat-content {
  flex: 1;
  min-width: 0;
}

.stat-label {
  font-size: 12px;
  color: var(--fg-3);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--fg);
  line-height: 1;
}

.stat-delta {
  font-size: 11px;
  color: #52c41a;
  margin-top: 4px;
}

.upload-zone {
  border: 2px dashed var(--border-2);
  border-radius: var(--radius-xl);
  padding: 60px 24px;
  text-align: center;
  background: var(--bg-2);
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.upload-zone:hover, 
.upload-zone.dragging { 
  border-color: var(--accent); 
  background: var(--accent-soft);
  transform: scale(1.01);
}

.upload-zone.uploading { 
  cursor: progress;
  border-style: solid;
}

.upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.upload-icon { 
  width: 48px; 
  height: 48px; 
  color: var(--fg-3); 
  stroke: currentColor; 
  fill: none;
  transition: all 0.3s ease;
}

.upload-zone:hover .upload-icon,
.upload-zone.dragging .upload-icon {
  color: var(--accent);
  transform: translateY(-4px);
}

.upload-zone h3 { 
  font-size: 16px; 
  font-weight: 600; 
  margin: 0;
}

.upload-zone p { 
  color: var(--fg-3); 
  font-size: 13px;
  margin: 0;
}

.upload-zone kbd { 
  background: var(--bg-3); 
  border: 1px solid var(--border-2); 
  padding: 2px 6px; 
  border-radius: 4px; 
  font-size: 12px; 
  font-family: var(--font-mono);
}

.progress-bar { 
  width: 100%;
  max-width: 400px;
  height: 4px; 
  background: var(--bg-3); 
  border-radius: 2px; 
  overflow: hidden;
}

.progress-bar div { 
  height: 100%; 
  background: linear-gradient(90deg, var(--accent), var(--accent-hover, var(--accent)));
  transition: width 0.2s ease;
  border-radius: 2px;
}

.options-row { 
  display: flex; 
  gap: 10px; 
  margin-top: 20px; 
  flex-wrap: wrap; 
}

.chip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: 20px;
  font-size: 13px;
  color: var(--fg-2);
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
}

.chip:hover {
  border-color: var(--accent);
  color: var(--fg);
}

.chip.active {
  background: var(--accent-soft);
  border-color: var(--accent);
  color: var(--accent);
}

.chip svg {
  width: 14px;
  height: 14px;
}

.error-box {
  margin-top: 20px;
  padding: 14px 16px;
  background: rgba(235, 87, 87, 0.08);
  border: 1px solid rgba(235, 87, 87, 0.3);
  border-radius: var(--radius-lg);
  color: var(--danger);
  font-size: 13px;
  animation: shake 0.4s ease;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-8px); }
  75% { transform: translateX(8px); }
}

.error-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  margin-bottom: 8px;
}

.error-item {
  padding: 4px 0;
}

.results-card {
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  margin-top: 28px;
  overflow: hidden;
}

.results-header {
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.results-tabs {
  display: flex;
  gap: 4px;
  align-items: center;
}

.link-tab {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  color: var(--fg-3);
  cursor: pointer;
  transition: all 0.2s ease;
  font-weight: 500;
}

.link-tab:hover {
  color: var(--fg);
}

.link-tab.active {
  background: var(--bg-3);
  color: var(--fg);
}

.results-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.results-count {
  font-size: 13px;
  color: var(--fg-2);
}

.error-count {
  color: var(--danger);
  margin-left: 8px;
}

.results-list {
  max-height: 500px;
  overflow-y: auto;
}

.link-row {
  display: grid;
  grid-template-columns: 48px 1fr auto auto;
  gap: 14px;
  align-items: center;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  transition: background 0.2s ease;
}

.link-row:hover {
  background: var(--bg-3);
}

.link-row img { 
  width: 48px; 
  height: 48px; 
  object-fit: cover; 
  border-radius: var(--radius); 
  background: var(--bg-3);
}

.link-info { 
  min-width: 0; 
}

.link-info .fn { 
  font-size: 13px; 
  color: var(--fg-3); 
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.link-info code { 
  display: block; 
  font-family: var(--font-mono); 
  font-size: 12px; 
  color: var(--fg); 
  overflow: hidden; 
  text-overflow: ellipsis; 
  white-space: nowrap;
  background: var(--bg);
  padding: 6px 8px;
  border-radius: 4px;
}

.link-row .meta { 
  font-size: 12px; 
  color: var(--fg-3); 
  font-family: var(--font-mono);
  display: flex;
  align-items: center;
  gap: 6px;
}

.dot {
  color: var(--fg-3);
}

.error-row {
  background: rgba(235, 87, 87, 0.05);
}

.error-icon {
  color: var(--danger);
  display: flex;
  align-items: center;
  justify-content: center;
}

.error-msg {
  font-size: 12px;
  color: var(--danger);
  margin-top: 4px;
}

@media (max-width: 768px) {
  .stats {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .upload-zone {
    padding: 40px 20px;
  }
  
  .results-header {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .results-actions {
    width: 100%;
    justify-content: space-between;
  }
  
  .link-row {
    grid-template-columns: 40px 1fr auto;
    gap: 10px;
  }
  
  .link-row .meta {
    display: none;
  }
}
</style>
