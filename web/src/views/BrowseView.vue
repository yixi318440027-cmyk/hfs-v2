<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { useAuthStore } from '../stores/auth'
import api, { getAuthToken } from '../api'
import { refreshLucide } from '../utils/lucide'
import PublicBrowseView from './PublicBrowseView.vue'

interface FileItem {
  name: string
  size: number
  sizeHuman: string
  modTime: string
  isDir: boolean
  mime: string
  comment: string
  uploadedBy: string
  createdAt: string
  downloads: number
}

interface ContextMenuOption {
  label: string
  icon: string
  action: () => void
  disabled?: boolean
  dividerAfter?: boolean
}

// Auth store
const authStore = useAuthStore()

// --- state ---
const path = ref('/Files')
const files = ref<FileItem[]>([])
const loading = ref(false)
const sortKey = ref<'name' | 'size' | 'modTime'>('name')
const sortDir = ref<'asc' | 'desc'>('asc')
const searchQuery = ref('')
const viewMode = ref<'list' | 'grid'>('list')
const isDragging = ref(false)
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadTotal = ref(0)

// multi-select
const selectedNames = ref<Set<string>>(new Set())
const lastClickedIndex = ref(-1)

// context menu
const ctxVisible = ref(false)
const ctxX = ref(0)
const ctxY = ref(0)
const ctxFile = ref<FileItem | null>(null)
const ctxMenuRef = ref<HTMLElement>()

// --- helpers ---
function parsePath(vfsPath: string): string[] {
  return vfsPath.split('/').filter(Boolean)
}

function buildPath(parts: string[]): string {
  return '/' + parts.join('/')
}

function getFullPath(file: FileItem): string {
  return path.value + '/' + file.name
}

// --- computed ---
const breadcrumbs = computed(() => {
  const parts = parsePath(path.value)
  return parts.map((part, index) => ({
    name: part,
    path: buildPath(parts.slice(0, index + 1)),
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
      case 'name':
        cmp = a.name.localeCompare(b.name)
        break
      case 'size':
        cmp = a.size - b.size
        break
      case 'modTime':
        cmp = new Date(a.modTime).getTime() - new Date(b.modTime).getTime()
        break
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

const selectedCount = computed(() => selectedNames.value.size)

const isAllSelected = computed(() => {
  return filteredFiles.value.length > 0 && selectedNames.value.size === filteredFiles.value.length
})

// context menu items
const ctxMenuItems = computed<ContextMenuOption[]>(() => {
  const selected = getSelectedFiles()
  const multi = selected.length > 1
  const f = ctxFile.value
  const items: ContextMenuOption[] = []

  if (multi) {
    items.push({ label: `复制 ${selected.length} 个路径`, icon: 'copy', action: () => copySelectedPaths() })
    items.push({ label: '批量下载', icon: 'download', action: () => batchDownload(), disabled: selected.every(x => x.isDir) })
    items.push({ label: '批量删除', icon: 'trash-2', action: () => batchDelete(), dividerAfter: true })
    items.push({ label: '取消选中', icon: 'x', action: () => clearSelection() })
    return items
  }

  if (!f) return items

  items.push({ label: '复制路径', icon: 'copy', action: () => copyFilePath(f) })
  if (!f.isDir) {
    items.push({ label: '下载', icon: 'download', action: () => handleDownload(f) })
  }
  items.push({ label: '重命名', icon: 'pencil', action: () => handleRename(f) })
  items.push({ label: '编辑备注', icon: 'message-square', action: () => handleEditComment(f) })
  items.push({ label: '删除', icon: 'trash-2', action: () => handleDelete(f), dividerAfter: true })
  items.push({ label: '属性', icon: 'info', action: () => showProperties(f) })

  return items
})

// --- selection helpers ---
function getSelectedFiles(): FileItem[] {
  return filteredFiles.value.filter(f => selectedNames.value.has(f.name))
}

function isSelected(file: FileItem): boolean {
  return selectedNames.value.has(file.name)
}

function toggleSelect(file: FileItem) {
  const s = new Set(selectedNames.value)
  if (s.has(file.name)) {
    s.delete(file.name)
  } else {
    s.add(file.name)
  }
  selectedNames.value = s
}

function clearSelection() {
  selectedNames.value = new Set()
  lastClickedIndex.value = -1
}

function selectAll() {
  selectedNames.value = new Set(filteredFiles.value.map(f => f.name))
}

function selectRange(from: number, to: number) {
  const start = Math.min(from, to)
  const end = Math.max(from, to)
  const s = new Set(selectedNames.value)
  for (let i = start; i <= end; i++) {
    s.add(filteredFiles.value[i].name)
  }
  selectedNames.value = s
}

// --- data loading ---
function onSearch() {
  // frontend filter
}

async function loadFiles() {
  loading.value = true
  try {
    const res = await api.get('/files', { params: { path: path.value } })
    if (res.data.ok) {
      files.value = res.data.data.files || []
    }
  } catch {
    files.value = []
  } finally {
    loading.value = false
    clearSelection()
    nextTick(() => refreshLucide())
  }
}

function navigateTo(targetPath: string) {
  path.value = targetPath
  searchQuery.value = ''
  loadFiles()
  window.dispatchEvent(new CustomEvent('browse-path-change', { detail: { path: targetPath } }))
}

// --- single file operations ---
async function handleDownload(file: FileItem) {
  const token = getAuthToken()
  const url = `/api/files/download?path=${encodeURIComponent(getFullPath(file))}`
  try {
    const res = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
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

async function handleRename(file: FileItem) {
  const newName = prompt('输入新名称：', file.name)
  if (!newName || newName === file.name) return
  try {
    const res = await api.put('/files/rename', {
      path: getFullPath(file),
      newName,
    })
    if (res.data.ok) {
      loadFiles()
    } else {
      alert('重命名失败')
    }
  } catch {
    alert('重命名失败')
  }
}

async function handleEditComment(file: FileItem) {
  const newComment = prompt('输入文件备注：', file.comment || '')
  if (newComment === null) return // cancelled
  try {
    const res = await api.put('/files/comment', {
      path: getFullPath(file),
      comment: newComment || '',
    })
    if (res.data.ok) {
      loadFiles()
    } else {
      alert('更新备注失败')
    }
  } catch {
    alert('更新备注失败')
  }
}

async function handleDelete(file: FileItem) {
  if (!confirm(`确定要删除 "${file.name}" 吗？`)) return
  try {
    const res = await api.delete('/files', {
      params: { path: getFullPath(file) },
    })
    if (res.data.ok) {
      loadFiles()
    } else {
      alert('删除失败')
    }
  } catch {
    alert('删除失败')
  }
}

// --- batch operations ---
async function batchDelete() {
  const selected = getSelectedFiles()
  if (selected.length === 0) return
  const names = selected.map(f => f.name).join(', ')
  if (!confirm(`确定要删除 ${selected.length} 个文件：${names} 吗？`)) return

  try {
    const res = await api.post('/files/batch-delete', {
      paths: selected.map(f => getFullPath(f)),
    })
    if (res.data.ok) {
      clearSelection()
      loadFiles()
    } else {
      alert('批量删除失败')
    }
  } catch {
    alert('批量删除失败')
  }
}

async function batchDownload() {
  const selected = getSelectedFiles().filter(f => !f.isDir)
  if (selected.length === 0) return
  const token = getAuthToken()
  const params = new URLSearchParams()
  selected.forEach(f => params.append('paths', getFullPath(f)))
  const url = `/api/files/download-zip?${params.toString()}`
  try {
    const res = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({ error: 'download failed' }))
      alert('下载失败: ' + ((data as any).error || res.statusText))
      return
    }
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = 'download.zip'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(blobUrl)
  } catch (err: any) {
    alert('下载失败: ' + (err.message || 'unknown'))
  }
}

// --- context menu ---
function copyFilePath(file: FileItem) {
  navigator.clipboard.writeText(getFullPath(file))
  closeContextMenu()
}

function copySelectedPaths() {
  const paths = getSelectedFiles().map(f => getFullPath(f)).join('\n')
  navigator.clipboard.writeText(paths)
  closeContextMenu()
}

function showProperties(file: FileItem) {
  const info = [
    `名称：${file.name}`,
    `大小：${file.isDir ? '-' : (file.sizeHuman || formatSize(file.size))}`,
    `修改时间：${formatTime(file.modTime)}`,
    `上传时间：${file.createdAt ? formatTime(file.createdAt) : '-'}`,
    `上传者：${file.uploadedBy || '-'}`,
    `下载次数：${file.downloads || 0}`,
    `备注：${file.comment || '-'}`,
    `MIME：${file.mime || '-'}`,
    `类型：${file.isDir ? '文件夹' : '文件'}`,
  ].join('\n')
  alert(info)
  closeContextMenu()
}

function showContextMenu(e: MouseEvent, file: FileItem) {
  e.preventDefault()

  if (!selectedNames.value.has(file.name)) {
    selectedNames.value = new Set([file.name])
  }

  ctxFile.value = file
  ctxX.value = e.clientX
  ctxY.value = e.clientY
  ctxVisible.value = true

  nextTick(() => {
    adjustMenuPosition()
    refreshLucide()
  })
}

function adjustMenuPosition() {
  if (!ctxMenuRef.value) return
  const rect = ctxMenuRef.value.getBoundingClientRect()
  const vw = window.innerWidth
  const vh = window.innerHeight

  if (rect.right > vw) {
    ctxX.value = vw - rect.width - 8
  }
  if (rect.bottom > vh) {
    ctxY.value = vh - rect.height - 8
  }
}

function closeContextMenu() {
  ctxVisible.value = false
  ctxFile.value = null
}

function handleContextMenuAction(action: () => void) {
  closeContextMenu()
  action()
}

// --- keyboard shortcuts ---
function handleKeydown(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement).tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

  if (e.key === 'F2' || e.code === 'F2') {
    e.preventDefault()
    const first = getSelectedFiles()[0]
    if (first) handleRename(first)
    return
  }

  if (e.key === 'Delete' || e.code === 'Delete') {
    e.preventDefault()
    const selected = getSelectedFiles()
    if (selected.length === 0) return
    if (selected.length === 1) {
      handleDelete(selected[0])
    } else {
      batchDelete()
    }
    return
  }

  if (e.key === 'F5' || e.code === 'F5') {
    e.preventDefault()
    loadFiles()
    return
  }

  if (e.ctrlKey && e.key === 'a') {
    e.preventDefault()
    selectAll()
    return
  }

  if (e.key === 'Escape') {
    closeContextMenu()
    return
  }
}

// --- row click handling ---
function handleRowClick(e: MouseEvent, file: FileItem, index: number) {
  if (e.ctrlKey || e.metaKey) {
    toggleSelect(file)
    lastClickedIndex.value = index
  } else if (e.shiftKey && lastClickedIndex.value >= 0) {
    selectRange(lastClickedIndex.value, index)
  } else {
    selectedNames.value = new Set([file.name])
    lastClickedIndex.value = index
  }
}

function handleCheckboxChange(file: FileItem, index: number) {
  toggleSelect(file)
  if (isSelected(file)) {
    lastClickedIndex.value = index
  }
}

function handleHeaderCheckboxChange() {
  if (isAllSelected.value) {
    clearSelection()
  } else {
    selectAll()
  }
}

// --- file operations ---
async function handleMkdir() {
  const dirName = prompt('输入文件夹名称：')
  if (!dirName) return
  try {
    const res = await api.post('/files/mkdir', {
      path: path.value,
      dirName,
    })
    if (res.data.ok) {
      loadFiles()
    } else {
      alert('创建文件夹失败')
    }
  } catch {
    alert('创建文件夹失败')
  }
}

function triggerUpload() {
  fileInput.value?.click()
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) {
    uploadFiles(Array.from(input.files))
    input.value = ''
  }
}

function handleDrop(e: DragEvent) {
  isDragging.value = false
  if (e.dataTransfer?.files) {
    uploadFiles(Array.from(e.dataTransfer.files))
  }
}

async function uploadFiles(fileList: File[]) {
  if (fileList.length === 0) return
  uploading.value = true
  uploadProgress.value = 0
  uploadTotal.value = fileList.length

  const formData = new FormData()
  formData.append('path', path.value)
  fileList.forEach(f => formData.append('files', f))

  try {
    const res = await api.post('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e: any) => {
        if (e.total) {
          uploadProgress.value = Math.round((e.loaded / e.total) * 100)
        }
      },
    })
    if (res.data.ok) {
      const uploaded = res.data.data.uploaded?.length || 0
      const errors = res.data.data.errors?.length || 0
      if (errors > 0) {
        alert(`上传完成：${uploaded} 个成功，${errors} 个失败`)
      }
      loadFiles()
    }
  } catch {
    alert('上传失败')
  } finally {
    uploading.value = false
    uploadProgress.value = 0
    uploadTotal.value = 0
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

// --- lifecycle ---
function onDocumentClick() {
  closeContextMenu()
}

function onFileTreeNavigate(e: Event) {
  const detail = (e as CustomEvent).detail
  if (detail?.path) {
    navigateTo(detail.path)
  }
}

onMounted(() => {
  loadFiles()
  document.addEventListener('click', onDocumentClick)
  window.addEventListener('filetree-navigate', onFileTreeNavigate)
})

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick)
  window.removeEventListener('filetree-navigate', onFileTreeNavigate)
})
</script>

<template>
  <!-- Show public browse view when not logged in -->
  <PublicBrowseView v-if="!authStore.isLoggedIn" />

  <!-- Full file management when logged in -->
  <div v-else class="browse-view" @keydown="handleKeydown" tabindex="0">
    <!-- Page header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">文件管理</h1>
        <p class="page-desc">管理您的所有文件和目录</p>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="toolbar-left">
        <button class="btn btn-primary" @click="handleMkdir">
          <i data-lucide="folder-plus" style="width:14px;height:14px"></i>
          新建文件夹
        </button>
        <button class="btn" @click="loadFiles">
          <i data-lucide="refresh-cw" style="width:14px;height:14px"></i>
          刷新
        </button>
        <button class="btn" @click="triggerUpload">
          <i data-lucide="upload" style="width:14px;height:14px"></i>
          上传
        </button>
      </div>

      <div class="toolbar-right">
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
            @input="onSearch"
          />
          <button v-if="searchQuery" class="search-clear" @click="searchQuery = ''">
            <i data-lucide="x" style="width:14px;height:14px"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- Batch toolbar -->
    <div v-if="selectedCount > 0" class="batch-bar">
      <i data-lucide="check-square" style="width:14px;height:14px" class="batch-icon"></i>
      <span class="batch-label">已选 {{ selectedCount }} 项</span>
      <button class="btn btn-sm btn-danger" @click="batchDelete">
        <i data-lucide="trash-2" style="width:12px;height:12px"></i>
        批量删除
      </button>
      <button class="btn btn-sm" @click="batchDownload">
        <i data-lucide="download" style="width:12px;height:12px"></i>
        批量下载
      </button>
      <button class="btn btn-sm" @click="clearSelection">取消选中</button>
    </div>

    <!-- Drop zone -->
    <div
      class="drop-zone"
      :class="{ 'drop-active': isDragging }"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @drop.prevent="handleDrop"
    >
      <i data-lucide="upload-cloud" style="width:20px;height:20px;color:var(--c-text-muted)"></i>
      <span>拖拽文件到此处上传，或 <button class="btn-link" @click="triggerUpload">选择文件</button></span>
    </div>
    <input
      ref="fileInput"
      type="file"
      multiple
      style="display: none"
      @change="handleFileSelect"
    />

    <!-- Upload progress -->
    <div v-if="uploading" class="upload-progress">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: uploadProgress + '%' }"></div>
      </div>
      <span class="progress-text">{{ uploadProgress }}%</span>
    </div>

    <!-- Breadcrumb -->
    <div class="breadcrumb">
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

    <!-- File table (list view) -->
    <div v-if="viewMode === 'list'" class="file-table-wrapper">
      <div v-if="searchQuery.trim()" class="search-info">
        <i data-lucide="search" style="width:13px;height:13px"></i>
        找到 {{ searchResultCount }} 个结果
      </div>

      <!-- Loading skeleton -->
      <div v-if="loading" class="skeleton-list">
        <div v-for="i in 8" :key="i" class="skeleton-row">
          <div class="skeleton-cell" style="width:20px"></div>
          <div class="skeleton-cell" style="flex:1"></div>
          <div class="skeleton-cell" style="width:80px"></div>
          <div class="skeleton-cell" style="width:140px"></div>
          <div class="skeleton-cell" style="width:120px"></div>
        </div>
      </div>

      <table v-else class="file-table">
        <thead>
          <tr>
            <th class="col-check">
              <input
                type="checkbox"
                :checked="isAllSelected"
                :indeterminate.prop="selectedCount > 0 && !isAllSelected"
                @change="handleHeaderCheckboxChange"
              />
            </th>
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
            <th class="col-comment">备注</th>
            <th class="col-actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <!-- Empty state -->
          <tr v-if="filteredFiles.length === 0">
            <td colspan="7">
              <div class="empty-state">
                <i data-lucide="folder-open" style="width:40px;height:40px;color:var(--c-text-muted)"></i>
                <div class="empty-text">当前目录为空</div>
                <div class="empty-hint">拖拽文件到此处，或点击上方按钮创建文件夹 / 上传文件</div>
              </div>
            </td>
          </tr>

          <tr
            v-for="(file, index) in filteredFiles"
            :key="file.name"
            class="file-row"
            :class="{
              'row-selected': isSelected(file),
              'row-stripe': index % 2 === 1 && !isSelected(file),
            }"
            @click="handleRowClick($event, file, index)"
            @contextmenu="showContextMenu($event, file)"
          >
            <td class="col-check" @click.stop>
              <input
                type="checkbox"
                :checked="isSelected(file)"
                @change="handleCheckboxChange(file, index)"
              />
            </td>
            <td class="col-name">
              <i
                v-if="file.isDir"
                data-lucide="folder"
                class="file-icon file-icon-folder"
                :class="{ clickable: file.isDir }"
                @click.stop="file.isDir && navigateTo(path + '/' + file.name)"
              ></i>
              <i
                v-else
                data-lucide="file-text"
                class="file-icon file-icon-file"
              ></i>
              <span
                class="file-name"
                :class="{ clickable: file.isDir }"
                @click.stop="file.isDir && navigateTo(path + '/' + file.name)"
              >
                {{ file.name }}
              </span>
            </td>
            <td class="col-size">
              {{ file.isDir ? '--' : (file.sizeHuman || formatSize(file.size)) }}
            </td>
            <td class="col-downloads">{{ file.isDir ? '--' : (file.downloads || 0) }}</td>
            <td class="col-time">{{ formatTime(file.modTime) }}</td>
            <td class="col-comment" :title="file.comment">{{ file.comment || '--' }}</td>
            <td class="col-actions" @click.stop>
              <button
                v-if="!file.isDir"
                class="btn-dl"
                @click="handleDownload(file)"
              >
                <i data-lucide="download" style="width:12px;height:12px"></i>
                下载
              </button>
              <button class="btn-dl" @click="handleRename(file)">
                <i data-lucide="pencil" style="width:12px;height:12px"></i>
                重命名
              </button>
              <button class="btn-dl btn-dl-danger" @click="handleDelete(file)">
                <i data-lucide="trash-2" style="width:12px;height:12px"></i>
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- File grid (grid view) -->
    <div v-else class="file-grid-wrapper">
      <div v-if="searchQuery.trim()" class="search-info">
        <i data-lucide="search" style="width:13px;height:13px"></i>
        找到 {{ searchResultCount }} 个结果
      </div>

      <div v-if="loading" class="skeleton-grid">
        <div v-for="i in 8" :key="i" class="skeleton-card"></div>
      </div>

      <div v-else-if="filteredFiles.length === 0" class="empty-state">
        <i data-lucide="folder-open" style="width:40px;height:40px;color:var(--c-text-muted)"></i>
        <div class="empty-text">当前目录为空</div>
        <div class="empty-hint">拖拽文件到此处，或点击上方按钮创建文件夹 / 上传文件</div>
      </div>

      <div v-else class="file-grid">
        <div
          v-for="(file, index) in filteredFiles"
          :key="file.name"
          class="file-card"
          :class="{ 'card-selected': isSelected(file) }"
          @click="handleRowClick($event, file, index)"
          @contextmenu="showContextMenu($event, file)"
          @dblclick="file.isDir && navigateTo(path + '/' + file.name)"
        >
          <div class="card-check">
            <input
              type="checkbox"
              :checked="isSelected(file)"
              @click.stop
              @change="handleCheckboxChange(file, index)"
            />
          </div>
          <div class="card-icon-wrap">
            <i v-if="file.isDir" data-lucide="folder-open" style="width:32px;height:32px;color:#F59E0B"></i>
            <i v-else data-lucide="file-text" style="width:32px;height:32px;color:#6B7280"></i>
          </div>
          <div class="card-name" :title="file.name">{{ file.name }}</div>
          <div class="card-meta">
            <span v-if="!file.isDir">{{ file.sizeHuman || formatSize(file.size) }}</span>
            <span v-else>文件夹</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Context menu -->
    <Teleport to="body">
      <div
        v-if="ctxVisible"
        ref="ctxMenuRef"
        class="context-menu"
        :style="{ left: ctxX + 'px', top: ctxY + 'px' }"
        @click.stop
      >
        <template v-for="(item, idx) in ctxMenuItems" :key="idx">
          <div
            class="ctx-item"
            :class="{ 'ctx-disabled': item.disabled }"
            @click="!item.disabled && handleContextMenuAction(item.action)"
          >
            <i :data-lucide="item.icon" style="width:14px;height:14px"></i>
            <span>{{ item.label }}</span>
          </div>
          <div v-if="item.dividerAfter" class="ctx-divider"></div>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.browse-view {
  width: 100%;
  padding: 24px 28px;
  box-sizing: border-box;
  outline: none;
}

/* Page header */
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
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* View toggle */
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

/* Batch bar */
.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 12px;
  margin-bottom: 12px;
  background: var(--c-primary-light);
  border: 1px solid var(--c-primary-border);
  border-radius: 4px;
}

.batch-icon {
  color: var(--c-primary);
}

.batch-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--c-primary);
}

/* Drop zone */
.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px dashed var(--c-border);
  border-radius: 4px;
  padding: 12px;
  margin-bottom: 12px;
  color: var(--c-text-muted);
  font-size: 13px;
  transition: all 200ms ease-in-out;
  cursor: pointer;
}

.drop-zone:hover {
  border-color: var(--c-primary-border);
  color: var(--c-primary);
  background: var(--c-primary-light);
}

.drop-active {
  border-color: var(--c-primary) !important;
  background: var(--c-primary-light) !important;
  color: var(--c-primary) !important;
}

/* Upload progress */
.upload-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.progress-bar {
  flex: 1;
  height: 4px;
  background: var(--c-bg);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--c-primary);
  border-radius: 2px;
  transition: width 300ms ease;
}

.progress-text {
  font-size: 12px;
  color: var(--c-text-muted);
  min-width: 32px;
  text-align: right;
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

.file-table thead {
  position: sticky;
  top: 0;
  z-index: 5;
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

.col-check {
  width: 36px;
  text-align: center;
  padding: 0 8px !important;
}

.col-check input[type='checkbox'] {
  cursor: pointer;
  width: 14px;
  height: 14px;
  accent-color: var(--c-primary);
}

.col-name {
  min-width: 200px;
}

.col-size {
  width: 90px;
  white-space: nowrap;
  color: var(--c-text-muted);
  font-size: 12px;
}

.col-downloads {
  width: 60px;
  text-align: center;
  white-space: nowrap;
  color: var(--c-text-muted);
  font-size: 12px;
}

.col-time {
  width: 150px;
  white-space: nowrap;
  color: var(--c-text-muted);
  font-size: 12px;
}

.col-comment {
  width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--c-text-muted);
  font-size: 12px;
}

.col-actions {
  width: 200px;
  white-space: nowrap;
}

/* File rows */
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

.row-selected {
  background: var(--c-selected-bg) !important;
  box-shadow: inset 2px 0 0 var(--c-primary);
}

.file-icon {
  margin-right: 8px;
  vertical-align: middle;
}

.file-icon-folder {
  color: #F59E0B;
}

.file-icon-file {
  color: #6B7280;
}

.clickable {
  cursor: pointer;
  color: var(--c-primary);
}

.clickable:hover {
  text-decoration: underline;
}

.file-name {
  vertical-align: middle;
}

/* Download button */
.btn-dl {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 8px;
  margin-right: 4px;
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

.btn-dl-danger {
  color: var(--c-danger);
}

.btn-dl-danger:hover {
  background: var(--c-danger-light);
  border-color: #FECACA;
  color: var(--c-danger);
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

/* Skeleton loading */
.skeleton-list {
  padding: 12px;
}

.skeleton-row {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 40px;
  padding: 0 12px;
}

.skeleton-cell {
  height: 12px;
  border-radius: 4px;
  background: var(--c-bg);
  animation: shimmer 1.5s ease-in-out infinite;
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  padding: 16px;
}

.skeleton-card {
  height: 120px;
  border-radius: 4px;
  background: var(--c-bg);
  animation: shimmer 1.5s ease-in-out infinite;
}

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

.card-selected {
  background: #EFF6FF !important;
  border-color: #3B82F6 !important;
  box-shadow: 0 1px 2px rgba(0,0,0,0.05) !important;
}

.card-check {
  position: absolute;
  top: 6px;
  left: 6px;
}

.card-check input[type='checkbox'] {
  cursor: pointer;
  width: 14px;
  height: 14px;
  accent-color: #3B82F6;
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
</style>
