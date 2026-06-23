import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const api = axios.create({ baseURL: '/api' })

api.interceptors.request.use(config => {
  const authStore = useAuthStore()
  if (authStore.token) {
    config.headers.Authorization = `Bearer ${authStore.token}`
  }
  return config
})

api.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      const authStore = useAuthStore()
      // Only redirect to login if the user was previously authenticated.
      // Public API 401s are expected (e.g. accessing non-public roots without login).
      if (authStore.isLoggedIn) {
        authStore.logout()
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

/** Get the current auth token (for use in non-axios requests like downloads). */
export function getAuthToken(): string | null {
  const authStore = useAuthStore()
  return authStore.token
}

export default api
