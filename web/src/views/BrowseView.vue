<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import api from '../api'

interface FileItem {
  name: string
  size: number
  modTime: string
  isDir: boolean
  mime: string
}

const path = ref('/Files')
const files = ref<FileItem[]>([])
const loading = ref(false)
const sortKey = ref<'name' | 'size' | 'modTime'>('name')
const sortDir = ref<'asc' | 'desc'>('asc')
const searchQuery = ref('')
const isDragging = ref(false)
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadTotal = ref(0)

function parsePath(vfsPath: string): string[] {
  const parts = vfsPath.split('/').filter(Boolean)
  return parts
}

function buildPath(parts: string[]): string {
  return '/' + parts.join('/')
}

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

function onSearch() {
  // 前端过滤，无需额外操作
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
  }
}

function navigateTo(targetPath: string) {
  path.value = targetPath
  loadFiles()
}

function handleDownload(file: FileItem) {
  window.open(`/api/files/download?path=${path.value}/${file.name}`, '_blank')
}

async function handleRename(file: FileItem) {
  const newName = prompt('输入新名称：', file.name)
  if (!newName || newName === file.name) return
  try {
    const res = await api.put('/files/rename', {
      path: path.value + '/' + file.name,
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

async function handleDelete(file: FileItem) {
  if (!confirm(`确定要删除 "${file.name}" 吗？`)) return
  try {
    const res = await api.delete('/files', {
      params: { path: path.value + '/' + file.name },
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

function sortIndicator(key: 'name' | 'size' | 'modTime'): string {
  if (sortKey.value !== key) return ''
  return sortDir.value === 'asc' ? ' \u25B2' : ' \u25BC'
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
  loadFiles()
})
</script>

<template>
  <div class="browse-view">
    <div class="toolbar">
      <button class="btn btn-primary" @click="handleMkdir">新建文件夹</button>
      <button class="btn" @click="loadFiles">刷新</button>
      <div class="search-box">
        <span class="search-icon">🔍</span>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索文件..."
          class="search-input"
          @input="onSearch"
        />
        <span v-if="searchQuery" class="search-clear" @click="searchQuery = ''">✕</span>
      </div>
    </div>

    <div
      class="drop-zone"
      :class="{ 'drop-active': isDragging }"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @drop.prevent="handleDrop"
    >
      <p>拖拽文件到此处上传，或 <button class="btn-link" @click="triggerUpload">选择文件</button></p>
    </div>
    <input
      ref="fileInput"
      type="file"
      multiple
      style="display: none"
      @change="handleFileSelect"
    />

    <div v-if="uploading" class="upload-progress">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: uploadProgress + '%' }"></div>
      </div>
      <span class="progress-text">{{ uploadProgress }}%</span>
    </div>

    <div class="breadcrumb">
      <template v-for="crumb in breadcrumbs" :key="crumb.path">
        <span class="sep" v-if="!crumb.isLast"> / </span>
        <a
          v-if="!crumb.isLast"
          class="crumb-link"
          href="#"
          @click.prevent="navigateTo(crumb.path)"
        >
          {{ crumb.name }}
        </a>
        <span v-else class="crumb-current">{{ crumb.name }}</span>
      </template>
    </div>

    <div class="file-table-wrapper">
      <div v-if="searchQuery.trim()" class="search-result-info">
        找到 {{ searchResultCount }} 个结果
      </div>
      <table class="file-table" v-if="!loading">
        <thead>
          <tr>
            <th class="col-name" @click="toggleSort('name')">
              名称<span class="sort-indicator">{{ sortIndicator('name') }}</span>
            </th>
            <th class="col-size" @click="toggleSort('size')">
              大小<span class="sort-indicator">{{ sortIndicator('size') }}</span>
            </th>
            <th class="col-time" @click="toggleSort('modTime')">
              修改时间<span class="sort-indicator">{{ sortIndicator('modTime') }}</span>
            </th>
            <th class="col-actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="filteredFiles.length === 0">
            <td colspan="4" class="empty">目录为空</td>
          </tr>
          <tr v-for="file in filteredFiles" :key="file.name" class="file-row">
            <td class="col-name">
              <span
                class="file-icon"
                :class="{ clickable: file.isDir }"
                @click="file.isDir && navigateTo(path + '/' + file.name)"
              >
                {{ file.isDir ? '\uD83D\uDCC1' : '\uD83D\uDCC4' }}
              </span>
              <span
                class="file-name"
                :class="{ clickable: file.isDir }"
                @click="file.isDir && navigateTo(path + '/' + file.name)"
              >
                {{ file.name }}
              </span>
            </td>
            <td class="col-size">
              {{ file.isDir ? '-' : formatSize(file.size) }}
            </td>
            <td class="col-time">{{ formatTime(file.modTime) }}</td>
            <td class="col-actions">
              <button
                v-if="!file.isDir"
                class="btn btn-small"
                @click="handleDownload(file)"
              >
                下载
              </button>
              <button
                class="btn btn-small"
                @click="handleRename(file)"
              >
                重命名
              </button>
              <button
                class="btn btn-small btn-danger"
                @click="handleDelete(file)"
              >
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="loading">加载中...</div>
    </div>
  </div>
</template>

<style scoped>
.browse-view {
  max-width: 960px;
  margin: 0 auto;
  padding: 24px;
}

.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.breadcrumb {
  padding: 8px 0;
  margin-bottom: 16px;
  font-size: 14px;
  color: #666;
}

.crumb-link {
  color: #1677ff;
  text-decoration: none;
}

.crumb-link:hover {
  text-decoration: underline;
}

.crumb-current {
  color: #333;
  font-weight: 500;
}

.sep {
  color: #ccc;
  margin: 0 4px;
}

.file-table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  overflow-x: auto;
}

.file-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.file-table th {
  background: #fafafa;
  padding: 10px 16px;
  text-align: left;
  font-weight: 500;
  color: #333;
  border-bottom: 1px solid #e8e8e8;
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}

.file-table th:hover {
  background: #f0f0f0;
}

.file-table td {
  padding: 10px 16px;
  border-bottom: 1px solid #f0f0f0;
  color: #333;
}

.file-row:hover {
  background: #f5f7fa;
}

.col-name {
  min-width: 200px;
}

.col-size {
  width: 100px;
  white-space: nowrap;
}

.col-time {
  width: 180px;
  white-space: nowrap;
}

.col-actions {
  width: 200px;
  white-space: nowrap;
}

.file-icon {
  margin-right: 8px;
  font-size: 16px;
}

.clickable {
  cursor: pointer;
  color: #1677ff;
}

.clickable:hover {
  text-decoration: underline;
}

.file-name {
  vertical-align: middle;
}

.sort-indicator {
  font-size: 10px;
  margin-left: 2px;
}

.empty {
  text-align: center;
  color: #999;
  padding: 32px 16px !important;
}

.loading {
  text-align: center;
  padding: 32px;
  color: #999;
}

.btn {
  padding: 6px 16px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  color: #333;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn:hover {
  border-color: #1677ff;
  color: #1677ff;
}

.btn-primary {
  background: #1677ff;
  border-color: #1677ff;
  color: #fff;
}

.btn-primary:hover {
  background: #4096ff;
  border-color: #4096ff;
  color: #fff;
}

.btn-small {
  padding: 2px 8px;
  font-size: 12px;
  margin-right: 4px;
}

.btn-danger:hover {
  border-color: #ff4d4f;
  color: #ff4d4f;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  background: #fff;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  padding: 0 8px;
  transition: border-color 0.2s;
}

.search-box:focus-within {
  border-color: #1677ff;
  box-shadow: 0 0 0 2px rgba(22, 119, 255, 0.1);
}

.search-icon {
  font-size: 14px;
  color: #999;
  flex-shrink: 0;
}

.search-input {
  border: none;
  outline: none;
  padding: 6px 0;
  font-size: 14px;
  color: #333;
  width: 180px;
  background: transparent;
}

.search-input::placeholder {
  color: #bfbfbf;
}

.search-clear {
  cursor: pointer;
  color: #999;
  font-size: 14px;
  padding: 2px 4px;
  flex-shrink: 0;
}

.search-clear:hover {
  color: #333;
}

.search-result-info {
  padding: 8px 16px;
  font-size: 13px;
  color: #666;
  border-bottom: 1px solid #e8e8e8;
}

.drop-zone {
  border: 2px dashed #d9d9d9;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
  margin-bottom: 16px;
  color: #999;
  font-size: 14px;
  transition: all 0.2s;
  cursor: pointer;
}

.drop-zone:hover {
  border-color: #1677ff;
  color: #1677ff;
}

.drop-active {
  border-color: #1677ff;
  background: rgba(22, 119, 255, 0.04);
  color: #1677ff;
}

.drop-zone p {
  margin: 0;
}

.btn-link {
  background: none;
  border: none;
  color: #1677ff;
  cursor: pointer;
  font-size: 14px;
  padding: 0;
  text-decoration: underline;
}

.upload-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.progress-bar {
  flex: 1;
  height: 8px;
  background: #f0f0f0;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: #1677ff;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 13px;
  color: #666;
  min-width: 36px;
  text-align: right;
}

.highlight {
  background: #fff3cd;
  font-weight: 600;
}
</style>
