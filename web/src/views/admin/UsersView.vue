<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import api from '../../api'
import { refreshLucide } from '../../utils/lucide'

interface User {
  id: number
  username: string
  role: string
  enabled: boolean
  created_at: string
}

const users = ref<User[]>([])
const loading = ref(false)
const error = ref('')

const showCreateModal = ref(false)
const creating = ref(false)
const createError = ref('')
const createForm = ref({
  username: '',
  password: '',
  role: 'user',
  enabled: true,
})

async function loadUsers() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/admin/users')
    if (res.data.ok) {
      users.value = res.data.data?.users || res.data.users || []
    }
  } catch {
    error.value = '加载用户列表失败'
  } finally {
    loading.value = false
    nextTick(() => refreshLucide())
  }
}

function openCreateModal() {
  createForm.value = { username: '', password: '', role: 'user', enabled: true }
  createError.value = ''
  showCreateModal.value = true
  nextTick(() => refreshLucide())
}

function closeCreateModal() {
  showCreateModal.value = false
}

async function handleCreate() {
  if (!createForm.value.username.trim() || !createForm.value.password.trim()) {
    createError.value = '用户名和密码不能为空'
    return
  }
  creating.value = true
  createError.value = ''
  try {
    const res = await api.post('/admin/users', createForm.value)
    if (res.data.ok) {
      closeCreateModal()
      loadUsers()
    } else {
      createError.value = res.data.error || '创建失败'
    }
  } catch {
    createError.value = '创建失败'
  } finally {
    creating.value = false
  }
}

async function handleDelete(user: User) {
  if (!confirm(`确定要删除用户 "${user.username}" 吗？`)) return
  try {
    const res = await api.delete('/admin/users', { params: { id: user.id } })
    if (res.data.ok) {
      loadUsers()
    } else {
      alert('删除失败')
    }
  } catch {
    alert('删除失败')
  }
}

function formatTime(dateStr: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN')
}

onMounted(() => {
  loadUsers()
})
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">用户管理</h1>
        <p class="page-desc">管理系统用户和权限</p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">
        <i data-lucide="user-plus" style="width:14px;height:14px"></i>
        新建用户
      </button>
    </div>

    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="table-wrapper">
      <table class="data-table" v-if="!loading">
        <thead>
          <tr>
            <th>ID</th>
            <th>用户名</th>
            <th>角色</th>
            <th>状态</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="users.length === 0">
            <td colspan="6" class="empty">暂无用户数据</td>
          </tr>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.id }}</td>
            <td>{{ user.username }}</td>
            <td>
              <span class="badge" :class="user.role === 'admin' ? 'badge-admin' : 'badge-user'">
                {{ user.role === 'admin' ? '管理员' : '普通用户' }}
              </span>
            </td>
            <td>
              <span class="badge" :class="user.enabled ? 'badge-enabled' : 'badge-disabled'">
                {{ user.enabled ? '启用' : '禁用' }}
              </span>
            </td>
            <td>{{ formatTime(user.created_at) }}</td>
            <td>
              <button class="btn btn-sm btn-danger" @click="handleDelete(user)">
                <i data-lucide="trash-2" style="width:12px;height:12px"></i>
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="loading">加载中...</div>
    </div>

    <!-- Create modal -->
    <Teleport to="body">
      <div v-if="showCreateModal" class="modal-overlay" @click.self="closeCreateModal">
        <div class="modal">
          <h3 class="modal-title">新建用户</h3>
          <div v-if="createError" class="alert alert-error">{{ createError }}</div>
          <div class="form-group">
            <label>用户名</label>
            <input v-model="createForm.username" type="text" class="form-input" placeholder="输入用户名" />
          </div>
          <div class="form-group">
            <label>密码</label>
            <input v-model="createForm.password" type="password" class="form-input" placeholder="输入密码" />
          </div>
          <div class="form-group">
            <label>角色</label>
            <select v-model="createForm.role" class="form-input">
              <option value="user">普通用户</option>
              <option value="admin">管理员</option>
            </select>
          </div>
          <div class="form-group form-check">
            <label>
              <input v-model="createForm.enabled" type="checkbox" />
              启用账号
            </label>
          </div>
          <div class="modal-actions">
            <button class="btn" @click="closeCreateModal" :disabled="creating">取消</button>
            <button class="btn btn-primary" @click="handleCreate" :disabled="creating">
              {{ creating ? '创建中...' : '创建' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.admin-page {
  padding: 24px 28px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
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
</style>
