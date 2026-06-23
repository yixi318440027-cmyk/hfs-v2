import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'browse',
    // No requiresAuth — the view itself checks login state
    component: () => import('../views/BrowseView.vue'),
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
  },
  {
    path: '/admin/dashboard',
    name: 'admin-dashboard',
    component: () => import('../views/admin/DashboardView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/admin/users',
    name: 'admin-users',
    component: () => import('../views/admin/UsersView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/admin/config',
    name: 'admin-config',
    component: () => import('../views/admin/ConfigView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/admin/logs',
    name: 'admin-logs',
    component: () => import('../views/admin/LogsView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/admin/permissions',
    name: 'admin-permissions',
    component: () => import('../views/admin/PermissionsView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.meta.requiresAuth as boolean | undefined
  const requiresAdmin = to.meta.requiresAdmin as boolean | undefined

  if (requiresAuth && !authStore.token) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  if (requiresAdmin && authStore.user?.role !== 'admin') {
    next({ name: 'browse' })
    return
  }

  next()
})

export default router
