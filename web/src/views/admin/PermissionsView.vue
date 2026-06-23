<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import api from '../../api'
import { refreshLucide } from '../../utils/lucide'

interface PermissionRecord {
  id: number
  username: string
  vfsPath: string
  canSee: boolean
  canRead: boolean
  canList: boolean
  canUpload: boolean
  canDelete: boolean
  canArchive: boolean
}

interface UserItem {
  id: number
  username: string
  role: string
  enabled: boolean
}

interface TreePath {
  path: string
  name: string
  isRoot: boolean
}

// Visibility mode shortcut
type VisibilityMode = 'public' | 'login' | 'restricted'

// State
const permissions = ref<PermissionRecord[]>([])
const users = ref<UserItem[]>([])
const treePaths = ref<TreePath[]>([])
const loading = ref(false)
const error = ref('')

// Edit modal
const showEdit = ref(false)
const editingId = ref<number | null>(null)
const editUsername = ref('')
const editPath = ref('')
const editCanSee = ref(true)
const editCanRead = ref(true)
const editCanList = ref(true)
const editCanUpload = ref(false)
const editCanDelete = ref(false)
const editCanArchive = ref(false)
const visibilityMode = ref<VisibilityMode>('restricted')

// Path picker state
const showPathPicker = ref(false)
const pathPickerFilter = ref('')
const pathPickerTarget = ref<'edit' | 'batch'>('edit')
const pathPickerExpanded = ref<Set<string>>(new Set())

// Batch modal
const showBatch = ref(false)
const batchPath = ref('')
const batchCanSee = ref(true)
const batchCanRead = ref(true)
const batchCanList = ref(true)
const batchCanUpload = ref(false)
const batchCanDelete = ref(false)
const batchCanArchive = ref(false)
const batchUsernames = ref<string[]>([])
const batchVisibilityMode = ref<VisibilityMode>('restricted')

// Fetch all permissions
async function fetchPermissions() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get('/admin/permissions')
    permissions.value = (data as any).data || []
  } catch (e: any) {
    error.value = e.response?.data?.error || '加载权限失败'
  } finally {
    loading.value = false
  }
}

// Fetch users for dropdowns
async function fetchUsers() {
  try {
    const { data } = await api.get('/admin/users')
    users.value = (data as any).data || []
  } catch (_) {
    // non-critical
  }
}

// Fetch VFS tree paths for path picker
async function fetchTreePaths() {
  try {
    const { data } = await api.get('/files/tree')
    treePaths.value = (data as any).data || []
  } catch (_) {
    treePaths.value = []
  }
}

// Apply visibility mode to edit form
function applyVisibilityMode(mode: VisibilityMode) {
  visibilityMode.value = mode
  editCanSee.value = true
  editCanRead.value = true
  editCanList.value = true
}

function applyBatchVisibilityMode(mode: VisibilityMode) {
  batchVisibilityMode.value = mode
  batchCanSee.value = true
  batchCanRead.value = true
  batchCanList.value = true
}

// Open edit modal for existing record
function openEdit(record: PermissionRecord) {
  editingId.value = record.id
  editUsername.value = record.username
  editPath.value = record.vfsPath
  editCanSee.value = record.canSee
  editCanRead.value = record.canRead
  editCanList.value = record.canList
  editCanUpload.value = record.canUpload
  editCanDelete.value = record.canDelete
  editCanArchive.value = record.canArchive
  visibilityMode.value = 'restricted'
  showEdit.value = true
}

// Open create modal
function openCreate() {
  editingId.value = null
  editUsername.value = ''
  editPath.value = ''
  editCanSee.value = true
  editCanRead.value = true
  editCanList.value = true
  editCanUpload.value = false
  editCanDelete.value = false
  editCanArchive.value = false
  visibilityMode.value = 'restricted'
  showEdit.value = true
}

// Save permission (create or update)
async function savePermission() {
  if (!editUsername.value || !editPath.value) {
    error.value = '用户名和路径不能为空'
    return
  }
  try {
    await api.post('/admin/permissions', {
      username: editUsername.value,
      vfsPath: editPath.value,
      canSee: editCanSee.value,
      canRead: editCanRead.value,
      canList: editCanList.value,
      canUpload: editCanUpload.value,
      canDelete: editCanDelete.value,
      canArchive: editCanArchive.value,
    })
    showEdit.value = false
    await fetchPermissions()
  } catch (e: any) {
    error.value = e.response?.data?.error || '保存失败'
  }
}

// Delete a permission record
async function deletePermission(id: number) {
  if (!confirm('确定要删除这条权限记录吗？')) return
  try {
    await api.delete(`/admin/permissions?id=${id}`)
    await fetchPermissions()
  } catch (e: any) {
    error.value = e.response?.data?.error || '删除失败'
  }
}

// --- Path picker ---

// Computed: group tree paths by parent
const groupedPaths = computed(() => {
  const filtered = pathPickerFilter.value
    ? treePaths.value.filter(p =>
        p.path.toLowerCase().includes(pathPickerFilter.value.toLowerCase()) ||
        p.name.toLowerCase().includes(pathPickerFilter.value.toLowerCase())
      )
    : treePaths.value

  // Build a map: parent path → children
  const map = new Map<string, TreePath[]>()
  for (const p of filtered) {
    const parent = p.isRoot ? '' : p.path.substring(0, p.path.lastIndexOf('/'))
    const key = parent || '__roots__'
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(p)
  }
  return map
})

function openPathPicker(target: 'edit' | 'batch') {
  pathPickerTarget.value = target
  pathPickerFilter.value = ''
  // Auto-expand roots
  pathPickerExpanded.value = new Set(['__roots__'])
  showPathPicker.value = true
}

function togglePickerExpand(key: string) {
  const expanded = new Set(pathPickerExpanded.value)
  if (expanded.has(key)) {
    expanded.delete(key)
  } else {
    expanded.add(key)
  }
  pathPickerExpanded.value = expanded
}

function selectPath(path: string) {
  if (pathPickerTarget.value === 'edit') {
    editPath.value = path
  } else {
    batchPath.value = path
  }
  showPathPicker.value = false
}

// Computed helper (needed for template)
import { computed } from 'vue'

// Batch grant
function openBatch() {
  batchPath.value = ''
  batchUsernames.value = []
  batchCanSee.value = true
  batchCanRead.value = true
  batchCanList.value = true
  batchCanUpload.value = false
  batchCanDelete.value = false
  batchCanArchive.value = false
  batchVisibilityMode.value = 'restricted'
  showBatch.value = true
}

async function saveBatch() {
  if (batchUsernames.value.length === 0 || !batchPath.value) {
    error.value = '请选择用户并填写路径'
    return
  }
  try {
    await api.post('/admin/permissions/batch', {
      usernames: batchUsernames.value,
      vfsPath: batchPath.value,
      canSee: batchCanSee.value,
      canRead: batchCanRead.value,
      canList: batchCanList.value,
      canUpload: batchCanUpload.value,
      canDelete: batchCanDelete.value,
      canArchive: batchCanArchive.value,
    })
    showBatch.value = false
    await fetchPermissions()
  } catch (e: any) {
    error.value = e.response?.data?.error || '批量授权失败'
  }
}

function toggleBatchUser(username: string) {
  const idx = batchUsernames.value.indexOf(username)
  if (idx >= 0) {
    batchUsernames.value.splice(idx, 1)
  } else {
    batchUsernames.value.push(username)
  }
}

// Recursive tree renderer helper
function renderTreeNodes(parentKey: string, depth: number): any[] {
  const children = groupedPaths.value.get(parentKey)
  if (!children || children.length === 0) return []

  return children.map(child => {
    const childKey = child.isRoot ? '__roots__' : child.path
    const hasChildren = groupedPaths.value.has(child.path)
    const isExpanded = pathPickerExpanded.value.has(child.path)

    return {
      ...child,
      depth,
      hasChildren,
      isExpanded,
    }
  })
}

onMounted(async () => {
  await Promise.all([fetchPermissions(), fetchUsers(), fetchTreePaths()])
  await nextTick()
  refreshLucide()
})

watch([showEdit, showBatch, showPathPicker], async () => {
  await nextTick()
  refreshLucide()
})
</script>

<template>
  <div class="permissions-page">
    <div class="page-header">
      <h1 class="page-title">权限管理</h1>
      <div class="header-actions">
        <button class="btn btn-outline" @click="openBatch">
          <i data-lucide="users" class="btn-icon-left"></i>
          批量授权
        </button>
        <button class="btn btn-primary" @click="openCreate">
          <i data-lucide="plus" class="btn-icon-left"></i>
          新增权限
        </button>
      </div>
    </div>

    <div v-if="error" class="alert alert-error">
      <i data-lucide="alert-circle"></i>
      {{ error }}
      <button class="alert-close" @click="error = ''">&times;</button>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else class="perm-table-wrapper">
      <table class="perm-table">
        <thead>
          <tr>
            <th class="col-user">用户</th>
            <th class="col-path">VFS 路径</th>
            <th class="col-perm">查看</th>
            <th class="col-perm">读取</th>
            <th class="col-perm">列表</th>
            <th class="col-perm">上传</th>
            <th class="col-perm">删除</th>
            <th class="col-perm">归档</th>
            <th class="col-actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="permissions.length === 0">
            <td colspan="9" class="empty-row">暂无权限记录</td>
          </tr>
          <tr v-for="perm in permissions" :key="perm.id">
            <td class="col-user">
              <span class="user-badge">{{ perm.username }}</span>
            </td>
            <td class="col-path">
              <code>{{ perm.vfsPath }}</code>
            </td>
            <td class="col-perm">
              <span class="perm-dot" :class="{ granted: perm.canSee, denied: !perm.canSee }"></span>
            </td>
            <td class="col-perm">
              <span class="perm-dot" :class="{ granted: perm.canRead, denied: !perm.canRead }"></span>
            </td>
            <td class="col-perm">
              <span class="perm-dot" :class="{ granted: perm.canList, denied: !perm.canList }"></span>
            </td>
            <td class="col-perm">
              <span class="perm-dot" :class="{ granted: perm.canUpload, denied: !perm.canUpload }"></span>
            </td>
            <td class="col-perm">
              <span class="perm-dot" :class="{ granted: perm.canDelete, denied: !perm.canDelete }"></span>
            </td>
            <td class="col-perm">
              <span class="perm-dot" :class="{ granted: perm.canArchive, denied: !perm.canArchive }"></span>
            </td>
            <td class="col-actions">
              <button class="btn-action" @click="openEdit(perm)" title="编辑">
                <i data-lucide="pencil"></i>
              </button>
              <button class="btn-action btn-action-danger" @click="deletePermission(perm.id)" title="删除">
                <i data-lucide="trash-2"></i>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Edit / Create Modal -->
    <div v-if="showEdit" class="modal-overlay" @click.self="showEdit = false">
      <div class="modal-content">
        <div class="modal-header">
          <h2>{{ editingId ? '编辑权限' : '新增权限' }}</h2>
          <button class="modal-close" @click="showEdit = false">
            <i data-lucide="x"></i>
          </button>
        </div>

        <div class="modal-body">
          <div class="form-row">
            <label class="form-label">用户</label>
            <select v-model="editUsername" class="form-select" :disabled="!!editingId">
              <option value="">选择用户</option>
              <option v-for="u in users" :key="u.username" :value="u.username">
                {{ u.username }} {{ u.role === 'admin' ? '(管理员)' : '' }}
              </option>
            </select>
          </div>

          <div class="form-row">
            <label class="form-label">VFS 路径</label>
            <div class="path-input-row">
              <input
                v-model="editPath"
                type="text"
                class="form-input path-input"
                placeholder="如 /Files、/Files/project-a"
                :disabled="!!editingId"
              />
              <button
                v-if="!editingId"
                class="btn btn-outline btn-path-ref"
                @click="openPathPicker('edit')"
                title="引用已有路径"
              >
                <i data-lucide="folder-tree" class="btn-icon-left"></i>
                引用
              </button>
            </div>
          </div>

          <!-- Visibility shortcut -->
          <div class="form-row">
            <label class="form-label">可见性级别</label>
            <div class="visibility-tabs">
              <button
                class="vis-tab"
                :class="{ active: visibilityMode === 'public' }"
                @click="applyVisibilityMode('public')"
              >
                <i data-lucide="globe"></i> 公开可见
              </button>
              <button
                class="vis-tab"
                :class="{ active: visibilityMode === 'login' }"
                @click="applyVisibilityMode('login')"
              >
                <i data-lucide="lock"></i> 仅登录可见
              </button>
              <button
                class="vis-tab"
                :class="{ active: visibilityMode === 'restricted' }"
                @click="applyVisibilityMode('restricted')"
              >
                <i data-lucide="user-check"></i> 指定用户
              </button>
            </div>
            <span class="form-hint">切换可见性级别会自动设置"查看/读取/列表"权限</span>
          </div>

          <!-- Permission checkboxes -->
          <div class="perm-grid">
            <label class="perm-check">
              <input type="checkbox" v-model="editCanSee" />
              <span>查看 (can_see)</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="editCanRead" />
              <span>读取 (can_read)</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="editCanList" />
              <span>列表 (can_list)</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="editCanUpload" />
              <span>上传 (can_upload)</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="editCanDelete" />
              <span>删除 (can_delete)</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="editCanArchive" />
              <span>归档 (can_archive)</span>
            </label>
          </div>
        </div>

        <div class="modal-footer">
          <button class="btn btn-outline" @click="showEdit = false">取消</button>
          <button class="btn btn-primary" @click="savePermission">保存</button>
        </div>
      </div>
    </div>

    <!-- Batch Modal -->
    <div v-if="showBatch" class="modal-overlay" @click.self="showBatch = false">
      <div class="modal-content">
        <div class="modal-header">
          <h2>批量授权</h2>
          <button class="modal-close" @click="showBatch = false">
            <i data-lucide="x"></i>
          </button>
        </div>

        <div class="modal-body">
          <div class="form-row">
            <label class="form-label">VFS 路径</label>
            <div class="path-input-row">
              <input v-model="batchPath" type="text" class="form-input path-input" placeholder="如 /Files/shared" />
              <button class="btn btn-outline btn-path-ref" @click="openPathPicker('batch')" title="引用已有路径">
                <i data-lucide="folder-tree" class="btn-icon-left"></i>
                引用
              </button>
            </div>
          </div>

          <div class="form-row">
            <label class="form-label">选择用户</label>
            <div class="user-check-list">
              <label v-for="u in users" :key="u.username" class="user-check-item">
                <input
                  type="checkbox"
                  :checked="batchUsernames.includes(u.username)"
                  @change="toggleBatchUser(u.username)"
                />
                <span>{{ u.username }}</span>
              </label>
            </div>
          </div>

          <div class="form-row">
            <label class="form-label">可见性级别</label>
            <div class="visibility-tabs">
              <button class="vis-tab" :class="{ active: batchVisibilityMode === 'public' }" @click="applyBatchVisibilityMode('public')">
                <i data-lucide="globe"></i> 公开可见
              </button>
              <button class="vis-tab" :class="{ active: batchVisibilityMode === 'login' }" @click="applyBatchVisibilityMode('login')">
                <i data-lucide="lock"></i> 仅登录可见
              </button>
              <button class="vis-tab" :class="{ active: batchVisibilityMode === 'restricted' }" @click="applyBatchVisibilityMode('restricted')">
                <i data-lucide="user-check"></i> 指定用户
              </button>
            </div>
          </div>

          <div class="perm-grid">
            <label class="perm-check">
              <input type="checkbox" v-model="batchCanSee" />
              <span>查看</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="batchCanRead" />
              <span>读取</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="batchCanList" />
              <span>列表</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="batchCanUpload" />
              <span>上传</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="batchCanDelete" />
              <span>删除</span>
            </label>
            <label class="perm-check">
              <input type="checkbox" v-model="batchCanArchive" />
              <span>归档</span>
            </label>
          </div>
        </div>

        <div class="modal-footer">
          <button class="btn btn-outline" @click="showBatch = false">取消</button>
          <button class="btn btn-primary" @click="saveBatch">批量授权</button>
        </div>
      </div>
    </div>

    <!-- Path Picker Modal -->
    <div v-if="showPathPicker" class="modal-overlay" @click.self="showPathPicker = false">
      <div class="modal-content path-picker-modal">
        <div class="modal-header">
          <h2>引用文件路径</h2>
          <button class="modal-close" @click="showPathPicker = false">
            <i data-lucide="x"></i>
          </button>
        </div>

        <div class="modal-body">
          <div class="picker-search">
            <i data-lucide="search" class="picker-search-icon"></i>
            <input
              v-model="pathPickerFilter"
              type="text"
              class="form-input"
              placeholder="搜索路径或目录名..."
            />
          </div>

          <div class="picker-tree">
            <!-- Root level -->
            <template v-for="root in treePaths.filter(p => p.isRoot)" :key="root.path">
              <div class="picker-node-group">
                <div
                  class="picker-node picker-node-root"
                  :class="{ 'picker-node-selected': (pathPickerTarget === 'edit' ? editPath : batchPath) === root.path }"
                >
                  <button
                    class="picker-toggle"
                    :class="{ 'picker-toggle-expanded': pathPickerExpanded.has(root.path) }"
                    @click="togglePickerExpand(root.path)"
                  >
                    <i data-lucide="chevron-right" style="width:12px;height:12px"></i>
                  </button>
                  <i data-lucide="folder" class="picker-folder-icon"></i>
                  <span class="picker-name" @click="selectPath(root.path)">{{ root.name }}</span>
                  <code class="picker-path-hint">{{ root.path }}</code>
                </div>

                <!-- Level 1 children -->
                <div v-if="pathPickerExpanded.has(root.path)" class="picker-children">
                  <template v-for="child in treePaths.filter(p => !p.isRoot && p.path.startsWith(root.path + '/') && p.path.split('/').length === root.path.split('/').length + 1)" :key="child.path">
                    <div class="picker-node-group">
                      <div
                        class="picker-node"
                        :class="{ 'picker-node-selected': (pathPickerTarget === 'edit' ? editPath : batchPath) === child.path }"
                      >
                        <button
                          class="picker-toggle"
                          :class="{ 'picker-toggle-expanded': pathPickerExpanded.has(child.path) }"
                          @click="togglePickerExpand(child.path)"
                        >
                          <i data-lucide="chevron-right" style="width:12px;height:12px"></i>
                        </button>
                        <i data-lucide="folder" class="picker-folder-icon"></i>
                        <span class="picker-name" @click="selectPath(child.path)">{{ child.name }}</span>
                        <code class="picker-path-hint">{{ child.path }}</code>
                      </div>

                      <!-- Level 2 children -->
                      <div v-if="pathPickerExpanded.has(child.path)" class="picker-children">
                        <template v-for="grandchild in treePaths.filter(p => !p.isRoot && p.path.startsWith(child.path + '/') && p.path.split('/').length === child.path.split('/').length + 1)" :key="grandchild.path">
                          <div class="picker-node-group">
                            <div
                              class="picker-node"
                              :class="{ 'picker-node-selected': (pathPickerTarget === 'edit' ? editPath : batchPath) === grandchild.path }"
                            >
                              <button
                                class="picker-toggle"
                                :class="{ 'picker-toggle-expanded': pathPickerExpanded.has(grandchild.path) }"
                                @click="togglePickerExpand(grandchild.path)"
                              >
                                <i data-lucide="chevron-right" style="width:12px;height:12px"></i>
                              </button>
                              <i data-lucide="folder" class="picker-folder-icon"></i>
                              <span class="picker-name" @click="selectPath(grandchild.path)">{{ grandchild.name }}</span>
                              <code class="picker-path-hint">{{ grandchild.path }}</code>
                            </div>

                            <!-- Level 3+ recursive -->
                            <div v-if="pathPickerExpanded.has(grandchild.path)" class="picker-children">
                              <template v-for="descendant in treePaths.filter(p => !p.isRoot && p.path.startsWith(grandchild.path + '/') && p.path.split('/').length === grandchild.path.split('/').length + 1)" :key="descendant.path">
                                <div
                                  class="picker-node"
                                  :class="{ 'picker-node-selected': (pathPickerTarget === 'edit' ? editPath : batchPath) === descendant.path }"
                                >
                                  <span class="picker-indent"></span>
                                  <i data-lucide="folder" class="picker-folder-icon"></i>
                                  <span class="picker-name" @click="selectPath(descendant.path)">{{ descendant.name }}</span>
                                  <code class="picker-path-hint">{{ descendant.path }}</code>
                                </div>
                              </template>
                            </div>
                          </div>
                        </template>
                      </div>
                    </div>
                  </template>
                </div>
              </div>
            </template>

            <div v-if="treePaths.length === 0" class="picker-empty">
              暂无 VFS 目录，请先创建根目录和子文件夹
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button class="btn btn-outline" @click="showPathPicker = false">取消</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.permissions-page {
  padding: 24px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 8px;
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 32px;
  padding: 0 12px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 150ms;
}

.btn-primary {
  background: #3B82F6;
  color: #fff;
  border-color: #3B82F6;
}

.btn-primary:hover { background: #2563EB; }

.btn-outline {
  background: #F3F4F6;
  color: #374151;
  border-color: #E5E7EB;
}

.btn-outline:hover { background: #E5E7EB; }

.btn-icon-left {
  width: 14px;
  height: 14px;
}

/* Alert */
.alert {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 13px;
  margin-bottom: 12px;
}

.alert-error {
  background: #FEF2F2;
  color: #DC2626;
  border: 1px solid #FECACA;
}

.alert-close {
  margin-left: auto;
  background: none;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: inherit;
}

/* Table */
.perm-table-wrapper {
  background: #F3F4F6;
  border: 1px solid #E5E7EB;
  border-radius: 4px;
  overflow-x: auto;
}

.perm-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.perm-table th {
  background: #F9FAFB;
  text-align: left;
  padding: 8px 12px;
  font-weight: 600;
  color: #6B7280;
  border-bottom: 1px solid #E5E7EB;
  white-space: nowrap;
}

.perm-table td {
  padding: 8px 12px;
  border-bottom: 1px solid #F3F4F6;
  color: #374151;
}

.perm-table tr:hover td { background: #F9FAFB; }

.col-user { width: 100px; }
.col-path { width: auto; }
.col-perm { width: 56px; text-align: center; }
.col-actions { width: 80px; text-align: center; }

.empty-row {
  text-align: center;
  color: #9CA3AF;
  padding: 32px 12px !important;
}

.user-badge {
  display: inline-block;
  padding: 1px 6px;
  background: #EFF6FF;
  color: #3B82F6;
  border-radius: 3px;
  font-size: 12px;
  font-weight: 500;
}

.perm-table code {
  font-size: 12px;
  background: #F3F4F6;
  padding: 1px 4px;
  border-radius: 2px;
  color: #6B7280;
}

.perm-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.perm-dot.granted {
  background: #10B981;
}

.perm-dot.denied {
  background: #E5E7EB;
}

.btn-action {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  color: #6B7280;
  border-radius: 3px;
}

.btn-action:hover {
  background: #F3F4F6;
  color: #374151;
}

.btn-action-danger:hover {
  background: #FEF2F2;
  color: #DC2626;
}

.btn-action i {
  width: 14px;
  height: 14px;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal-content {
  background: #fff;
  border: 1px solid #E5E7EB;
  border-radius: 4px;
  width: 480px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 4px 16px rgba(0,0,0,0.12);
}

.path-picker-modal {
  width: 520px;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #E5E7EB;
}

.modal-header h2 {
  font-size: 15px;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  cursor: pointer;
  color: #6B7280;
  padding: 2px;
}

.modal-close:hover { color: #111827; }
.modal-close i { width: 16px; height: 16px; }

.modal-body {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.modal-footer {
  padding: 12px 16px;
  border-top: 1px solid #E5E7EB;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-label {
  font-size: 12px;
  font-weight: 600;
  color: #6B7280;
}

.form-input, .form-select {
  height: 32px;
  padding: 0 8px;
  border: 1px solid #E5E7EB;
  border-radius: 4px;
  font-size: 13px;
  background: #F9FAFB;
  color: #111827;
  outline: none;
}

.form-input:focus, .form-select:focus {
  border-color: #3B82F6;
  background: #fff;
}

.form-hint {
  font-size: 11px;
  color: #9CA3AF;
}

/* Path input row with ref button */
.path-input-row {
  display: flex;
  gap: 6px;
}

.path-input {
  flex: 1;
}

.btn-path-ref {
  flex-shrink: 0;
  white-space: nowrap;
}

/* Visibility tabs */
.visibility-tabs {
  display: flex;
  gap: 0;
  border: 1px solid #E5E7EB;
  border-radius: 4px;
  overflow: hidden;
}

.vis-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 32px;
  border: none;
  background: #F9FAFB;
  font-size: 12px;
  color: #6B7280;
  cursor: pointer;
  transition: all 150ms;
  border-right: 1px solid #E5E7EB;
}

.vis-tab:last-child { border-right: none; }

.vis-tab:hover { background: #F3F4F6; }

.vis-tab.active {
  background: #EFF6FF;
  color: #3B82F6;
  font-weight: 500;
}

.vis-tab i { width: 14px; height: 14px; }

/* Permission checkboxes */
.perm-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}

.perm-check {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #374151;
  cursor: pointer;
}

.perm-check input[type="checkbox"] {
  accent-color: #3B82F6;
}

/* User check list for batch */
.user-check-list {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid #E5E7EB;
  border-radius: 4px;
  padding: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.user-check-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: #F3F4F6;
  border-radius: 3px;
  font-size: 12px;
  cursor: pointer;
}

.user-check-item input[type="checkbox"] {
  accent-color: #3B82F6;
}

/* Path picker */
.picker-search {
  position: relative;
  display: flex;
  align-items: center;
}

.picker-search-icon {
  position: absolute;
  left: 8px;
  width: 14px;
  height: 14px;
  color: #9CA3AF;
  pointer-events: none;
}

.picker-search .form-input {
  padding-left: 28px;
  width: 100%;
}

.picker-tree {
  max-height: 360px;
  overflow-y: auto;
  border: 1px solid #E5E7EB;
  border-radius: 4px;
  background: #F9FAFB;
}

.picker-empty {
  padding: 24px;
  text-align: center;
  color: #9CA3AF;
  font-size: 13px;
}

.picker-node-group {
  margin-bottom: 0;
}

.picker-node {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 32px;
  padding: 0 8px;
  cursor: pointer;
  font-size: 13px;
  color: #374151;
  transition: background 100ms;
  border-left: 4px solid transparent;
}

.picker-node:hover {
  background: #F3F4F6;
}

.picker-node-selected {
  background: #EFF6FF !important;
  border-left-color: #3B82F6;
}

.picker-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: none;
  background: none;
  color: #9CA3AF;
  cursor: pointer;
  flex-shrink: 0;
}

.picker-toggle i {
  transition: transform 150ms;
}

.picker-toggle-expanded i {
  transform: rotate(90deg);
}

.picker-folder-icon {
  width: 14px;
  height: 14px;
  color: #F59E0B;
  flex-shrink: 0;
}

.picker-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 1;
}

.picker-path-hint {
  margin-left: auto;
  font-size: 11px;
  color: #9CA3AF;
  flex-shrink: 0;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-children {
  /* indented by parent */
}

.picker-indent {
  width: 16px;
  flex-shrink: 0;
}
</style>
