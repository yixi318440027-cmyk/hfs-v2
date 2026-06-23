<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from './stores/auth'
import NavBar from './components/NavBar.vue'
import FileTree from './components/FileTree.vue'
import AdminSidebar from './components/AdminSidebar.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// Fetch public roots for the sidebar when logged in
const publicRootNames = ref<string[]>([])
const fileRootNames = ref<string[]>(['Files'])

// Determine if current route is admin
const isAdminRoute = computed(() => route.path.startsWith('/admin'))

// Determine if current route is browse (file management)
const isBrowseRoute = computed(() => route.path === '/' || route.path === '/files')

// Determine if current route is login
const isLoginRoute = computed(() => route.path === '/login')

// Should show sidebar
const showSidebar = computed(() => {
  if (isLoginRoute.value) return false
  if (isAdminRoute.value) return authStore.isAdmin
  return authStore.isLoggedIn || isBrowseRoute.value
})

// Should show file tree sidebar (only for browse routes)
const showFileTree = computed(() => {
  return isBrowseRoute.value
})

// Should show admin sidebar (only for admin routes)
const showAdminSidebar = computed(() => {
  return isAdminRoute.value && authStore.isAdmin
})

// Current browse path for file tree highlight
const currentBrowsePath = ref('/Files')

function onFileTreeNavigate(path: string) {
  currentBrowsePath.value = path
  // Emit a custom event that BrowseView can listen to, or use a shared state
  window.dispatchEvent(new CustomEvent('filetree-navigate', { detail: { path } }))
}

// Listen for path changes from BrowseView
function onBrowsePathChange(e: Event) {
  const detail = (e as CustomEvent).detail
  if (detail?.path) {
    currentBrowsePath.value = detail.path
  }
}

onMounted(() => {
  window.addEventListener('browse-path-change', onBrowsePathChange)
})
</script>

<template>
  <div id="app-root">
    <!-- Top nav bar - always visible -->
    <NavBar />

    <!-- Main layout: sidebar + content -->
    <div class="app-body">
      <!-- Left Sidebar -->
      <aside class="app-sidebar" v-if="showSidebar">
        <FileTree
          v-if="showFileTree"
          :roots="fileRootNames"
          :current-path="currentBrowsePath"
          @navigate="onFileTreeNavigate"
        />
        <AdminSidebar v-else-if="showAdminSidebar" />
      </aside>

      <!-- Main content area -->
      <main class="main-content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
#app-root {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--c-bg);
}

.app-body {
  display: flex;
  flex: 1;
  min-height: calc(100vh - 48px);
  overflow: hidden;
}

.app-sidebar {
  width: 220px;
  flex-shrink: 0;
  overflow-y: auto;
}

.main-content {
  flex: 1;
  overflow-y: auto;
  min-width: 0;
}
</style>
