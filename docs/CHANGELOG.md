# Changelog

## v0.6.0 (2026-06-23)

### 文件权限系统

- **6 种权限粒度**：can_see / can_read / can_list / can_upload / can_delete / can_archive
- **per-path × per-user 授权**：每个用户对每个 VFS 路径独立配置权限
- **子路径继承**：子目录自动继承父路径的权限配置
- **Admin 豁免**：admin 角色始终拥有全部权限
- **默认只读**：新用户默认 can_see/can_read/can_list=true，upload/delete/archive=false
- **管理后台**：新增权限管理页面，支持单条配置、批量授权、可见性级别快捷入口
- **管理 API**：`/api/admin/permissions` — CRUD + batch
- **403 提示**：权限拒绝时返回具体缺失的权限类型

### 变更文件

- `src/internal/permission/permission.go` — 权限检查引擎（新增）
- `src/internal/db/db.go` — permissions 表迁移
- `src/internal/server/server.go` — 注入 permission.Engine
- `src/internal/server/routes.go` — 8 个 VFS handler 注入权限检查 + 4 个权限 API handler
- `web/src/views/admin/PermissionsView.vue` — 权限管理页面（新增）
- `web/src/components/AdminSidebar.vue` — 新增"权限管理"菜单
- `web/src/router/index.ts` — 新增权限管理路由
- `docs/PRD-v0.6.0-file-permissions.md` — PRD 文档

## v0.5.1 (2026-06-23)

### Grid View 视觉重构

- **卡片重塑**：背景 #F3F4F6、1px 边框 #E5E7EB、4px 圆角、弱阴影 `0 1px 2px rgba(0,0,0,0.05)`
- **悬停效果**：hover 时背景提亮至 #FFF，阴影加深为 `0 4px 6px rgba(0,0,0,0.08)`，`transition-all duration-200`
- **图标规范化**：文件夹 `folder` (#F59E0B 黄色)、文件 `file-text` (#6B7280 灰色)，彻底移除 Emoji
- **视图切换**：激活按钮浅蓝背景 #EFF6FF + 蓝色边框 #3B82F6
- **搜索框优化**：无边框设计，聚焦时底部显示灰色底线 #D1D5DB
- **排版精调**：文件名 14px 加粗 #111827，辅助信息 12px #6B7280，内容垂直居中

## v0.5.0 (2026-06-23)

### 新增功能

- **文件大小人类可读展示**：表格自动显示 KB/MB/GB 格式，替代原始字节数
- **下载计数**：每次下载自动 +1，文件列表展示下载次数，管理员可查看全局统计
- **文件备注**：支持为文件添加备注描述，右键菜单"编辑备注"
- **上传时间 & 上传者**：文件上传时自动记录上传用户和时间戳，文件属性中可查看
- **磁盘空间展示**：仪表盘显示各 VFS 根目录所在磁盘的已用/可用/总空间
- **连接用户面板**：仪表盘实时展示活跃连接（IP/用户/请求路径）
- **请求日志**：自动记录所有 HTTP 请求到 access.log，管理员可在日志页面查看最近 200 条

### 改进

- 全面视觉重构：工业级 UI 风格，Lucide 图标替代 Emoji，专业配色方案
- 左右布局：侧边栏目录树 + 主内容区，管理后台独立侧栏
- 公开文件浏览：`/` 路由无需登录即可浏览/搜索/下载 Public 文件
- DashboardView 接入实时数据（磁盘空间 + 活跃连接）
- 数据库新增 `download_counts` 表和 `files_meta` 表

### API 新增

| 路由 | 方法 | 说明 |
|------|------|------|
| `/api/files/comment` | PUT | 更新文件备注 |
| `/api/admin/download-counts` | GET | 获取下载统计 |
| `/api/admin/connections` | GET | 获取活跃连接 |
| `/api/admin/disk-usage` | GET | 获取磁盘空间 |

### 修复

- `/api/admin/logs` 从占位实现改为读取 access.log 真实数据
- 下载计数在公开下载和认证下载时均正确递增
- 上传文件时自动记录 uploadedBy 和 createdAt

## v0.4.0 (2026-06-23)

### 新增功能

- 公开文件浏览：`/` 无需登录即可浏览/下载 Public 文件
- 左右布局：侧边栏 + 主内容区
- FileTree 目录树组件（展开/折叠、懒加载）
- AdminSidebar 管理后台独立侧栏
- 公开 API：`/api/public/files/roots|list|download`

### 修复

- Exe 从非项目目录启动时找不到前端（改用 //go:embed 嵌入二进制）
- VFS 路径全用相对路径导致新建文件夹失败（改用 os.Executable 绝对路径）
- 下载文件 401 未授权（改用 fetch + 手动注入 token）
- 前端页面被 style.css 中 1126px 固定宽度锁死

## v0.3.0 及更早

- Go 核心服务器 + Vue 3 SPA
- 文件浏览（列表/网格视图、面包屑导航、排序）
- 拖拽上传 + 进度条
- 文件操作（下载/删除/重命名/新建文件夹）
- 批量操作（多选/批量删除/批量 ZIP 下载）
- 用户认证（JWT + bcrypt）
- WebDAV 支持
- 用户管理（CRUD）
- 配置管理（读写 config.yaml）
