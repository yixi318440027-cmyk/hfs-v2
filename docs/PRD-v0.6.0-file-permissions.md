# PRD: v0.6.0 文件权限系统

## 1. 问题陈述

当前 hfs-v2 的授权模型只有两层：**未登录用户（只能看 Public 根目录）** 和 **已登录用户（能看到所有根目录、能做所有操作）**。所有认证用户拥有完全相同的权限——能上传、能删除、能重命名所有文件。

这在以下场景中产生问题：
- **团队协作**：A 用户上传的文件被 B 用户误删
- **只读共享**：想让某个用户只能下载不能上传/删除，当前做不到
- **受限可见**：想隐藏某个子目录对特定用户不可见，当前做不到
- **审计合规**：没有操作审计链路，谁删了什么无从追溯

**对标原生 HFS**：原生 HFS 支持 6 种权限（can_see / can_read / can_list / can_upload / can_delete / can_archive），可对每个用户或用户组按路径粒度授权。

## 2. 目标

| # | 目标 | 衡量标准 |
|---|------|---------|
| G1 | 管理员可为任意用户配置任意 VFS 路径的 6 种细粒度权限 | 权限配置页面可完成增删改查 |
| G2 | 管理界面提供"可见性级别"快捷入口（公开/仅登录/指定用户），降低逐条配置成本 | 切换可见性级别后，底层权限自动写入 |
| G3 | 已登录用户的 VFS 操作受权限约束，无权限操作被拒绝 | 非授权用户尝试删除/上传/查看文件返回 403 |
| G4 | Admin 角色不受权限系统约束（超级用户豁免） | Admin 始终可执行任何 VFS 操作 |
| G5 | 子目录自动继承父目录权限（简化配置） | 对 `/Files` 授权 can_read 后，`/Files/sub/` 自动可读 |
| G6 | 不影响现有公开浏览功能 | Public 根目录的公开访问行为不变 |

## 3. 非目标（Non-Goals）

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | 用户组（Group） | v0.6.0 只做 per-user 授权；用户组留到后续版本 |
| N2 | 基于 IP 的访问控制 | 属于网络安全层面，不在此版本范围 |
| N3 | 操作审计日志（谁在何时做了什么） | 已有 access.log，精细化审计留到后续 |
| N4 | 前端文件树按权限裁剪（看不到的文件夹不展示） | v0.6.0 先做操作拦截，UI 裁剪是后续优化 |

## 4. 用户故事

### P0 — 必须交付

**US1 — 管理员配置路径可见性与权限**
> 作为管理员，我希望在管理后台为每个 VFS 路径设置"公开可见 / 仅登录可见 / 指定用户可见"，并在此基础上微调 6 种操作权限，以便快速完成授权。

**US1b — 可见性级别快捷配置**
> 作为管理员，切换路径的可见性级别时，底层 6 种权限自动批量写入，无需逐条手动勾选。

**US2 — 权限检查拦截未授权操作**
> 作为系统，当用户尝试查看/下载/上传/删除/重命名文件时，我需要检查该用户对该路径是否拥有对应权限，如果没有则拒绝操作并返回 403。

**US3 — Admin 豁免**
> 作为管理员，我的所有 VFS 操作应跳过权限检查，以便我能不受限制地管理所有文件。

**US4 — 子目录权限继承**
> 作为管理员，当我为用户授予 `/Files/project-a` 的 can_read 权限后，该用户应自动拥有 `/Files/project-a/subdir/file.txt` 的 can_read 权限，无需逐层配置。

### P1 — 应尽快交付

**US5 — 权限拒绝时有明确提示**
> 作为普通用户，当我的操作因权限不足被拒绝时，我希望看到明确的提示信息（如"权限不足：您没有删除此文件的权限"），以便我知道原因并联系管理员。

**US6 — 文件列表隐藏无 can_list 权限的目录**
> 作为普通用户，当我浏览文件时，没有 can_list 权限的目录不应出现在文件列表中，以便我不被无法访问的目录困扰。

**US7 — 批量授权**
> 作为管理员，我希望为多个用户批量配置相同路径的权限，以便快速完成团队权限初始化。

### P2 — 后续版本

**US8 — 用户组权限**
> 作为管理员，我希望创建用户组并为组配置权限，用户加入组后自动获得组权限。

**US9 — 权限变更通知**
> 当管理员修改我的权限后，我应收到通知（或下次操作时看到权限已变更）。

## 5. 权限模型

### 5.1 六种权限定义

| 权限 | 常量名 | 控制的操作 | 默认值（未配置时） |
|------|--------|-----------|-------------------|
| `can_see` | `PermSee` | 文件是否存在于系统中（最基础权限） | `true`（所有登录用户可见） |
| `can_read` | `PermRead` | 下载文件、读取内容 | `true` |
| `can_list` | `PermList` | 列出目录内容 | `true` |
| `can_upload` | `PermUpload` | 上传文件、创建目录 | `false` |
| `can_delete` | `PermDelete` | 删除文件/目录 | `false` |
| `can_archive` | `PermArchive` | 批量 ZIP 下载 | `false` |

### 5.2 可见性级别（管理界面快捷入口）

管理界面提供三个可见性级别，切换时自动设置底层 6 种权限：

| 可见性 | 对谁生效 | 自动设置的权限 |
|--------|---------|---------------|
| **公开可见** | 所有人（含未登录） | `can_see/can_read/can_list = true` |
| **仅登录可见** | 所有已登录用户 | `can_see/can_read/can_list = true`（需登录） |
| **指定用户可见** | 手动选择用户 | 仅选中用户的 `can_see/can_read/can_list = true` |

写入权限（can_upload/can_delete/can_archive）不随可见性级别自动设置，需单独勾选。

### 5.3 权限生效优先级

```
1. 用户是 admin → 所有权限 = true（最高优先级）
2. 路径精确匹配的权限记录 → 使用该记录
3. 父路径匹配的权限记录 → 继承最近父路径的权限
4. 无任何匹配记录 → 使用默认值（见上表）
```

### 5.3 数据库设计

```sql
CREATE TABLE IF NOT EXISTS permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    vfs_path TEXT NOT NULL,
    can_see INTEGER NOT NULL DEFAULT 1,
    can_read INTEGER NOT NULL DEFAULT 1,
    can_list INTEGER NOT NULL DEFAULT 1,
    can_upload INTEGER NOT NULL DEFAULT 0,
    can_delete INTEGER NOT NULL DEFAULT 0,
    can_archive INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(username, vfs_path)
);
```

### 5.4 继承规则

- 查找用户对路径 `/Files/a/b/c.txt` 的权限：
  1. 先查 `vfs_path = '/Files/a/b/c.txt'` 的精确记录
  2. 若没有，查 `vfs_path = '/Files/a/b'`
  3. 若没有，查 `vfs_path = '/Files/a'`
  4. 若没有，查 `vfs_path = '/Files'`
  5. 都没有 → 使用默认值

## 6. 需求分级

### P0 — Must Have

| # | 需求 | 对应 US |
|---|------|---------|
| R1 | `permissions` 表设计与迁移 | — |
| R2 | 权限检查引擎（含继承逻辑、admin 豁免） | US2, US3, US4 |
| R3 | VFS 操作注入权限检查（list/download/upload/delete/rename/mkdir/download-zip） | US2 |
| R4 | 管理员 API：GET/POST/PUT/DELETE `/api/admin/permissions` | US1 |
| R5 | 管理后台权限配置页面（路径列表 + 可见性下拉 + 用户选择 + 6 种权限复选框） | US1, US1b |

### P1 — Should Have

| # | 需求 | 对应 US |
|---|------|---------|
| R6 | 403 响应携带具体权限缺失信息 | US5 |
| R7 | ListDir 时过滤无 can_list 权限的目录 | US6 |
| R8 | 批量授权 API（一次请求配置多个用户-路径） | US7 |

### P2 — Future

| # | 需求 |
|---|------|
| R9 | 用户组权限 |
| R10 | 前端按权限禁用操作按钮（非仅拦截） |
| R11 | 权限审计日志 |

## 7. API 设计

### 7.1 获取某用户的权限列表

```
GET /api/admin/permissions?username=alice
```

Response:
```json
{
  "ok": true,
  "data": [
    {
      "id": 1,
      "username": "alice",
      "vfsPath": "/Files",
      "canSee": true,
      "canRead": true,
      "canList": true,
      "canUpload": false,
      "canDelete": false,
      "canArchive": false
    }
  ]
}
```

### 7.2 为用户创建/更新权限

```
POST /api/admin/permissions
```

Request:
```json
{
  "username": "alice",
  "vfsPath": "/Files/project-a",
  "canRead": true,
  "canList": true,
  "canUpload": true,
  "canDelete": false,
  "canArchive": false
}
```

### 7.3 删除权限记录

```
DELETE /api/admin/permissions?id=1
```

### 7.4 批量授权

```
POST /api/admin/permissions/batch
```

Request:
```json
{
  "usernames": ["alice", "bob", "charlie"],
  "vfsPath": "/Files/shared",
  "canRead": true,
  "canList": true,
  "canUpload": false,
  "canDelete": false,
  "canArchive": false
}
```

## 8. 成功指标

| 指标 | 目标 | 测量方法 |
|------|------|---------|
| 权限配置成功率 | 100% 的配置操作成功生效 | 管理员配置后，用对应用户 token 验证 |
| 越权拦截率 | 100% 的未授权操作被 403 拦截 | 自动化测试覆盖 6 种权限 × 4 种操作 |
| 权限查询延迟 | <5ms（SQLite 本地查询） | Benchmark 测试 |
| Admin 豁免零误拦 | Admin 操作永不被权限系统拦截 | 自动化测试 |
| 公开访问不受影响 | Public 根目录的未登录访问行为完全不变 | 回归测试 |

## 9. 开放问题

| # | 问题 | 负责人 | 阻塞？ |
|---|------|--------|--------|
| Q1 | 权限修改后是否需要让已登录用户的 token 立即感知变更？（当前 JWT 不含权限信息，权限每次请求实时查 DB，天然即时生效） | 工程 | 否 |
| Q2 | `can_see` vs `can_list` 的边界：`can_see=false` 但 `can_list=true` 是否有意义？ | 产品 | 否 — 实现时 `can_see` 是 can_list 的前置条件 |
| Q3 | 默认权限策略：未配置权限的用户是否可以访问所有 Public 根目录？ | 产品 | 是 — 建议默认 can_see/can_read/can_list=true |

## 10. 实施计划

| 阶段 | 内容 | 预估 |
|------|------|------|
| Phase 1 | 数据库迁移 + 权限检查引擎 | 核心逻辑 |
| Phase 2 | 权限 API + 中间件注入 | 后端完成 |
| Phase 3 | 管理后台权限页面 | 前端 |
| Phase 4 | 集成测试 + 构建验证 | 质量保障 |
