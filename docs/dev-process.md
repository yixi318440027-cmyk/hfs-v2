# HFS v2 开发流程与规范

> 版本：v1.0 | 制定日期：2026-06-23 | 适用范围：hfs-v2 全生命周期

---

## 一、总体原则

1. **迭代驱动**：按路线图 Phase 拆分 Sprint（每 Sprint 1-2 周），每 Sprint 产出可交付的增量
2. **测试先行**：单元测试覆盖核心逻辑，集成测试覆盖 API 端点，未通过测试不发布
3. **本地验证**：每次迭代先在本地编译、运行、人工验证，确认无误后打 tag 推送 GitHub
4. **文档同步**：版本日志（CHANGELOG）、快速入门（QUICKSTART）、API 文档随版本更新

---

## 二、工作空间结构

```
hfs-v2/
├── src/                    ← Go 源码
│   ├── cmd/
│   │   └── hfs/            ← main.go 入口
│   ├── internal/           ← 内部包（不对外暴露）
│   │   ├── server/         ← HTTP/HTTPS 服务器
│   │   ├── vfs/            ← 虚拟文件系统
│   │   ├── auth/           ← 认证模块
│   │   ├── db/             ← SQLite 数据层
│   │   ├── api/            ← REST API 路由
│   │   ├── search/         ← 搜索模块（FTS5）
│   │   └── config/         ← 配置管理
│   └── pkg/                ← 可复用公共库
├── web/                    ← 前端源码（Vue 3 + Vite）
│   ├── src/
│   │   ├── views/          ← 页面组件
│   │   │   ├── files/      ← 文件浏览页
│   │   │   ├── admin/      ← 管理页（仪表盘、用户、配置、日志）
│   │   │   └── login/      ← 登录页
│   │   ├── components/     ← 通用组件
│   │   ├── stores/         ← Pinia 状态管理
│   │   ├── api/            ← 前端 API 调用层
│   │   └── i18n/           ← 多语言
│   └── ...
├── bin/                    ← 编译产物（gitignore）
├── test/                   ← 集成测试（Go）
│   └── e2e/                ← 端到端测试（Playwright/Cypress，Phase 2+）
├── docs/                   ← 产品与技术文档
│   ├── hfs-improvement-analysis.md
│   ├── file-manager-design.md
│   ├── hfs-roadmap-v1.md
│   ├── CHANGELOG.md
│   ├── QUICKSTART.md
│   └── API.md
├── .github/
│   └── workflows/
│       ├── ci.yml          ← CI：lint + test + build
│       └── release.yml     ← CD：打 tag 自动构建发布
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 三、迭代流程（Sprint 循环）

### 3.1 Sprint 规划

```
路线图 Phase → 拆分为 N 个 Sprint
每个 Sprint 周期：1-2 周

Sprint 开始前：
1. 从路线图 Phase 中选取本轮需求
2. 按 RICE 分数排序，取 Top 3-5 项
3. 为每项写 Mini-PRD（1 页纸：用户故事 + 验收标准）
4. 在 GitHub Issues 创建对应 Issue，打 label（feature/bug/chore）
```

### 3.2 开发流程

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ 写测试    │ → │ 写实现    │ → │ 本地验证  │ → │ 代码评审  │
│ (先失败)  │    │ (通过测试) │    │ (人工)    │    │ (自审)    │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                                                     │
                                                     ▼
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ 推送 GitHub│ ← │ 本地构建  │ ← │ 合并到    │ ← │ 运行全量  │
│ + 打 Tag  │    │ + 运行    │    │ main      │    │ 测试      │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
```

### 3.3 分支策略

```
main          ← 稳定分支，每个 Sprint 结束合并到这里，打 tag
  │
  ├─ feat/xxx ← 功能分支，从 main 拉出，完成后合并回 main
  ├─ fix/xxx  ← 修复分支
  └─ chore/xxx← 杂项分支（文档、构建脚本等）

Tag 命名：v0.1.0 格式（主版本.次版本.修订号）
  - 主版本：重大架构变更或不兼容 API
  - 次版本：新功能（每个 Sprint）
  - 修订号：Bug 修复
```

---

## 四、测试规范

### 4.1 测试金字塔

```
         ╱  E2E 测试  ╲          ← Phase 2+：Playwright，核心流程
        ╱───────────────╲
       ╱  集成测试(API)  ╲         ← Go httptest，所有 API 端点
      ╱───────────────────╲
     ╱   单元测试(Go)      ╲        ← 所有 internal/pkg 包
    ╱───────────────────────╲
```

### 4.2 单元测试要求

| 包 | 覆盖率要求 | 重点测试 |
|----|----------|---------|
| `internal/vfs` | ≥ 80% | 文件树构建、权限匹配、路径解析 |
| `internal/auth` | ≥ 90% | 登录、鉴权、Token 生成/校验 |
| `internal/db` | ≥ 80% | CRUD 操作、事务、迁移 |
| `internal/search` | ≥ 80% | FTS5 索引、查询、中文分词 |
| `internal/api` | ≥ 70% | 参数校验、错误处理 |
| `internal/config` | ≥ 90% | YAML 解析、默认值、环境变量覆盖 |

### 4.3 集成测试要求

每个 API 端点至少覆盖：
- **正常场景**：200 OK，返回预期数据
- **参数错误**：400 Bad Request
- **权限不足**：401/403
- **资源不存在**：404 Not Found
- **并发安全**：同一资源并发操作

### 4.4 测试命令

```bash
# 运行所有单元测试
go test ./internal/... ./pkg/...

# 运行集成测试
go test ./test/...

# 查看覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 竞态检测
go test -race ./...
```

---

## 五、发布流程

### 5.1 本地发布前检查清单

```
[ ] 全量单元测试通过（go test ./...）
[ ] 集成测试通过
[ ] 代码无 race condition（go test -race ./...）
[ ] 编译成功（go build -o bin/hfs.exe ./cmd/hfs）
[ ] 启动无报错（bin/hfs.exe）
[ ] 前端构建成功（cd web && npm run build）
[ ] 手动验证核心流程：
    [ ] 浏览器访问文件列表页
    [ ] 上传一个文件
    [ ] 下载一个文件
    [ ] 搜索一个文件
    [ ] 管理员登录 → 管理面板
[ ] CHANGELOG.md 已更新
[ ] QUICKSTART.md 已更新（如有新功能）
[ ] API.md 已更新（如有新接口）
```

### 5.2 推送 GitHub

```bash
# 1. 合并到 main
git checkout main
git merge feat/xxx

# 2. 更新版本号（手动修改代码中的 version 常量）

# 3. 提交
git add .
git commit -m "release: v0.1.0 - 文件浏览 MVP"

# 4. 打 tag
git tag -a v0.1.0 -m "v0.1.0: 文件浏览 MVP，含列表视图、上传、搜索、WebDAV"

# 5. 推送
git push origin main --tags
```

### 5.3 CI/CD（.github/workflows/ci.yml）

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go vet ./...
  build:
    needs: test
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [linux, windows, darwin]
        goarch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4
      - run: GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build -o bin/hfs-${{ matrix.goos }}-${{ matrix.goarch }} ./cmd/hfs
      - uses: actions/upload-artifact@v4
        with:
          name: hfs-${{ matrix.goos }}-${{ matrix.goarch }}
          path: bin/
```

---

## 六、文档更新规范

### 6.1 CHANGELOG.md 格式

```markdown
# Changelog

## [v0.1.0] - 2026-07-15

### Added
- 文件列表视图（面包屑导航 + 排序）
- 文件上传（拖拽 + 点击 + 进度条）
- 文件名模糊搜索（前端实时过滤）
- 基础文件操作（下载、删除、重命名、新建文件夹）
- WebDAV 基础支持
- SQLite 持久化（账户、配置）

### Changed
- 后端由 Node.js 重写为 Go
- 前端由 React(CRA) 重写为 Vue 3 + Vite

### Fixed
- (无，首次发布)

### Security
- 账户密码使用 bcrypt 加密存储
```

### 6.2 API.md 格式

每个 API 端点记录：

```markdown
## GET /api/files

列出指定目录下的文件和子目录。

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | 否 | 目录路径，默认根目录 `/` |
| `sort` | string | 否 | 排序字段：`name`/`size`/`time`，默认 `name` |
| `order` | string | 否 | `asc`/`desc`，默认 `asc` |

**响应示例**

```json
{
  "files": [
    {
      "name": "report.pdf",
      "size": 2416640,
      "modTime": "2026-06-20T10:30:00Z",
      "isDir": false,
      "mime": "application/pdf"
    }
  ]
}
```

**错误码**

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 401 | 未登录 |
| 403 | 无权访问该目录 |
| 404 | 目录不存在 |
```

---

## 七、WebAdmin 处理方案

### 7.1 结论：完全重写，与文件浏览前端统一

HFS 原版 Admin（`/~/admin`）独立于文件浏览页面，技术栈老旧（CRA + 无 JSX），配置读写方式不兼容（YAML 直接编辑 vs SQLite 管理）。**不改造，直接重写。**

### 7.2 新 Admin 设计

```
统一前端入口
├── 未登录 → 登录页
├── 普通用户登录 → 文件浏览页（无管理入口）
└── 管理员登录 → 文件浏览页 + 顶部导航多出「⚙️ 管理」
                   │
                   ├── 📊 仪表盘
                   │     ├── 存储空间使用
                   │     ├── 在线用户数
                   │     ├── 今日流量
                   │     └── 热门文件 Top 10
                   │
                   ├── 👥 用户管理
                   │     ├── 用户列表（搜索/筛选/分页）
                   │     ├── 新建/编辑/禁用用户
                   │     ├── 用户组管理
                   │     └── 批量导入（Phase 2+）
                   │
                   ├── ⚙️ 系统配置
                   │     ├── 基础设置（端口、标题、主题）
                   │     ├── HTTPS 证书管理
                   │     ├── 速度限制
                   │     ├── 日志配置
                   │     └── VFS 目录管理（可视化）
                   │
                   └── 📋 操作日志
                         ├── 登录日志
                         ├── 文件操作日志
                         └── 管理员操作日志
```

### 7.3 Phase 1 Admin 最小实现

Phase 1（MVP）只做最基本的：

- 登录/登出
- 用户管理（增删改查，仅管理员可见）
- 系统配置（YAML 导入导出 + 基本参数 Web 编辑）
- 日志查看（只读，显示最近 500 条）

不做：
- 仪表盘（Phase 3）
- 用户组（Phase 4 RBAC 时做）
- VFS 可视化编辑（Phase 1 用 YAML 手工编辑，Phase 3 做可视化）
- 操作审计（Phase 4）

### 7.4 与文件浏览前端的关系

```
web/src/
├── router/
│   └── index.ts          ← 统一路由
│       ├── /             → 文件浏览（所有登录用户）
│       ├── /login        → 登录页
│       └── /admin/*      → 管理页（需 admin 角色）
│           ├── /admin/dashboard
│           ├── /admin/users
│           ├── /admin/config
│           └── /admin/logs
```

同一个 Vue 3 项目，同一个构建产物，根据 `user.role` 控制导航栏是否显示管理入口。

---

## 八、Sprint 0：项目初始化（当前阶段）

在进入 Phase 1 第一个功能 Sprint 之前，先完成基础设施：

```
Sprint 0 任务清单：
[ ] 初始化 Go module（go mod init github.com/yixi318440027-cmyk/hfs-v2）
[ ] 创建目录结构（src/cmd, src/internal/*, web/, test/）
[ ] 配置 .gitignore
[ ] 编写 Makefile（build/test/run/clean 目标）
[ ] 搭建 Vue 3 + Vite 项目脚手架
[ ] 配置 ESLint + Prettier（Go: golangci-lint, Vue: eslint-plugin-vue）
[ ] 创建 CI workflow（.github/workflows/ci.yml）
[ ] 编写 README.md（项目介绍 + 快速开始 + 路线图链接）
[ ] 初始化 SQLite 数据库迁移框架
[ ] 搭建基础 HTTP Server（net/http + chi），返回 "Hello, hfs-v2"
[ ] 前端基础 App.vue + 路由框架（Vue Router + Pinia）
```

---

## 九、版本规划速览

| 版本 | 预计日期 | 核心交付 | 标签 |
|------|---------|---------|------|
| **v0.1.0** | 2026-07-15 | Go 核心 + 文件浏览 MVP + 上传 + 搜索 + WebDAV + Admin 最小版 | Sprint 1-3 |
| **v0.2.0** | 2026-08-15 | 右键菜单 + 快捷键 + 目录树 + 批量操作 + 搜索筛选 | Sprint 4-5 |
| **v0.3.0** | 2026-09-15 | 网格视图 + 缩略图 + HTTP/2 + FTP + 分享链接 + FTS5 搜索 | Sprint 6-8 |
| **v0.4.0** | 2026-10-30 | RBAC + OAuth2 + 仪表盘 + 回收站 + 文件预览 | Sprint 9-11 |
| **v1.0.0** | 2026-12-30 | 移动端 PWA + S3 API + 插件系统 + 多节点管理 | Sprint 12-15 |
