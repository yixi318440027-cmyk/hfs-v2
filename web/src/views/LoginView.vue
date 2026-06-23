<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

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
</script>

<template>
  <div class="login-container">
    <form class="login-form" @submit.prevent="handleLogin">
      <h1>hfs-v2 登录</h1>
      <div v-if="error" class="error">{{ error }}</div>
      <label>
        用户名
        <input v-model="username" type="text" placeholder="请输入用户名" />
      </label>
      <label>
        密码
        <input v-model="password" type="password" placeholder="请输入密码" />
      </label>
      <button type="submit" :disabled="loading">
        {{ loading ? '登录中...' : '登录' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f0f2f5;
}

.login-form {
  background: #fff;
  padding: 40px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  width: 360px;
}

h1 {
  text-align: center;
  margin-bottom: 24px;
  font-size: 22px;
  color: #1a1a1a;
}

.error {
  background: #fff2f0;
  border: 1px solid #ffccc7;
  color: #ff4d4f;
  padding: 8px 12px;
  border-radius: 4px;
  margin-bottom: 16px;
  font-size: 14px;
}

label {
  display: block;
  margin-bottom: 16px;
  font-size: 14px;
  color: #333;
}

input {
  display: block;
  width: 100%;
  margin-top: 6px;
  padding: 8px 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 14px;
  box-sizing: border-box;
}

input:focus {
  border-color: #4096ff;
  outline: none;
  box-shadow: 0 0 0 2px rgba(64, 150, 255, 0.2);
}

button {
  width: 100%;
  padding: 10px;
  background: #1677ff;
  color: #fff;
  border: none;
  border-radius: 4px;
  font-size: 16px;
  cursor: pointer;
  margin-top: 8px;
}

button:hover:not(:disabled) {
  background: #4096ff;
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
