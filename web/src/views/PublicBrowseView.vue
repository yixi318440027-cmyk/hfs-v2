<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import api from '../api'
import { refreshLucide } from '../utils/lucide'

interface FileItem {
  name: string
  size: number
  sizeHuman: string
  modTime: string
  isDir: boolean
  mime: string
  downloads: number
}

// State
const path = ref('')
const files = ref<FileItem[]>([])
const loading = ref(false)
const sortKey = ref<'name' | 'size' | 'modTime'>('name')
const sortDir = ref<'asc' | 'desc'>('asc')
const searchQuery = ref('')
const viewMode = ref<'list' | 'grid'>('list')
const publicRoots = ref<{ name: string }[]>([])
const rootNames = ref<string[]>([])

// Computed
const breadcrumbs = computed(() => {
  if (!path.value) return []
  const parts = path.value.split('/').filter(Boolean)
  return parts.map((part, index) => ({
    name: part,
    path: '/' + parts.slice(0, index + 1).join('/'),
    isLast: index === parts.length - 1,
  }))
})

const sortedFiles = computed(() => {
  const list = [...files.value]
  list.sort((a, b) => {
    if (a.isDir && !b.isDir) return -1
    if (!a.isDir && b.isDir) return 1
    let cmp = 0
    switch (sortKey.value) {
      case 'name': cmp = a.name.localeCompare(b.name); break
      case 'size': cmp = a.size - b.size; break
      case 'modTime': cmp = new Date(a.modTime).getTime() - new Date(b.modTime).getTime(); break
    }
    return sortDir.value === 'asc' ? cmp : -cmp
  })
  return list
})

const filteredFiles = computed(() => {
  if (!searchQuery.value.trim()) return sortedFiles.value
  const q = searchQuery.value.toLowerCase()
  return sortedFiles.value.filter(f => f.name.toLowerCase().includes(q))
})

const searchResultCount = computed(() => {
  if (!searchQuery.value.trim()) return 0
  return filteredFiles.value.length
})

// Methods
async function loadRoots() {
  try {
    const res = await api.get('/public/files/roots')
    if (res.data.ok) {
      publicRoots.value = res.data.data || []
      rootNames.value = publicRoots.value.map(r => r.name)
      if (publicRoots.value.length > 0 && !path.value) {
        path.value = '/' + publicRoots.value[0].name
        loadFiles()
      }
    }
  } catch {
    publicRoots.value = []
  }
  nextTick(() => refreshLucide())
}

async function loadFiles() {
  if (!path.value) return
  loading.value = true
  try {
    const res = await api.get('/public/files/list', { params: { path: path.value } })
    if (res.data.ok) {
      files.value = res.data.data.files || []
    }
  } catch {
    files.value = []
  } finally {
    loading.value = false
    nextTick(() => refreshLucide())
  }
}

function navigateTo(targetPath: string) {
  path.value = targetPath
  searchQuery.value = ''
  loadFiles()
}

async function handleDownload(file: FileItem) {
  const url = `/api/public/files/download?path=${encodeURIComponent(path.value + '/' + file.name)}`
  try {
    const res = await fetch(url)
    if (!res.ok) {
      const data = await res.json().catch(() => ({ error: 'download failed' }))
      alert('下载失败: ' + ((data as any).error || res.statusText))
      return
    }
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = file.name
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(blobUrl)
  } catch (err: any) {
    alert('下载失败: ' + (err.message || 'unknown'))
  }
}

function toggleSort(key: 'name' | 'size' | 'modTime') {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDir.value = 'asc'
  }
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

function formatTime(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN')
}

onMounted(() => {
  loadRoots()
})
</script>

<template>
  <div class="public-browse">
    <!-- Page header -->
    <div class="page-header">
      <h1 class="page-title">公开文件</h1>
      <p class="page-desc">浏览和下载公开分享的文件，无需登录。</p>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="view-toggle">
        <button
          class="btn-icon"
          :class="{ active: viewMode === 'list' }"
          title="列表视图"
          @click="viewMode = 'list'"
        >
          <i data-lucide="list" style="width:14px;height:14px"></i>
        </button>
        <button
          class="btn-icon"
          :class="{ active: viewMode === 'grid' }"
          title="网格视图"
          @click="viewMode = 'grid'"
        >
          <i data-lucide="grid-3x3" style="width:14px;height:14px"></i>
        </button>
      </div>
      <div class="search-box">
        <i data-lucide="search" style="width:14px;height:14px" class="search-icon"></i>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索文件..."
          class="search-input"
        />
        <button v-if="searchQuery" class="search-clear" @click="searchQuery = ''">
          <i data-lucide="x" style="width:14px;height:14px"></i>
        </button>
      </div>
    </div>

    <!-- Breadcrumb -->
    <div class="breadcrumb" v-if="breadcrumbs.length > 0">
      <template v-for="(crumb, index) in breadcrumbs" :key="crumb.path">
        <span v-if="index > 0" class="bc-sep">
          <i data-lucide="chevron-right" style="width:12px;height:12px"></i>
        </span>
        <a
          v-if="!crumb.isLast"
          class="bc-link"
          href="#"
          @click.prevent="navigateTo(crumb.path)"
        >
          {{ crumb.name }}
        </a>
        <span v-else class="bc-current">{{ crumb.name }}</span>
      </template>
    </div>

    <!-- Empty state for no public roots -->
    <div v-if="publicRoots.length === 0 && !loading" class="empty-state">
      <i data-lucide="folder-x" style="width:48px;height:48px;color:var(--c-text-muted)"></i>
      <div class="empty-text">暂无公开文件</div>
      <div class="empty-hint">管理员尚未设置任何公开可访问的文件目录。</div>
    </div>

    <!-- File table (list view) -->
    <div v-if="viewMode === 'list' && publicRoots.length > 0" class="file-table-wrapper">
      <div v-if="searchQuery.trim()" class="search-info">
        <i data-lucide="search" style="width:13px;height:13px"></i>
        找到 {{ searchResultCount }} 个结果
      </div>

      <div v-if="loading" class="skeleton-list">
        <div v-for="i in 6" :key="i" class="skeleton-row">
          <div class="skeleton-cell" style="flex:1"></div>
          <div class="skeleton-cell" style="width:80px"></div>
          <div class="skeleton-cell" style="width:140px"></div>
          <div class="skeleton-cell" style="width:80px"></div>
        </div>
      </div>

      <table v-else class="file-table">
        <thead>
          <tr>
            <th class="col-name sortable" @click="toggleSort('name')">
              名称
              <span v-if="sortKey === 'name'" class="sort-arrow">
                <i v-if="sortDir === 'asc'" data-lucide="chevron-up" style="width:12px;height:12px"></i>
                <i v-else data-lucide="chevron-down" style="width:12px;height:12px"></i>
              </span>
            </th>
            <th class="col-size sortable" @click="toggleSort('size')">
              大小
              <span v-if="sortKey === 'size'" class="sort-arrow">
                <i v-if="sortDir === 'asc'" data-lucide="chevron-up" style="width:12px;height:12px"></i>
                <i v-else data-lucide="chevron-down" style="width:12px;height:12px"></i>
              </span>
            </th>
            <th class="col-downloads">下载</th>
            <th class="col-time sortable" @click="toggleSort('modTime')">
              修改时间
              <span v-if="sortKey === 'modTime'" class="sort-arrow">
                <i v-if="sortDir === 'asc'" data-lucide="chevron-up" style="width:12px;height:12px"></i>
                <i v-else data-lucide="chevron-down" style="width:12px;height:12px"></i>
              </span>
            </th>
            <th class="col-actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="filteredFiles.length === 0">
            <td colspan="5">
              <div class="empty-state">
                <i data-lucide="folder-open" style="width:36px;height:36px;color:var(--c-text-muted)"></i>
                <div class="empty-text">目录为空</div>
              </div>
            </td>
          </tr>
          <tr
            v-for="(file, index) in filteredFiles"
            :key="file.name"
            class="file-row"
            :class="{ 'row-stripe': index % 2 === 1 }"
          >
            <td class="col-name">
              <i
                v-if="file.isDir"
                data-lucide="folder"
                class="file-icon file-icon-folder clickable"
                @click="navigateTo(path + '/' + file.name)"
              ></i>
              <i
                v-else
                data-lucide="file-text"
                class="file-icon file-icon-file"
              ></i>
              <span
                class="file-name"
                :class="{ clickable: file.isDir }"
                @click="file.isDir && navigateTo(path + '/' + file.name)"
              >
                {{ file.name }}
              </span>
            </td>
            <td class="col-size">
              {{ file.isDir ? '--' : (file.sizeHuman || formatSize(file.size)) }}
            </td>
            <td class="col-downloads">{{ file.isDir ? '--' : (file.downloads || 0) }}</td>
            <td class="col-time">{{ formatTime(file.modTime) }}</td>
            <td class="col-actions">
              <button
                v-if="!file.isDir"
                class="btn-dl"
                @click="handleDownload(file)"
              >
                <i data-lucide="download" style="width:12px;height:12px"></i>
                下载
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- File grid (grid view) -->
    <div v-else-if="viewMode === 'grid' && publicRoots.length > 0" class="file-grid-wrapper">
      <div v-if="searchQuery.trim()" class="search-info">
        <i data-lucide="search" style="width:13px;height:13px"></i>
        找到 {{ searchResultCount }} 个结果
      </div>
      <div v-if="loading" class="skeleton-grid">
        <div v-for="i in 8" :key="i" class="skeleton-card"></div>
      </div>
      <div v-else-if="filteredFiles.length === 0" class="empty-state">
        <i data-lucide="folder-open" style="width:36px;height:36px;color:var(--c-text-muted)"></i>
        <div class="empty-text">目录为空</div>
      </div>
      <div v-else class="file-grid">
        <div
          v-for="file in filteredFiles"
          :key="file.name"
          class="file-card"
          @dblclick="file.isDir && navigateTo(path + '/' + file.name)"
        >
          <div class="card-icon-wrap">
            <i v-if="file.isDir" data-lucide="folder-open" style="width:32px;height:32px;color:#F59E0B"></i>
            <i v-else data-lucide="file-text" style="width:32px;height:32px;color:#6B7280"></i>
          </div>
          <div class="card-name" :title="file.name">{{ file.name }}</div>
          <div class="card-meta">
            <span v-if="!file.isDir">{{ file.sizeHuman || formatSize(file.size) }}</span>
            <span v-else>文件夹</span>
          </div>
          <div class="card-actions">
            <button
              v-if="!file.isDir"
              class="btn btn-sm"
              @click="handleDownload(file)"
            >
              <i data-lucide="download" style="width:12px;height:12px"></i>
              下载
            </button>
            <button
              v-if="file.isDir"
              class="btn btn-sm"
              @click="navigateTo(path + '/' + file.name)"
            >
              <i data-lucide="folder-open" style="width:12px;height:12px"></i>
              打开
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.public-browse {
  padding: 24px 28px;
  height: 100%;
  overflow-y: auto;
}

.page-header {
  margin-bottom: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--c-text);
  margin: 0;
}

.page-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--c-text-muted);
}

/* Toolbar */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
}

.view-toggle {
  display: flex;
  border: 1px solid var(--c-border);
  border-radius: 4px;
  overflow: hidden;
}

.view-toggle .btn-icon {
  border: none;
  border-radius: 0;
}

.view-toggle .btn-icon + .btn-icon {
  border-left: 1px solid var(--c-border);
}

/* Search box — borderless, bottom-line on focus */
.search-box {
  display: flex;
  align-items: center;
  height: 32px;
  background: var(--c-white);
  border: 1px solid transparent;
  border-radius: 4px;
  padding: 0 8px;
  transition: border-color 200ms;
  min-width: 200px;
  position: relative;
}

.search-box::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 8px;
  right: 8px;
  height: 1px;
  background: transparent;
  transition: background 200ms;
}

.search-box:focus-within {
  border-color: #D1D5DB;
}

.search-box:focus-within::after {
  background: #D1D5DB;
}

.search-icon {
  color: var(--c-text-muted);
  flex-shrink: 0;
}

.search-input {
  border: none;
  outline: none;
  height: 100%;
  padding: 0 4px;
  font-size: 13px;
  color: var(--c-text);
  flex: 1;
  min-width: 0;
  background: transparent;
  font-family: inherit;
}

.search-input::placeholder {
  color: var(--c-text-placeholder);
}

.search-clear {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: none;
  background: none;
  color: var(--c-text-muted);
  cursor: pointer;
  flex-shrink: 0;
}

.search-clear:hover {
  color: var(--c-text);
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0;
  height: 28px;
  margin-bottom: 12px;
  font-size: 13px;
}

.bc-sep {
  display: flex;
  align-items: center;
  color: var(--c-text-muted);
  margin: 0 2px;
}

.bc-link {
  color: var(--c-primary);
  text-decoration: none;
  padding: 2px 4px;
  border-radius: 2px;
  transition: background 150ms;
}

.bc-link:hover {
  background: var(--c-primary-light);
  text-decoration: none;
}

.bc-current {
  color: var(--c-text);
  font-weight: 500;
  padding: 2px 4px;
}

/* Search info */
.search-info {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--c-text-muted);
  border-bottom: 1px solid var(--c-border);
}

/* File table */
.file-table-wrapper {
  background: var(--c-white);
  border: 1px solid var(--c-border);
  border-radius: 4px;
  overflow: hidden;
}

.file-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.file-table th {
  background: var(--c-header-bg);
  padding: 0 12px;
  height: 36px;
  text-align: left;
  font-weight: 500;
  font-size: 12px;
  color: var(--c-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.025em;
  border-bottom: 1px solid var(--c-border);
  white-space: nowrap;
  user-select: none;
}

.file-table th.sortable {
  cursor: pointer;
}

.file-table th.sortable:hover {
  background: var(--c-primary-light);
  color: var(--c-primary);
}

.sort-arrow {
  display: inline-flex;
  vertical-align: middle;
  margin-left: 2px;
  color: var(--c-primary);
}

.file-table td {
  padding: 0 12px;
  height: 40px;
  border-bottom: 1px solid var(--c-border-light);
  color: var(--c-text);
}

.col-name { min-width: 200px; }
.col-size { width: 90px; white-space: nowrap; font-size: 12px; color: var(--c-text-muted); }
.col-downloads { width: 60px; text-align: center; white-space: nowrap; font-size: 12px; color: var(--c-text-muted); }
.col-time { width: 170px; white-space: nowrap; font-size: 12px; color: var(--c-text-muted); }
.col-actions { width: 100px; white-space: nowrap; }

.file-row {
  cursor: default;
  user-select: none;
  transition: background 100ms;
}

.file-row:hover {
  background: var(--c-row-hover);
}

.row-stripe {
  background: var(--c-row-stripe);
}

.row-stripe:hover {
  background: var(--c-row-hover);
}

.file-icon {
  margin-right: 8px;
  vertical-align: middle;
}

.file-icon-folder { color: #F59E0B; }
.file-icon-file { color: #6B7280; }

.clickable {
  cursor: pointer;
  color: var(--c-primary);
}

.clickable:hover {
  text-decoration: underline;
}

.file-name { vertical-align: middle; }

/* Download button */
.btn-dl {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--c-primary);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  transition: all 150ms ease-in-out;
  white-space: nowrap;
}

.btn-dl:hover {
  background: var(--c-primary-light);
  border-color: var(--c-primary-border);
  color: var(--c-primary-hover);
}

/* Empty state */
.empty-state {
  text-align: center;
  padding: 48px 16px;
}

.empty-state i {
  display: block;
  margin: 0 auto 12px;
}

.empty-text {
  font-size: 15px;
  color: var(--c-text-secondary);
  margin-bottom: 4px;
}

.empty-hint {
  font-size: 12px;
  color: var(--c-text-muted);
}

/* Skeleton */
.skeleton-list { padding: 12px; }
.skeleton-row { display: flex; align-items: center; gap: 12px; height: 40px; padding: 0 12px; }
.skeleton-cell { height: 12px; border-radius: 4px; background: var(--c-bg); animation: shimmer 1.5s ease-in-out infinite; }
.skeleton-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 12px; padding: 16px; }
.skeleton-card { height: 120px; border-radius: 4px; background: var(--c-bg); animation: shimmer 1.5s ease-in-out infinite; }

@keyframes shimmer {
  0% { opacity: 0.6; }
  50% { opacity: 1; }
  100% { opacity: 0.6; }
}

/* Grid view */
.file-grid-wrapper {
  background: var(--c-bg);
  border: 1px solid var(--c-border);
  border-radius: 4px;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 8px;
  padding: 12px;
}

.file-card {
  position: relative;
  border: 1px solid #E5E7EB;
  border-radius: 4px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: #F3F4F6;
  box-shadow: 0 1px 2px rgba(0,0,0,0.05);
  cursor: default;
  user-select: none;
  transition: all 200ms ease-in-out;
}

.file-card:hover {
  background: #FFFFFF;
  box-shadow: 0 4px 6px rgba(0,0,0,0.08);
  border-color: #D1D5DB;
}

.card-icon-wrap {
  margin-bottom: 8px;
}

.card-name {
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  text-align: center;
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.35;
  max-width: 100%;
}

.card-meta {
  margin-top: 2px;
  font-size: 12px;
  color: #6B7280;
  text-align: center;
  line-height: 1.4;
}

.card-actions {
  margin-top: 8px;
  opacity: 0;
  transition: opacity 200ms ease-in-out;
}

.file-card:hover .card-actions {
  opacity: 1;
}
</style>
