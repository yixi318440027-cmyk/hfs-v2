# hfs-v2

> 基于 [rejetto/hfs](https://github.com/rejetto/hfs) 的 Go 语言重写版本，高性能文件服务器。

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](./LICENSE)

当前版本：**v0.5.0**

## 特性

- **单二进制部署** — 前端嵌入，10MB 级体积，无运行时依赖
- **高性能并发** — Go 原生 Goroutine，内存占用低
- **现代化文件浏览** — 列表/网格视图、面包屑导航、排序、Lucide 图标
- **拖拽上传** — 支持拖拽上传，实时进度条
- **文件元数据** — 下载计数、文件备注、上传者、上传时间
- **批量操作** — 多选文件/文件夹，批量下载（ZIP）、删除
- **左右布局** — 侧边栏目录树 + 主内容区，专业工业级 UI
- **公开访问** — 无需登录即可浏览/下载 Public 文件
- **连接监控** — 实时活跃连接、磁盘空间展示
- **访问日志** — 自动记录请求日志（access.log）
- **跨平台** — 支持 Windows、Linux、macOS

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go + chi router |
| 前端 | Vue 3 + Vite + TypeScript |
| 图标 | Lucide Icons（线性风格） |
| 数据库 | SQLite（modernc.org/sqlite，纯 Go 实现，无 CGO） |
| 部署 | 单二进制 |

## 快速开始

```bash
# 直接运行（无需任何安装）
./hfs-v2.exe

# 访问
# 公开浏览：http://localhost:8080/
# 管理后台：http://localhost:8080/login （默认 admin/admin）
```

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/yixi318440027-cmyk/hfs-v2.git
cd hfs-v2

# 构建前端
cd web
npm install
npm run build

# 编译后端（前端自动嵌入）
cd ..
go build -o bin/hfs-v2 ./src/cmd/hfs
```

## 项目结构

```
hfs-v2/
├── bin/                        # 编译产物
│   ├── hfs-v2.exe              # 可执行文件
│   └── data/                   # 运行时数据（自动生成）
│       ├── hfs.db              # SQLite 数据库
│       ├── config.yaml         # 配置文件
│       ├── access.log          # 访问日志
│       └── Files/              # 默认文件存储目录
├── docs/                       # 项目文档
│   ├── API.md                  # API 接口文档
│   ├── CHANGELOG.md            # 版本日志
│   ├── hfs-roadmap-v1.md       # 产品路线图
│   └── dev-process.md          # 开发流程
├── src/
│   ├── cmd/hfs/                # 主入口
│   └── internal/
│       ├── auth/               # 认证模块（bcrypt + JWT）
│       ├── config/             # 配置解析（YAML）
│       ├── db/                 # 数据库层（SQLite 迁移）
│       ├── search/             # 搜索服务
│       ├── server/             # HTTP 服务器 + 路由
│       └── vfs/                # 虚拟文件系统
├── web/                        # 前端项目（Vue 3 SPA）
│   ├── src/
│   │   ├── views/              # 页面组件
│   │   │   ├── BrowseView.vue          # 文件管理
│   │   │   ├── PublicBrowseView.vue    # 公开浏览
│   │   │   ├── LoginView.vue           # 登录页
│   │   │   └── admin/                  # 管理后台
│   │   ├── components/         # 通用组件
│   │   │   ├── NavBar.vue
│   │   │   ├── FileTree.vue
│   │   │   └── AdminSidebar.vue
│   │   ├── stores/             # Pinia 状态管理
│   │   ├── api/                # API 客户端
│   │   └── utils/              # 工具函数
│   └── package.json
├── go.mod
└── go.sum
```

## 默认配置

启动后自动在 `data/config.yaml` 生成配置：

```yaml
port: ":8080"
data_dir: "<exe所在目录>/data"
log_level: "info"
jwt_secret: "change-me-in-production"
admin_user: "admin"
admin_pass: "admin"
vfs:
  roots:
    - path: "<exe所在目录>/data/Files"
      name: "Files"
      public: true
      read_only: false
```

## 文档

- [API 接口文档](./docs/API.md)
- [版本日志](./docs/CHANGELOG.md)
- [产品路线图](./docs/hfs-roadmap-v1.md)
- [开发流程](./docs/dev-process.md)

## License

[MIT](./LICENSE)
