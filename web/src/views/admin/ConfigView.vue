<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../../api'

interface Config {
  port: number
  dataDir: string
  logLevel: string
  vfsRoots: string
}

const config = ref<Config>({
  port: 8080,
  dataDir: '',
  logLevel: 'info',
  vfsRoots: '',
})

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const successMsg = ref('')

async function loadConfig() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/admin/config')
    if (res.data.ok) {
      const data = res.data.data || res.data
      config.value = {
        port: data.port ?? 8080,
        dataDir: data.dataDir || data.data_dir || '',
        logLevel: data.logLevel || data.log_level || 'info',
        vfsRoots: data.vfsRoots || data.vfs_roots || '',
      }
    } else {
      error.value = '加载配置失败'
    }
  } catch {
    error.value = '加载配置失败'
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  error.value = ''
  successMsg.value = ''
  try {
    const res = await api.put('/admin/config', config.value)
    if (res.data.ok) {
      successMsg.value = '配置保存成功'
      setTimeout(() => { successMsg.value = '' }, 3000)
    } else {
      error.value = res.data.error || '保存失败'
    }
  } catch {
    error.value = '保存失败'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">系统配置</h1>
        <p class="page-desc">管理服务器运行参数和 VFS 挂载点</p>
      </div>
      <button class="btn btn-primary" @click="handleSave" :disabled="saving">
        <i data-lucide="save" style="width:14px;height:14px"></i>
        {{ saving ? '保存中...' : '保存配置' }}
      </button>
    </div>

    <div v-if="successMsg" class="alert alert-success">{{ successMsg }}</div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else class="config-form">
      <div class="form-group">
        <label for="port">端口</label>
        <input
          id="port"
          v-model.number="config.port"
          type="number"
          class="form-input"
          placeholder="8080"
        />
        <span class="form-hint">HTTP 服务监听端口</span>
      </div>
      <div class="form-group">
        <label for="dataDir">数据目录</label>
        <input
          id="dataDir"
          v-model="config.dataDir"
          type="text"
          class="form-input"
          placeholder="./data"
        />
        <span class="form-hint">数据库和配置文件的存储路径</span>
      </div>
      <div class="form-group">
        <label for="logLevel">日志级别</label>
        <select id="logLevel" v-model="config.logLevel" class="form-input">
          <option value="debug">debug</option>
          <option value="info">info</option>
          <option value="warn">warn</option>
          <option value="error">error</option>
        </select>
      </div>
      <div class="form-group">
        <label for="vfsRoots">VFS 根目录配置</label>
        <textarea
          id="vfsRoots"
          v-model="config.vfsRoots"
          class="form-textarea"
          rows="6"
          placeholder="输入 YAML 格式的 VFS 根目录配置"
        ></textarea>
        <span class="form-hint">使用 YAML 格式定义虚拟文件系统挂载点</span>
      </div>
    </div>
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

.config-form {
  background: var(--c-white);
  border: 1px solid var(--c-border);
  border-radius: 4px;
  padding: 24px;
  max-width: 800px;
}
</style>
