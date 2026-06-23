<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import api from '../../api'
import { refreshLucide } from '../../utils/lucide'

interface DiskInfo {
  mountPoint: string
  total: number
  free: number
  used: number
  label: string
}

interface ConnInfo {
  ip: string
  username: string
  path: string
  method: string
  connected: string
}

const diskUsage = ref<DiskInfo[]>([])
const connections = ref<ConnInfo[]>([])
const userCount = ref(0)
const connectionCount = ref(0)

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

async function loadDashboard() {
  try {
    const [diskRes, connRes, usersRes] = await Promise.all([
      api.get('/admin/disk-usage'),
      api.get('/admin/connections'),
      api.get('/admin/users'),
    ])
    if (diskRes.data.ok) diskUsage.value = diskRes.data.data || []
    if (connRes.data.ok) {
      connections.value = connRes.data.data || []
      connectionCount.value = connections.value.length
      const ips = new Set(connections.value.map(c => c.ip))
      userCount.value = ips.size
    }
    if (usersRes.data.ok) {
      // Users count already handled by userCount from connections
    }
    nextTick(() => refreshLucide())
  } catch {
    // Silently fail
  }
}

onMounted(() => {
  loadDashboard()
})
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">仪表盘</h1>
        <p class="page-desc">系统运行概况</p>
      </div>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon-wrap stat-icon-storage">
          <i data-lucide="hard-drive" style="width:20px;height:20px"></i>
        </div>
        <div class="stat-body">
          <div class="stat-label">在线 IP</div>
          <div class="stat-value">{{ userCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon-wrap stat-icon-users">
          <i data-lucide="users" style="width:20px;height:20px"></i>
        </div>
        <div class="stat-body">
          <div class="stat-label">活跃连接</div>
          <div class="stat-value">{{ connectionCount }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon-wrap stat-icon-traffic">
          <i data-lucide="activity" style="width:20px;height:20px"></i>
        </div>
        <div class="stat-body">
          <div class="stat-label">存储卷</div>
          <div class="stat-value">{{ diskUsage.length }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon-wrap stat-icon-files">
          <i data-lucide="files" style="width:20px;height:20px"></i>
        </div>
        <div class="stat-body">
          <div class="stat-label">VFS 根目录</div>
          <div class="stat-value">{{ diskUsage.length }}</div>
        </div>
      </div>
    </div>

    <!-- Disk Usage -->
    <div v-if="diskUsage.length > 0" class="section">
      <h2 class="section-title">磁盘空间</h2>
      <div class="disk-grid">
        <div v-for="disk in diskUsage" :key="disk.mountPoint" class="disk-card">
          <div class="disk-header">
            <i data-lucide="hard-drive" style="width:16px;height:16px"></i>
            <span class="disk-label">{{ disk.label }}</span>
            <span class="disk-path">{{ disk.mountPoint }}</span>
          </div>
          <div class="disk-bar-wrap">
            <div class="disk-bar">
              <div
                class="disk-bar-fill"
                :style="{ width: disk.total > 0 ? ((disk.used / disk.total) * 100).toFixed(1) + '%' : '0%' }"
              ></div>
            </div>
          </div>
          <div class="disk-stats">
            <span>已用 {{ formatSize(disk.used) }}</span>
            <span>可用 {{ formatSize(disk.free) }}</span>
            <span>总计 {{ formatSize(disk.total) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Active Connections -->
    <div v-if="connections.length > 0" class="section">
      <h2 class="section-title">活跃连接</h2>
      <div class="conn-table-wrap">
        <table class="conn-table">
          <thead>
            <tr>
              <th>IP</th>
              <th>用户</th>
              <th>请求路径</th>
              <th>方法</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(c, idx) in connections" :key="idx" :class="{ 'row-stripe': idx % 2 === 1 }">
              <td>{{ c.ip }}</td>
              <td>{{ c.username || '--' }}</td>
              <td>{{ c.path }}</td>
              <td>{{ c.method }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-page {
  padding: 24px 28px;
}

.page-header {
  margin-bottom: 24px;
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

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.stat-card {
  background: var(--c-white);
  border: 1px solid var(--c-border);
  border-radius: 4px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: box-shadow 200ms ease-in-out;
}

.stat-card:hover {
  box-shadow: var(--shadow-card);
}

.stat-icon-wrap {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon-storage { background: var(--c-primary-light); color: var(--c-primary); }
.stat-icon-users   { background: var(--c-success-light); color: var(--c-success); }
.stat-icon-traffic { background: var(--c-warning-light); color: var(--c-warning); }
.stat-icon-files   { background: #F5F3FF; color: #7C3AED; }

.stat-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  font-size: 12px;
  color: var(--c-text-muted);
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--c-text);
  line-height: 1.2;
}

/* Section */
.section {
  margin-bottom: 28px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--c-text);
  margin: 0 0 12px;
}

/* Disk cards */
.disk-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}

.disk-card {
  background: var(--c-white);
  border: 1px solid var(--c-border);
  border-radius: 4px;
  padding: 16px;
}

.disk-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--c-text);
}

.disk-label {
  font-weight: 600;
}

.disk-path {
  font-size: 11px;
  color: var(--c-text-muted);
  margin-left: auto;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disk-bar-wrap {
  margin-bottom: 8px;
}

.disk-bar {
  height: 6px;
  background: var(--c-bg);
  border-radius: 3px;
  overflow: hidden;
}

.disk-bar-fill {
  height: 100%;
  background: var(--c-primary);
  border-radius: 3px;
  min-width: 2px;
  transition: width 500ms ease;
}

.disk-stats {
  display: flex;
  gap: 16px;
  font-size: 11px;
  color: var(--c-text-muted);
}

/* Connections table */
.conn-table-wrap {
  background: var(--c-white);
  border: 1px solid var(--c-border);
  border-radius: 4px;
  overflow: hidden;
}

.conn-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.conn-table th {
  background: var(--c-header-bg);
  padding: 0 12px;
  height: 32px;
  text-align: left;
  font-weight: 500;
  font-size: 12px;
  color: var(--c-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.025em;
  border-bottom: 1px solid var(--c-border);
}

.conn-table td {
  padding: 0 12px;
  height: 36px;
  border-bottom: 1px solid var(--c-border-light);
}

.row-stripe {
  background: var(--c-row-stripe);
}
</style>
