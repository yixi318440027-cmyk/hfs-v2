<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import api from '../../api'
import { refreshLucide } from '../../utils/lucide'

interface LogEntry {
  time: string
  type: string
  detail: string
}

const logs = ref<LogEntry[]>([])
const loading = ref(false)
const error = ref('')

async function loadLogs() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/admin/logs')
    if (res.data.ok) {
      logs.value = res.data.data?.logs || res.data.logs || []
    } else {
      error.value = '加载日志失败'
    }
  } catch {
    error.value = '加载日志失败'
  } finally {
    loading.value = false
    nextTick(() => refreshLucide())
  }
}

function formatTime(dateStr: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN')
}

onMounted(() => {
  loadLogs()
})
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">日志查看</h1>
        <p class="page-desc">系统操作日志和运行记录</p>
      </div>
      <button class="btn" @click="loadLogs" :disabled="loading">
        <i data-lucide="refresh-cw" style="width:14px;height:14px"></i>
        {{ loading ? '刷新中...' : '刷新' }}
      </button>
    </div>

    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="table-wrapper">
      <table class="data-table" v-if="!loading">
        <thead>
          <tr>
            <th class="col-time">时间</th>
            <th class="col-type">类型</th>
            <th>详情</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="logs.length === 0">
            <td colspan="3" class="empty">暂无日志数据</td>
          </tr>
          <tr v-for="(log, index) in logs" :key="index">
            <td>{{ formatTime(log.time) }}</td>
            <td>
              <span class="log-type" :class="'log-' + log.type.toLowerCase()">
                {{ log.type }}
              </span>
            </td>
            <td class="col-detail">{{ log.detail }}</td>
          </tr>
        </tbody>
      </table>
      <div v-else class="loading">加载中...</div>
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

.col-time { width: 170px; }
.col-type { width: 90px; }
.col-detail { word-break: break-all; }

.log-type {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.log-debug { background: var(--c-bg); color: var(--c-text-muted); }
.log-info  { background: var(--c-primary-light); color: var(--c-primary); }
.log-warn  { background: var(--c-warning-light); color: var(--c-warning); }
.log-error { background: var(--c-danger-light); color: var(--c-danger); }
</style>
