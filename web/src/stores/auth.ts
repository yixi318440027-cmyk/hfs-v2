import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

interface User {
  username: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(username: string, password: string): Promise<boolean> {
    try {
      const res = await axios.post('/api/auth/login', { username, password })
      if (res.data.ok) {
        token.value = res.data.token
        user.value = res.data.user
        return true
      }
      return false
    } catch {
      return false
    }
  }

  function logout() {
    token.value = null
    user.value = null
  }

  function checkAuth(): boolean {
    return !!token.value
  }

  return {
    user,
    token,
    isLoggedIn,
    isAdmin,
    login,
    logout,
    checkAuth,
  }
})
