<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const adminMenus = [
  { label: '仪表盘', icon: 'layout-dashboard', path: '/admin/dashboard' },
  { label: '用户管理', icon: 'users', path: '/admin/users' },
  { label: '权限管理', icon: 'shield', path: '/admin/permissions' },
  { label: '系统配置', icon: 'settings', path: '/admin/config' },
  { label: '日志查看', icon: 'scroll-text', path: '/admin/logs' },
]

function isActive(path: string): boolean {
  return route.path === path
}

function navigateTo(path: string) {
  router.push(path)
}
</script>

<template>
  <nav class="admin-sidebar">
    <div class="sidebar-section-title">管理后台</div>
    <div class="sidebar-nav">
      <div
        v-for="menu in adminMenus"
        :key="menu.path"
        class="nav-item"
        :class="{ active: isActive(menu.path) }"
        @click="navigateTo(menu.path)"
      >
        <i :data-lucide="menu.icon" class="nav-icon"></i>
        <span class="nav-label">{{ menu.label }}</span>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.admin-sidebar {
  width: 220px;
  height: 100%;
  background: var(--c-sidebar-bg);
  border-right: 1px solid var(--c-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-section-title {
  padding: 12px 16px;
  font-size: 11px;
  font-weight: 600;
  color: var(--c-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--c-border);
  flex-shrink: 0;
}

.sidebar-nav {
  flex: 1;
  padding: 8px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  color: var(--c-text-secondary);
  transition: all 150ms ease-in-out;
  margin-bottom: 1px;
  border-left: 4px solid transparent;
}

.nav-item:hover {
  background: var(--c-white);
  color: var(--c-text);
}

.nav-item.active {
  background: var(--c-white);
  color: var(--c-text);
  font-weight: 500;
  border-left-color: var(--c-primary);
  border-radius: 0 4px 4px 0;
  padding-left: 4px;
}

.nav-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.nav-label {
  white-space: nowrap;
}
</style>
