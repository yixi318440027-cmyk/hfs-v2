# hfs-v2

> 基于 [rejetto/hfs](https://github.com/rejetto/hfs) 的 Go 语言重写版本，高性能文件服务器。

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](./LICENSE)
[![CI](https://github.com/yixi318440027-cmyk/hfs-v2/workflows/CI/badge.svg)](https://github.com/yixi318440027-cmyk/hfs-v2/actions)

## 特性

- **单二进制部署** — 10MB 级体积，无运行时依赖，开箱即用
- **高性能并发** — Go 原生 Goroutine，内存占用 <20MB，高并发不退化
- **现代化文件浏览** — 列表/网格视图、面包屑导航、排序、缩略图预览
- **拖拽上传** — 支持拖拽上传文件/文件夹，实时进度条，断点续传
- **智能搜索** — 文件名模糊搜索 + SQLite FTS5 全文搜索，支持类型/时间/大小筛选
- **批量操作** — 多选文件/文件夹，批量下载、删除、移动
- **分享链接** — 支持时效、密码、下载次数限制的安全分享
- **企业级特性** — RBAC 权限模型、OAuth2/OIDC 单点登录、审计日志
- **跨平台** — 支持 Windows、Linux、macOS 及 Docker 部署
- **协议扩展** — HTTP/2、WebDAV、FTP/FTPS 多协议支持

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go + chi router |
| 前端 | Vue 3 + Vite + TypeScript |
| 数据库 | SQLite（modernc.org/sqlite，纯 Go 实现，无 CGO） |
| 搜索 | SQLite FTS5 全文搜索引擎 |
| 部署 | 单二进制 + Docker |

## 快速开始

```bash
# 克隆仓库
git clone https://github.com/yixi318440027-cmyk/hfs-v2.git
cd hfs-v2

# 编译后端
go build -o bin/hfs-v2 ./src/cmd/hfs
./bin/hfs-v2

# 前端开发
cd web
npm install
npm run dev
```

## 项目结构

```
hfs-v2/
├── .github/
│   └── workflows/
│       └── ci.yml              # CI 流水线
├── bin/                        # 编译产物
├── docs/                       # 项目文档
│   ├── hfs-roadmap-v1.md       # 产品路线图
│   ├── dev-process.md          # 开发流程
│   ├── file-manager-design.md  # 文件管理设计
│   └── hfs-improvement-analysis.md
├── src/
│   ├── cmd/
│   │   └── hfs/                # 主入口
│   ├── internal/
│   │   ├── api/                # API 路由与处理器
│   │   ├── auth/               # 认证模块
│   │   ├── config/             # 配置解析
│   │   ├── db/                 # 数据库层
│   │   ├── search/             # 搜索服务
│   │   ├── server/             # HTTP 服务器
│   │   └── vfs/                # 虚拟文件系统
│   └── pkg/                    # 可复用公共库
├── web/                        # 前端项目
│   ├── src/                    # Vue 3 源码
│   ├── dist/                   # 前端构建产物
│   └── package.json
├── go.mod
├── go.sum
└── Makefile
```

## 文档

- [产品路线图](./docs/hfs-roadmap-v1.md)
- [开发流程](./docs/dev-process.md)

## License

[MIT](./LICENSE)
