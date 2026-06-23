<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { refreshLucide } from '../utils/lucide'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''

  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }

  loading.value = true
  const success = await authStore.login(username.value, password.value)
  loading.value = false

  if (success) {
    router.push('/')
  } else {
    error.value = '用户名或密码错误'
  }
}

onMounted(() => {
  nextTick(() => refreshLucide())
})
</script>

<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <i data-lucide="hard-drive" style="width:28px;height:28px;color:var(--c-primary)"></i>
        <h1>hfs-v2</h1>
        <p>轻量级文件服务器</p>
      </div>
      <form class="login-form" @submit.prevent="handleLogin">
        <div v-if="error" class="alert alert-error">
          <i data-lucide="alert-circle" style="width:14px;height:14px"></i>
          {{ error }}
        </div>
        <div class="form-group">
          <label for="username">用户名</label>
          <div class="input-wrap">
            <i data-lucide="user" style="width:14px;height:14px" class="input-icon"></i>
            <input
              id="username"
              v-model="username"
              type="text"
              placeholder="请输入用户名"
              autocomplete="username"
            />
          </div>
        </div>
        <div class="form-group">
          <label for="password">密码</label>
          <div class="input-wrap">
            <i data-lucide="lock" style="width:14px;height:14px" class="input-icon"></i>
            <input
              id="password"
              v-model="password"
              type="password"
              placeholder="请输入密码"
              autocomplete="current-password"
            />
          </div>
        </div>
        <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
          <i v-if="!loading" data-lucide="log-in" style="width:14px;height:14px"></i>
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: var(--c-bg);
}

.login-card {
  background: var(--c-white);
  border: 1px solid var(--c-border);
  border-radius: 8px;
  padding: 40px;
  width: 380px;
  box-shadow: var(--shadow-card);
}

.login-header {
  text-align: center;
  margin-bottom: 28px;
}

.login-header h1 {
  font-size: 20px;
  font-weight: 700;
  color: var(--c-text);
  margin: 8px 0 4px;
}

.login-header p {
  font-size: 13px;
  color: var(--c-text-muted);
  margin: 0;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.alert {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 16px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 13px;
  color: var(--c-text-secondary);
  margin-bottom: 4px;
  font-weight: 500;
}

.input-wrap {
  display: flex;
  align-items: center;
  border: 1px solid var(--c-border);
  border-radius: 4px;
  padding: 0 10px;
  transition: border-color 200ms, box-shadow 200ms;
}

.input-wrap:focus-within {
  border-color: var(--c-primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.input-icon {
  color: var(--c-text-muted);
  flex-shrink: 0;
  margin-right: 8px;
}

.input-wrap input {
  flex: 1;
  border: none;
  outline: none;
  height: 36px;
  font-size: 14px;
  color: var(--c-text);
  background: transparent;
  font-family: inherit;
}

.input-wrap input::placeholder {
  color: var(--c-text-placeholder);
}

.login-btn {
  width: 100%;
  height: 36px;
  margin-top: 4px;
  gap: 6px;
}
</style>
