<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const showAdminMenu = ref(false)

function toggleAdminMenu() {
  showAdminMenu.value = !showAdminMenu.value
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <nav class="navbar">
    <div class="navbar-left">
      <router-link to="/" class="logo">hfs-v2</router-link>
    </div>
    <div class="navbar-right" v-if="authStore.isLoggedIn">
      <span class="username">{{ authStore.user?.username }}</span>
      <div class="admin-dropdown" v-if="authStore.isAdmin">
        <button class="dropdown-btn" @click="toggleAdminMenu">
          管理
          <span class="arrow" :class="{ open: showAdminMenu }">&#9662;</span>
        </button>
        <div class="dropdown-menu" v-if="showAdminMenu" @click="showAdminMenu = false">
          <router-link to="/admin/dashboard">仪表盘</router-link>
          <router-link to="/admin/users">用户管理</router-link>
          <router-link to="/admin/config">系统配置</router-link>
          <router-link to="/admin/logs">日志查看</router-link>
        </div>
      </div>
      <button class="logout-btn" @click="handleLogout">退出</button>
    </div>
  </nav>
</template>

<style scoped>
.navbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  height: 56px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.navbar-left .logo {
  font-size: 20px;
  font-weight: 700;
  color: #1677ff;
  text-decoration: none;
}

.navbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.username {
  font-size: 14px;
  color: #333;
}

.admin-dropdown {
  position: relative;
}

.dropdown-btn {
  background: none;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  padding: 4px 12px;
  font-size: 14px;
  cursor: pointer;
  color: #333;
  display: flex;
  align-items: center;
  gap: 4px;
}

.dropdown-btn:hover {
  border-color: #1677ff;
  color: #1677ff;
}

.arrow {
  font-size: 10px;
  transition: transform 0.2s;
}

.arrow.open {
  transform: rotate(180deg);
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 4px;
  background: #fff;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  min-width: 140px;
  z-index: 100;
}

.dropdown-menu a {
  display: block;
  padding: 8px 16px;
  font-size: 14px;
  color: #333;
  text-decoration: none;
}

.dropdown-menu a:hover {
  background: #f5f5f5;
  color: #1677ff;
}

.logout-btn {
  background: none;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  padding: 4px 12px;
  font-size: 14px;
  cursor: pointer;
  color: #666;
}

.logout-btn:hover {
  border-color: #ff4d4f;
  color: #ff4d4f;
}
</style>
