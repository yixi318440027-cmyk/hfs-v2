<script setup lang="ts">
import { ref, onMounted, watch, computed, nextTick } from 'vue'
import api from '../api'
import { refreshLucide } from '../utils/lucide'

export interface TreeNode {
  name: string
  path: string
  isDir: boolean
  children?: TreeNode[]
  loaded: boolean
  loading: boolean
  expanded: boolean
}

const props = withDefaults(defineProps<{
  roots: string[]
  currentPath?: string
  publicOnly?: boolean
}>(), {
  currentPath: '',
  publicOnly: false,
})

const emit = defineEmits<{
  'navigate': [path: string]
}>()

const tree = ref<TreeNode[]>([])

const apiPrefix = computed(() => props.publicOnly ? '/public/files' : '/files')

async function loadRoots() {
  tree.value = props.roots.map(name => ({
    name,
    path: '/' + name,
    isDir: true,
    children: [],
    loaded: false,
    loading: false,
    expanded: true,
  }))
  for (const node of tree.value) {
    await loadChildren(node)
  }
  nextTick(() => refreshLucide())
}

async function loadChildren(node: TreeNode) {
  if (node.loaded || node.loading) return
  node.loading = true
  try {
    const res = await api.get(apiPrefix.value + '/list', { params: { path: node.path } })
    if (res.data.ok) {
      const files = res.data.data.files || []
      node.children = files
        .filter((f: any) => f.isDir)
        .map((f: any) => ({
          name: f.name,
          path: node.path + '/' + f.name,
          isDir: true,
          children: [],
          loaded: false,
          loading: false,
          expanded: false,
        }))
      node.loaded = true
    }
  } catch {
    node.children = []
    node.loaded = true
  } finally {
    node.loading = false
    nextTick(() => refreshLucide())
  }
}

async function toggleNode(node: TreeNode) {
  if (!node.expanded) {
    node.expanded = true
    await loadChildren(node)
  } else {
    node.expanded = false
  }
  nextTick(() => refreshLucide())
}

function isActive(node: TreeNode): boolean {
  return props.currentPath === node.path ||
    (props.currentPath && props.currentPath.startsWith(node.path + '/'))
}

function handleNavigate(node: TreeNode) {
  emit('navigate', node.path)
}

onMounted(() => {
  loadRoots()
})

watch(() => props.roots, () => {
  loadRoots()
}, { deep: true })

watch(() => props.currentPath, () => {
  nextTick(() => refreshLucide())
})
</script>

<template>
  <nav class="file-tree">
    <div class="tree-section-title">目录</div>
    <div class="tree-list">
      <template v-for="root in tree" :key="root.path">
        <!-- Root node -->
        <div class="tree-node-group">
          <div
            class="tree-item"
            :class="{ 'tree-item-active': isActive(root) }"
            @click="handleNavigate(root)"
          >
            <button
              class="tree-toggle"
              :class="{ 'tree-toggle-expanded': root.expanded }"
              @click.stop="toggleNode(root)"
            >
              <i data-lucide="chevron-right" style="width:12px;height:12px"></i>
            </button>
            <i data-lucide="folder" class="tree-folder-icon" :class="{ 'tree-folder-active': isActive(root) }"></i>
            <span class="tree-name">{{ root.name }}</span>
          </div>

          <!-- Children -->
          <div v-if="root.expanded && root.children?.length" class="tree-children">
            <template v-for="child in root.children" :key="child.path">
              <div class="tree-node-group">
                <div
                  class="tree-item"
                  :class="{ 'tree-item-active': isActive(child) }"
                  @click="handleNavigate(child)"
                >
                  <button
                    class="tree-toggle"
                    :class="{ 'tree-toggle-expanded': child.expanded }"
                    @click.stop="toggleNode(child)"
                  >
                    <i data-lucide="chevron-right" style="width:12px;height:12px"></i>
                  </button>
                  <i data-lucide="folder" class="tree-folder-icon" :class="{ 'tree-folder-active': isActive(child) }"></i>
                  <span class="tree-name">{{ child.name }}</span>
                </div>

                <!-- Grandchildren -->
                <div v-if="child.expanded && child.children?.length" class="tree-children">
                  <div
                    v-for="grandchild in child.children"
                    :key="grandchild.path"
                    class="tree-item"
                    :class="{ 'tree-item-active': isActive(grandchild) }"
                    @click="handleNavigate(grandchild)"
                  >
                    <span class="tree-indent"></span>
                    <i data-lucide="folder" class="tree-folder-icon" :class="{ 'tree-folder-active': isActive(grandchild) }"></i>
                    <span class="tree-name">{{ grandchild.name }}</span>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>
      </template>
    </div>
  </nav>
</template>

<style scoped>
.file-tree {
  width: 220px;
  height: 100%;
  background: var(--c-sidebar-bg);
  border-right: 1px solid var(--c-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
}

.tree-section-title {
  padding: 12px 16px;
  font-size: 11px;
  font-weight: 600;
  color: var(--c-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  flex-shrink: 0;
}

.tree-list {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 12px;
}

.tree-node-group {
  margin-bottom: 1px;
}

.tree-item {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 32px;
  padding: 0 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  color: var(--c-text-secondary);
  transition: all 150ms ease-in-out;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  border-left: 4px solid transparent;
}

.tree-item:hover {
  background: var(--c-white);
  color: var(--c-text);
}

.tree-item-active {
  background: var(--c-white);
  color: var(--c-text);
  font-weight: 500;
  border-left-color: var(--c-primary);
  border-radius: 0 4px 4px 0;
  padding-left: 4px;
}

.tree-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: none;
  background: none;
  color: var(--c-text-muted);
  cursor: pointer;
  flex-shrink: 0;
  transition: transform 200ms ease-in-out;
}

.tree-toggle-expanded i {
  transform: rotate(90deg);
}

.tree-toggle i {
  display: block;
  transition: transform 200ms ease-in-out;
}

.tree-folder-icon {
  color: var(--c-folder);
  flex-shrink: 0;
}

.tree-folder-active {
  color: var(--c-folder);
}

.tree-name {
  overflow: hidden;
  text-overflow: ellipsis;
}

.tree-children {
  padding-left: 0;
}

.tree-indent {
  width: 16px;
  flex-shrink: 0;
}
</style>
