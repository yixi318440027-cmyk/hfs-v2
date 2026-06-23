<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { refreshLucide } from '../utils/lucide'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const showAdminMenu = ref(false)

function toggleAdminMenu() {
  showAdminMenu.value = !showAdminMenu.value
}

function handleLogout() {
  showAdminMenu.value = false
  authStore.logout()
  router.push('/')
}

function closeAdminMenu() {
  showAdminMenu.value = false
}

onMounted(() => {
  refreshLucide()
})
</script>

<template>
  <header class="navbar" @click="closeAdminMenu">
    <div class="navbar-inner">
      <!-- Logo -->
      <router-link to="/" class="logo">hfs-v2</router-link>

      <div class="navbar-actions">
        <!-- Not logged in -->
        <template v-if="!authStore.isLoggedIn">
          <router-link to="/login" class="btn btn-ghost">
            <i data-lucide="log-in" style="width:14px;height:14px"></i>
            <span>登录</span>
          </router-link>
        </template>

        <!-- Logged in -->
        <template v-else>
          <router-link
            to="/"
            class="nav-link"
            :class="{ active: route.path === '/' }"
          >
            文件管理
          </router-link>

          <!-- Admin dropdown -->
          <div class="admin-dropdown" v-if="authStore.isAdmin" @click.stop>
            <button class="dropdown-btn" @click="toggleAdminMenu">
              管理
              <i data-lucide="chevron-down" style="width:12px;height:12px" :class="{ rotated: showAdminMenu }"></i>
            </button>
            <div class="dropdown-menu" v-if="showAdminMenu">
              <router-link to="/admin/dashboard" @click="showAdminMenu = false">
                <i data-lucide="layout-dashboard" style="width:14px;height:14px"></i>
                仪表盘
              </router-link>
              <router-link to="/admin/users" @click="showAdminMenu = false">
                <i data-lucide="users" style="width:14px;height:14px"></i>
                用户管理
              </router-link>
              <router-link to="/admin/config" @click="showAdminMenu = false">
                <i data-lucide="settings" style="width:14px;height:14px"></i>
                系统配置
              </router-link>
              <router-link to="/admin/logs" @click="showAdminMenu = false">
                <i data-lucide="scroll-text" style="width:14px;height:14px"></i>
                日志查看
              </router-link>
            </div>
          </div>

          <span class="username">{{ authStore.user?.username }}</span>
          <button class="btn btn-sm" @click="handleLogout">退出</button>
        </template>
      </div>
    </div>
  </header>
</template>

<style scoped>
.navbar {
  height: 48px;
  background: var(--c-white);
  border-bottom: 1px solid var(--c-border);
  flex-shrink: 0;
  z-index: 20;
}

.navbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  padding: 0 20px;
  max-width: 100%;
}

.logo {
  font-size: 16px;
  font-weight: 700;
  color: var(--c-primary);
  text-decoration: none;
  letter-spacing: -0.01em;
  flex-shrink: 0;
}

.logo:hover {
  color: var(--c-primary-hover);
  text-decoration: none;
}

.navbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.nav-link {
  font-size: 13px;
  color: var(--c-text-secondary);
  text-decoration: none;
  padding: 2px 0;
  border-bottom: 2px solid transparent;
  transition: all 200ms ease-in-out;
}

.nav-link:hover {
  color: var(--c-primary);
  text-decoration: none;
}

.nav-link.active {
  color: var(--c-primary);
  border-bottom-color: var(--c-primary);
}

.username {
  font-size: 13px;
  color: var(--c-text-muted);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-dropdown {
  position: relative;
}

.dropdown-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--c-border);
  border-radius: 4px;
  background: var(--c-white);
  color: var(--c-text-secondary);
  font-size: 13px;
  font-family: inherit;
  cursor: pointer;
  transition: all 200ms ease-in-out;
}

.dropdown-btn:hover {
  border-color: var(--c-primary-border);
  color: var(--c-primary);
  background: var(--c-primary-light);
}

.dropdown-btn i {
  transition: transform 200ms ease-in-out;
}

.dropdown-btn i.rotated {
  transform: rotate(180deg);
}

.dropdown-menu {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  background: var(--c-white);
  border: 1px solid var(--c-border);
  border-radius: 4px;
  box-shadow: var(--shadow-menu);
  min-width: 160px;
  z-index: 100;
  padding: 4px 0;
}

.dropdown-menu a {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 12px;
  font-size: 13px;
  color: var(--c-text);
  text-decoration: none;
  transition: background 150ms;
}

.dropdown-menu a:hover {
  background: var(--c-bg);
  text-decoration: none;
}
</style>
