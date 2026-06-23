import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface User {
  username: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function login(_username: string, _password: string): boolean {
    // Mock login: accept any credentials, return admin role
    token.value = 'mock-token-sprint-0'
    user.value = {
      username: _username,
      role: 'admin',
    }
    return true
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
