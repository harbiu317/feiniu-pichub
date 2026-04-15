<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { api } from '@/api'
import { useToastStore } from '@/stores/toast'
import { useStatsStore } from '@/stores/stats'
const toast = useToastStore()
const statsStore = useStatsStore()

const items = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = 48
const keyword = ref('')
const loading = ref(false)
const selected = ref(new Set<number>())
const albums = ref<any[]>([])
const filterAlbum = ref<number | ''>('')
const preview = ref<any>(null)
const previewIndex = ref(0)
const isSelectMode = ref(false)
const isLoadingMore = ref(false)

// 使用懒加载图片
const observer = ref<IntersectionObserver | null>(null)
const imageRefs = ref<Map<number, HTMLImageElement>>(new Map())

onMounted(async () => {
  await load()
  try { 
    albums.value = await api.listAlbums() 
  } catch {}
  
  // 初始化 IntersectionObserver 用于图片懒加载
  observer.value = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const img = entry.target as HTMLImageElement
        if (img.dataset.src) {
          img.src = img.dataset.src
          observer.value?.unobserve(img)
        }
      }
    })
  }, {
    rootMargin: '50px'
  })
})

watch([keyword, filterAlbum, page], () => {
  load()
})

async function load() {
  loading.value = true
  try {
    const res = await api.listImages({
      page: page.value,
      size,
      q: keyword.value || undefined,
      album: filterAlbum.value ? Number(filterAlbum.value) : undefined,
    })
    items.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

function toggleSelect(id: number) {
  if (selected.value.has(id)) {
    selected.value.delete(id)
  } else {
    selected.value.add(id)
  }
  selected.value = new Set(selected.value)
}

function selectAll() {
  if (selected.value.size === items.value.length) {
    selected.value = new Set()
  } else {
    selected.value = new Set(items.value.map(item => item.id))
  }
}

function clearSelection() {
  selected.value = new Set()
  isSelectMode.value = false
}

async function deleteSelected() {
  if (!selected.value.size) return
  if (!confirm(`确定删除 ${selected.value.size} 张图片?此操作不可恢复。`)) return
  
  loading.value = true
  try {
    await api.batchDelete(Array.from(selected.value))
    clearSelection()
    await load()
    statsStore.load()
  } catch (e: any) {
    toast.error(e.message || '删除失败')
  } finally {
    loading.value = false
  }
}

async function moveSelected() {
  if (!selected.value.size) return
  const albumId = prompt('移到哪个相册?输入相册 ID(0 = 默认/移出)')
  if (albumId === null) return
  
  loading.value = true
  try {
    await api.moveImages(Array.from(selected.value), Number(albumId))
    clearSelection()
    await load()
  } catch (e: any) {
    toast.error(e.message || '移动失败')
  } finally {
    loading.value = false
  }
}

function formatSize(b: number) {
  if (b < 1024) return b + ' B'
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  return (b / 1024 / 1024).toFixed(1) + ' MB'
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

function previewImage(item: any, index: number) {
  if (isSelectMode.value) {
    toggleSelect(item.id)
    return
  }
  preview.value = item
  previewIndex.value = index
}

function navigatePreview(direction: 'prev' | 'next') {
  if (!items.value.length) return
  
  if (direction === 'prev') {
    previewIndex.value = (previewIndex.value - 1 + items.value.length) % items.value.length
  } else {
    previewIndex.value = (previewIndex.value + 1) % items.value.length
  }
  
  preview.value = items.value[previewIndex.value]
}

function copyURL() {
  if (preview.value) {
    navigator.clipboard.writeText(preview.value.url)
    toast.success('已复制链接')
  }
}

function closePreview() {
  preview.value = null
}

// 键盘导航
function handleKeydown(e: KeyboardEvent) {
  if (!preview.value) return
  
  switch (e.key) {
    case 'Escape':
      closePreview()
      break
    case 'ArrowLeft':
      navigatePreview('prev')
      break
    case 'ArrowRight':
      navigatePreview('next')
      break
    case 'ArrowUp':
      e.preventDefault()
      if (page.value > 1) {
        page.value--
        closePreview()
      }
      break
    case 'ArrowDown':
      e.preventDefault()
      if (page.value < totalPages.value) {
        page.value++
        closePreview()
      }
      break
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

import { onUnmounted } from 'vue'
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  observer.value?.disconnect()
})
</script>

<template>
  <div class="gallery-head">
    <div>
      <h2 class="section-title">图库</h2>
      <p class="section-sub">共 {{ total }} 张图片</p>
    </div>
    <div class="filter-bar">
      <div class="search-input">
        <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <path d="m21 21-4.35-4.35"/>
        </svg>
        <input class="input" v-model="keyword" placeholder="搜索文件名..." />
      </div>
      <select class="input" v-model="filterAlbum">
        <option value="">所有相册</option>
        <option v-for="a in albums" :key="a.id" :value="a.id">{{ a.name }}</option>
      </select>
      <button 
        class="btn btn-sm" 
        :class="{ 'btn-active': isSelectMode }"
        @click="isSelectMode = !isSelectMode"
      >
        {{ isSelectMode ? '完成' : '选择' }}
      </button>
    </div>
  </div>

  <!-- 批量操作工具栏 -->
  <div v-if="selected.size" class="toolbar">
    <span>已选择 <strong>{{ selected.size }}</strong> 张</span>
    <button class="btn btn-sm" @click="selectAll">
      {{ selected.size === items.length ? '取消全选' : '全选' }}
    </button>
    <button class="btn btn-sm" @click="clearSelection">取消选择</button>
    <button class="btn btn-sm" @click="moveSelected">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
        <path d="M5 12h14M12 5l7 7-7 7"/>
      </svg>
      移动到相册
    </button>
    <button class="btn btn-sm btn-danger" @click="deleteSelected">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
        <polyline points="3 6 5 6 21 6"/>
        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
      </svg>
      删除
    </button>
  </div>

  <!-- 加载状态 -->
  <div v-if="loading" class="loading">
    <div class="loading-spinner"></div>
    <span>加载中…</span>
  </div>
  
  <!-- 空状态 -->
  <div v-else-if="!items.length" class="empty">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
      <circle cx="8.5" cy="8.5" r="1.5"/>
      <polyline points="21 15 16 10 5 21"/>
    </svg>
    <p>暂无图片</p>
    <RouterLink to="/upload" class="btn btn-primary">去上传</RouterLink>
  </div>

  <!-- 图片网格 -->
  <div v-else class="grid">
    <div 
      v-for="(item, index) in items" 
      :key="item.id" 
      class="tile" 
      :class="{ selected: selected.has(item.id) }"
      @click="previewImage(item, index)"
    >
      <img 
        :data-src="item.thumb" 
        :alt="item.filename"
        loading="lazy"
      />
      <div class="pick" @click.stop="toggleSelect(item.id)">
        <svg v-if="selected.has(item.id)" viewBox="0 0 24 24">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
      </div>
      <div class="meta">
        <span class="name" :title="item.filename">{{ item.filename }}</span>
        <span class="size">{{ formatSize(item.size) }}</span>
      </div>
    </div>
  </div>

  <!-- 分页 -->
  <div v-if="totalPages > 1" class="pager">
    <button class="btn btn-sm" :disabled="page <= 1" @click="page--">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
        <polyline points="15 18 9 12 15 6"/>
      </svg>
      上一页
    </button>
    <span class="pager-info">{{ page }} / {{ totalPages }}</span>
    <button class="btn btn-sm" :disabled="page >= totalPages" @click="page++">
      下一页
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
        <polyline points="9 18 15 12 9 6"/>
      </svg>
    </button>
  </div>

  <!-- 图片预览模态框 -->
  <div v-if="preview" class="modal-mask" @click.self="closePreview">
    <div class="preview-container">
      <button class="preview-close" @click="closePreview">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
      
      <button 
        class="preview-nav preview-nav-prev" 
        @click="navigatePreview('prev')"
        :disabled="!items.length"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
      </button>
      
      <div class="preview">
        <img :src="preview.url" :alt="preview.filename" />
        <div class="preview-meta">
          <h3 :title="preview.filename">{{ preview.filename }}</h3>
          <p>{{ preview.width }}×{{ preview.height }} · {{ formatSize(preview.size) }}</p>
          <code>{{ preview.url }}</code>
          <div class="preview-actions">
            <button class="btn btn-sm" @click="copyURL">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
              </svg>
              复制链接
            </button>
            <a :href="preview.url" target="_blank" class="btn btn-sm">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                <polyline points="15 3 21 3 21 9"/>
                <line x1="10" y1="14" x2="21" y2="3"/>
              </svg>
              打开
            </a>
            <button class="btn btn-sm btn-ghost" @click="closePreview">关闭</button>
          </div>
        </div>
      </div>
      
      <button 
        class="preview-nav preview-nav-next" 
        @click="navigatePreview('next')"
        :disabled="!items.length"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6"/>
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.gallery-head { 
  display: flex; 
  align-items: flex-end; 
  justify-content: space-between; 
  margin-bottom: 16px; 
  gap: 16px; 
  flex-wrap: wrap; 
}

.filter-bar { 
  display: flex; 
  gap: 8px; 
  align-items: center;
}

.filter-bar .input { 
  width: 200px; 
}

.search-input {
  position: relative;
  display: flex;
  align-items: center;
}

.search-input .search-icon {
  position: absolute;
  left: 10px;
  width: 16px;
  height: 16px;
  color: var(--fg-3);
  pointer-events: none;
}

.search-input .input {
  padding-left: 34px;
}

.toolbar {
  display: flex; 
  align-items: center; 
  gap: 8px;
  padding: 10px 14px; 
  margin-bottom: 16px;
  background: var(--bg-2);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius);
  font-size: 13px;
  animation: slideDown 0.2s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.toolbar > span { 
  color: var(--fg-2); 
  margin-right: auto; 
}

.toolbar strong {
  color: var(--fg);
  font-weight: 600;
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
  width: 64px;
  height: 64px;
  color: var(--fg-3);
}

.empty p {
  font-size: 15px;
  margin: 0;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 14px;
}

.tile {
  background: var(--bg-2);
  border: 2px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  position: relative;
  aspect-ratio: 1;
  cursor: pointer;
  transition: all 0.2s ease;
}

.tile:hover { 
  border-color: var(--border-2);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.tile.selected { 
  border-color: var(--accent);
  box-shadow: 0 0 0 2px var(--accent-soft);
}

.tile img { 
  width: 100%; 
  height: 100%; 
  object-fit: cover; 
  display: block;
  background: var(--bg-3);
}

.tile .pick {
  position: absolute; 
  top: 8px; 
  left: 8px;
  width: 24px; 
  height: 24px; 
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  display: flex; 
  align-items: center; 
  justify-content: center;
  color: white;
  opacity: 0;
  transition: all 0.2s ease;
}

.tile:hover .pick,
.tile.selected .pick {
  opacity: 1;
}

.tile .pick svg { 
  width: 14px; 
  height: 14px; 
  stroke: currentColor; 
  fill: none; 
  stroke-width: 2.5; 
}

.tile.selected .pick { 
  background: var(--accent); 
  border-color: var(--accent);
  opacity: 1;
}

.tile .meta {
  position: absolute; 
  left: 0; 
  right: 0; 
  bottom: 0;
  padding: 10px;
  background: linear-gradient(to top, rgba(0,0,0,0.9), transparent);
  font-size: 11px;
  color: var(--fg-2);
  display: flex; 
  justify-content: space-between;
  align-items: flex-end;
  gap: 8px;
}

.tile .name { 
  overflow: hidden; 
  text-overflow: ellipsis; 
  white-space: nowrap; 
  max-width: 60%;
  font-weight: 500;
}

.tile .size { 
  color: var(--fg-3); 
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.pager { 
  display: flex; 
  justify-content: center; 
  align-items: center; 
  gap: 16px; 
  margin-top: 24px; 
  color: var(--fg-2); 
  font-size: 13px; 
}

.pager-info {
  font-weight: 500;
  min-width: 80px;
  text-align: center;
}

/* 预览模态框 */
.preview-container {
  position: relative;
  max-width: 95vw;
  max-height: 90vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-close {
  position: absolute;
  top: -40px;
  right: 0;
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--fg);
  transition: all 0.2s;
}

.preview-close:hover {
  background: var(--bg-3);
}

.preview-close svg {
  width: 18px;
  height: 18px;
}

.preview-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: var(--bg-2);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--fg);
  transition: all 0.2s;
}

.preview-nav:hover:not(:disabled) {
  background: var(--bg-3);
  border-color: var(--accent);
}

.preview-nav:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.preview-nav svg {
  width: 20px;
  height: 20px;
}

.preview-nav-prev {
  left: -60px;
}

.preview-nav-next {
  right: -60px;
}

.preview {
  display: grid;
  grid-template-columns: 1fr 320px;
  max-width: 90vw; 
  max-height: 90vh;
  background: var(--bg-2); 
  border: 1px solid var(--border);
  border-radius: var(--radius-xl); 
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  animation: zoomIn 0.2s ease-out;
}

@keyframes zoomIn {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.preview img { 
  width: 100%; 
  height: 100%; 
  object-fit: contain; 
  background: var(--bg);
}

.preview-meta { 
  padding: 24px; 
  display: flex; 
  flex-direction: column; 
  gap: 12px; 
}

.preview-meta h3 { 
  font-size: 16px; 
  font-weight: 600; 
  word-break: break-all;
  margin: 0;
}

.preview-meta p { 
  color: var(--fg-3); 
  font-size: 13px;
  margin: 0;
}

.preview-meta code { 
  background: var(--bg-3); 
  padding: 10px; 
  border-radius: var(--radius); 
  font-family: var(--font-mono); 
  font-size: 12px; 
  word-break: break-all;
  border: 1px solid var(--border);
}

.preview-actions { 
  display: flex; 
  gap: 8px; 
  margin-top: auto;
  flex-wrap: wrap;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .gallery-head {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .filter-bar {
    width: 100%;
    flex-wrap: wrap;
  }
  
  .filter-bar .input {
    flex: 1;
    min-width: 150px;
  }
  
  .grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
  }
  
  .preview {
    grid-template-columns: 1fr;
    max-height: 95vh;
  }
  
  .preview-meta {
    max-height: 40vh;
    overflow-y: auto;
  }
  
  .preview-nav-prev {
    left: 10px;
  }
  
  .preview-nav-next {
    right: 10px;
  }
  
  .preview-nav {
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
  }
}
</style>
