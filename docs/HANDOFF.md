# hfs-v2 项目交接文档

> 写给「换一台电脑、由新 Agent 接手继续开发」用的现状快照。
> 生成时间：2026-10-26 | 代码版本：v0.6.1 | 分支：main

---

## 一、一句话现状

**hfs-v2 是一个用 Go 重写的 HFS（HTTP File Server）**，前端 Vue 3 + TypeScript，数据库 SQLite（纯 Go，无 CGO），前端产物用 `//go:embed` 打进单个二进制。

当前 **v0.6.1 已完成并可用**：文件浏览（列表/网格）、上传下载、批量操作、JWT 认证、用户管理、WebDAV、文件元数据、磁盘/连接监控、访问日志、公开访问、**6 种权限 × 路径 × 用户**的权限系统。

**代码能编译、能跑通**，但**零单元测试**，且有若干技术债（见第六节）。

---

## 二、新机器上的第一步：跑起来

### 2.1 环境要求

| 组件 | 版本 | 说明 |
|------|------|------|
| Go | **1.25.0** | `go.mod` 要求 `go 1.25.0` |
| Node.js | **22.x** | 前端构建用，`vite 8` 需要 Node ≥ 20.19 |
| Git | 任意 | 远程目前是 **SSH** 地址，新机器需配 SSH key，或改成 HTTPS |

> **坑 1**：原机器的 `/etc/profile.d/go.sh`（把 go 加进 PATH）在新机器上不存在，需要重新配置或每次手动：
> ```bash
> export GOROOT="/c/Users/<用户名>/.workbuddy/binaries/go/versions/1.25.0/go"
> export PATH="$GOROOT/bin:$PATH"
> ```
> **坑 2**：远程是 `git@github.com:yixi318440027-cmyk/hfs-v2.git`（SSH）。没有 SSH key 就改用 HTTPS + PAT：
> ```bash
> git remote set-url origin https://github.com/yixi318440027-cmyk/hfs-v2.git
> ```

### 2.2 拉代码 + 首次构建

```bash
git clone git@github.com:yixi318440027-cmyk/hfs-v2.git
cd hfs-v2

# 1) 构建前端
cd web && npm install && npm run build && cd ..

# 2) 把前端产物同步到 Go 的 embed 目录（关键步骤，见坑 3）
rm -rf src/internal/server/webdist && cp -r web/dist src/internal/server/webdist

# 3) 编译后端
go build -o bin/hfs-v2.exe ./src/cmd/hfs

# 4) 运行
./bin/hfs-v2.exe
```

访问：
- 公开浏览 `http://localhost:8080/`
- 管理后台 `http://localhost:8080/login`（默认 `admin` / `admin`）

> **坑 3（最容易忘）**：`src/internal/server/webdist/` 是**提交进 git 的**（因为要 embed 进二进制），`web/dist/` 是 gitignore 的。
> 顺序必须是：**改前端 → npm run build → 删掉旧 webdist → 复制 web/dist 过去 → 再 go build**。
> 如果只 build 前端不复制，跑起来还是旧界面；如果只删不加，编译会报 embed 找不到文件。

> **坑 4（两个 data 目录）**：`DataDir` 是**相对可执行文件所在目录**计算的（`os.Executable()`）。
> - 从项目根 `./bin/hfs-v2.exe` 启动 → 数据落在 `bin/data/`
> - 把 exe 拷到别处启动 → 数据落在 exe 同级的 `data/`
>
> 目前仓库里同时存在 `data/` 和 `bin/data/`（历史遗留），两者都有 `hfs.db` + `Files/`。
> 排查"为什么数据不一样"时先确认服务实际读写哪个目录。清理时只删不要的那个即可。

---

## 三、代码地图

### 3.1 后端 `src/`

| 路径 | 职责 |
|------|------|
| `cmd/hfs/main.go` | 入口：`config.Default()` → 建 Server → 监听 → 优雅关闭 |
| `internal/config/config.go` | 配置结构 + `Default()` + `LoadConfig()`（**注意：LoadConfig 目前无人调用**，见技术债） |
| `internal/db/db.go` | SQLite 初始化与建表迁移（`users` / `download_counts` / `files_meta` / `permissions`） |
| `internal/auth/auth.go` | bcrypt 密码哈希 + JWT 签发/校验 |
| `internal/permission/permission.go` | **权限引擎**：`Check()` / `Can()`，路径继承 + admin 豁免 + 特殊用户名 |
| `internal/vfs/vfs.go` | 虚拟文件系统：路径穿越防护、列举、上传、删除、重命名、ZIP、目录树收集 |
| `internal/search/search.go` | 搜索服务（目前前端以实时过滤为主） |
| `internal/server/server.go` | Server 装配（cfg / vfs / db / perm / auth 注入） |
| `internal/server/routes.go` | **所有路由 + handler**（36 处注册），权限校验注入点在这里 |
| `internal/server/middleware.go` | Bearer token 校验、角色鉴权、请求日志、连接统计 |
| `internal/server/webdav.go` | WebDAV（OPTIONS/PROPFIND/GET/PUT） |
| `internal/server/disk_usage*.go` | 跨平台磁盘空间 |
| `internal/server/webdist/` | **前端 embed 产物（提交进 git，勿手改）** |

### 3.2 前端 `web/src/`

| 路径 | 职责 |
|------|------|
| `App.vue` | 整体布局：顶部 NavBar + 左侧栏（文件树 / 管理菜单）+ 主区 |
| `router/index.ts` | 路由 + 全局守卫（`requiresAuth` / `requiresAdmin`） |
| `stores/auth.ts` | Pinia 认证 store，**token + user 持久化到 localStorage** |
| `api/index.ts` | axios 实例，请求拦截注入 Bearer token，401 时按登录态决定是否跳登录 |
| `views/BrowseView.vue` | 文件管理主页面（列表/网格、上传、批量、右键菜单、快捷键） |
| `views/PublicBrowseView.vue` | 未登录时的公开浏览页 |
| `views/LoginView.vue` | 登录页 |
| `views/admin/DashboardView.vue` | 仪表盘（磁盘、活跃连接） |
| `views/admin/UsersView.vue` | 用户管理 |
| `views/admin/ConfigView.vue` | 配置管理 |
| `views/admin/LogsView.vue` | 访问日志 |
| `views/admin/PermissionsView.vue` | **权限管理**（本次改动最多的文件） |
| `components/FileTree.vue` | 目录树侧栏 |
| `components/NavBar.vue` / `AdminSidebar.vue` | 顶部导航 / 管理侧栏 |
| `utils/lucide.ts` | `refreshLucide()` — DOM 更新后重渲染 Lucide 图标 |

### 3.3 API 一览（前缀 `/api`）

```
# 认证
POST   /auth/login          POST /auth/logout

# 文件（需登录）
GET    /files                   列出目录
GET    /files/download          下载单文件
GET    /files/download-zip      批量打包下载
POST   /files/upload            多文件上传
POST   /files/mkdir             新建目录
POST   /files/rename            重命名
DELETE /files                   删除
POST   /files/batch-delete      批量删除
PUT    /files/comment           改文件备注
GET    /files/tree              全量目录树（权限路径选择器用）

# 公开（免登录）
GET    /public/files/roots | /list | /download

# 管理（需 admin）
GET    /admin/users             POST /admin/users     DELETE /admin/users
GET    /admin/config            PUT  /admin/config
GET    /admin/logs
GET    /admin/connections
GET    /admin/disk-usage
GET    /admin/download-counts
GET    /admin/permissions       POST/PUT /admin/permissions    DELETE /admin/permissions
POST   /admin/permissions/batch

# WebDAV
/dav/*
```

---

## 四、已实现功能（按版本）

| 版本 | 内容 |
|------|------|
| v0.1.0 | Go 核心 + VFS + 认证 + 文件浏览/上传/删除/重命名 + WebDAV + 批量操作 + 管理后台 4 页 |
| v0.4.0 | 公开访问（`/` 免登录）+ 左右布局 + 目录树侧栏 + 前端 embed |
| v0.5.0 | 下载计数、文件备注、上传者/时间、磁盘空间、连接监控、访问日志、全面视觉重构 |
| v0.5.1 | Grid View 工业级视觉（Lucide 图标、去 Emoji、统一 token） |
| v0.6.0 | **文件权限系统**：6 权限 × per-path × per-user、路径继承、admin 豁免、权限管理页、批量授权、路径引用选择器 |
| v0.6.1 | 修两个 bug：编辑权限新建记录、刷新掉登录态 |

---

## 五、关键设计决策（改代码前必读）

### 5.1 权限模型：用「特殊用户名」表达可见性

权限表只有一张：`permissions(username, vfs_path, can_see, can_read, can_list, can_upload, can_delete, can_archive)`，唯一键 `(username, vfs_path)`。

三种「可见性级别」**不是独立字段**，而是靠特殊 username 编码：

| 可见性级别 | 落库 username | 含义 |
|-----------|--------------|------|
| 公开可见 | `*` | 所有人（含匿名） |
| 仅登录可见 | `@` | 任何已登录用户 |
| 指定用户 | 真实用户名 | 仅该用户（+ admin） |

`Check()` 的查询优先级：**`*` > `@` > 具体用户 > 父路径继承 > 默认值**，逐级向上找父目录。

- 匿名默认：全 deny
- 登录默认：`see/read/list = true`，其余 false
- admin：直接返回全 true，绕过所有检查

**因此：前端「可见性级别」只是快捷方式，真正的判定在后端。** 编辑已有记录时 username 被锁定（改它就会变成另一条记录）。

### 5.2 前端产物 embed
`//go:embed webdist` → 单二进制自带前端。改了前端**必须**重新同步 webdist 再编译。

### 5.3 认证
JWT（24h），token 存 localStorage（v0.6.1 起）。401 拦截器只在「原本已登录」时才跳登录页，避免公开接口 401 误踢。

### 5.4 UI 规范（务必遵守）
- 背景 `#F3F4F6`，边框 `#E5E7EB`，圆角 4px（弹窗 8px），主色 `#3B82F6`
- 图标**只用 Lucide**（`<i data-lucide="xxx">` + `refreshLucide()`），**禁止 Emoji**
- 字体 14px 基准，系统无衬线栈

---

## 六、已知问题 / 技术债（按优先级）

| # | 问题 | 影响 | 建议 |
|---|------|------|------|
| 1 | **`config.yaml` 启动时不加载**：`main.go` 只调 `config.Default()`，`LoadConfig()` 从未被调用 | 在管理页改的 port / data_dir / vfs roots **重启后丢失**，等于配置管理功能半残 | 在 `main.go` 里加：若 `<DataDir>/config.yaml` 存在则 `LoadConfig`，并回落到 Default |
| 2 | **零测试**：整个仓库没有任何 `*_test.go` | 权限这类关键逻辑无回归保护 | 优先给 `permission` 包补表驱动测试（Check 的 4 级优先级 + 路径继承） |
| 3 | 两个 `data/` 目录并存 | 排查数据问题时容易搞错对象 | 删掉不用的那个，或统一改为固定路径 |
| 4 | 本地无 git tag（Release 只在 GitHub 上打） | 本地回滚不便 | `git push --tags` 同步，或改用 tag 驱动发布 |
| 5 | VFS 只支持**单 root**（默认 `data/Files`） | 原生 HFS 支持多 Real/Virtual Folder | 配置结构 `VFS.Roots` 已是数组，UI/管理接口还没暴露增删 |
| 6 | 权限判定中 root 层依赖 `VFSRoot.Public` 兜底（`routes.go` 公开 roots 处） | 与权限表存在两套来源，语义重叠 | 收敛为只认权限表 |
| 7 | 根目录 `hfs.exe`、`bin/`、`web/dist` 等产物混杂 | 仓库/目录略乱 | 已在 `.gitignore` 忽略，注意别误提交 |
| 8 | 无 CI 实际校验（有 `.github/workflows/ci.yml`，需确认是否跑通） | 质量门禁不确定 | 新机器上跑一次 Actions 确认 |

---

## 七、下一步建议

参考 `docs/hfs-roadmap-v1.md` 的 RICE 排序，**建议的优先级**：

**P0 — 先把地基补上（1~2 天量级）**
1. 修 `config.yaml` 不加载（技术债 #1）——否则配置页形同虚设
2. 给 `permission` 包补单元测试（技术债 #2）
3. 打 `v0.6.1` tag 并同步

**P1 — 权限系统收尾 + 体验**
4. 多 root 管理（VFS roots 的增删改 UI + API）
5. 分享链接（时效 + 密码）— roadmap RICE 100
6. 搜索筛选（类型/时间/大小）— roadmap RICE 189

**P2 — 能力扩展**
7. SQLite FTS5 全文搜索
8. 文件在线预览（图片/PDF/文本）
9. 回收站
10. 多协议：FTP / S3 / OPC 类对接（参照兄弟项目 plc-web 的驱动分层思路）

**不建议现在做**：OAuth2/OIDC、移动端 PWA——RICE 最低，且用户群尚未验证。

---

## 八、常用命令速查

```bash
# 构建（完整顺序，别跳步）
cd web && npm run build && cd ..
rm -rf src/internal/server/webdist && cp -r web/dist src/internal/server/webdist
go build -o bin/hfs-v2.exe ./src/cmd/hfs

# 开发模式（前后端分离热更新）
go run ./src/cmd/hfs          # 后端 :8080
cd web && npm run dev         # 前端 dev server（Vite 代理到 8080）

# 静态检查
go vet ./src/...

# 清端口（Windows）
netstat -ano | findstr :8080
taskkill /F /PID <pid>
# 或 PowerShell：
Get-NetTCPConnection -LocalPort 8080 | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force }

# 数据库直查
# 用 sqlite3 打开 data/hfs.db 或 bin/data/hfs.db
```

---

## 九、接手 Checklist

- [ ] `git pull` 拿到最新（确认 HEAD 包含 v0.6.1 的两个 bug 修复）
- [ ] 配好 Go 1.25 + Node 22 的 PATH
- [ ] 按 2.2 顺序完整构建一次，确认 `bin/hfs-v2.exe` 生成
- [ ] 启动后访问 `/` 和 `/login`，admin/admin 能登录
- [ ] **刷新页面**确认不掉登录态（v0.6.1 修的点）
- [ ] 进「权限管理」，编辑一条已有记录，确认**行数不变**（v0.6.1 修的点）
- [ ] 读完第六节技术债，从 P0 #1 开始
- [ ] 每轮改动后同步更新 `docs/CHANGELOG.md`

---

## 十、相关文档

- `docs/CHANGELOG.md` — 版本变更（含 v0.6.1 修复说明）
- `docs/PRD-v0.6.0-file-permissions.md` — 权限系统 PRD
- `docs/API.md` — API 细节
- `docs/dev-process.md` — 开发流程与规范
- `docs/hfs-roadmap-v1.md` — 产品路线图（RICE 打分 + Now/Next/Later）
- `docs/file-manager-design.md` — 文件管理页面设计论证
- `docs/hfs-improvement-analysis.md` — 二开改进分析（对标社区 Issue）
