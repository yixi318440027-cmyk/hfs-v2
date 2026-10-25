import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

interface User {
  username: string
  role: string
}

const TOKEN_KEY = 'hfs_token'
const USER_KEY = 'hfs_user'

function loadPersisted(): { token: string | null; user: User | null } {
  try {
    const t = localStorage.getItem(TOKEN_KEY)
    const u = localStorage.getItem(USER_KEY)
    return {
      token: t || null,
      user: u ? JSON.parse(u) : null,
    }
  } catch {
    return { token: null, user: null }
  }
}

function persist(token: string | null, user: User | null) {
  try {
    if (token) {
      localStorage.setItem(TOKEN_KEY, token)
    } else {
      localStorage.removeItem(TOKEN_KEY)
    }
    if (user) {
      localStorage.setItem(USER_KEY, JSON.stringify(user))
    } else {
      localStorage.removeItem(USER_KEY)
    }
  } catch {
    // localStorage unavailable (private browsing, etc.) — gracefully degrade
  }
}

export const useAuthStore = defineStore('auth', () => {
  const persisted = loadPersisted()
  const user = ref<User | null>(persisted.user)
  const token = ref<string | null>(persisted.token)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(username: string, password: string): Promise<boolean> {
    try {
      const res = await axios.post('/api/auth/login', { username, password })
      if (res.data.ok) {
        token.value = res.data.token
        user.value = res.data.user
        persist(token.value, user.value)
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
    persist(null, null)
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
