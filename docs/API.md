# hfs-v2 API 接口文档

> 版本：v0.5.0 | 基础路径：`http://localhost:8080`

## 认证

除公开接口外，所有 API 请求需携带 JWT Bearer Token：

```
Authorization: Bearer <token>
```

登录获取 token，未认证返回 `401`，非管理员访问管理接口返回 `403`。

---

## 1. 认证 API

### POST /api/auth/login

登录获取 JWT Token。

**请求体：**
```json
{
  "username": "admin",
  "password": "admin"
}
```

**响应：**
```json
{
  "ok": true,
  "token": "eyJhbGciOi...",
  "user": { "username": "admin", "role": "admin" }
}
```

### POST /api/auth/logout

登出（客户端丢弃 token 即可）。

---

## 2. 公开文件 API（无需认证）

仅可访问 `public: true` 的 VFS Root。

### GET /api/public/files/roots

获取所有公开的 VFS 根目录。

**响应：**
```json
{
  "ok": true,
  "data": [{ "name": "Files" }]
}
```

### GET /api/public/files/list?path=/Files

列出公开目录内容。

**参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| path | string | VFS 路径，默认第一个公开 Root |

**响应：**
```json
{
  "ok": true,
  "data": {
    "path": "/",
    "name": "Files",
    "files": [{
      "name": "readme.txt",
      "size": 1024,
      "sizeHuman": "1.0 KB",
      "modTime": "2026-06-23T12:00:00Z",
      "isDir": false,
      "mime": "text/plain",
      "downloads": 5
    }],
    "total": 1
  }
}
```

### GET /api/public/files/download?path=/Files/readme.txt

下载公开文件。自动 +1 下载计数。

---

## 3. 文件管理 API（需认证）

### GET /api/files/?path=/Files

列出目录内容（含完整元数据）。

**响应字段（相比公开接口新增）：**
| 字段 | 类型 | 说明 |
|------|------|------|
| comment | string | 文件备注 |
| uploadedBy | string | 上传者用户名 |
| createdAt | string | 上传时间 |

### GET /api/files/download?path=/Files/readme.txt

下载文件。自动 +1 下载计数。

### GET /api/files/download-zip?paths=...&paths=...

批量下载为 ZIP。

### DELETE /api/files/?path=/Files/old.txt

删除文件或空目录。ReadOnly 的 Root 不可删除。

### PUT /api/files/rename

重命名。

**请求体：**
```json
{
  "path": "/Files/old.txt",
  "newName": "new.txt"
}
```

### PUT /api/files/comment

更新文件备注。

**请求体：**
```json
{
  "path": "/Files/readme.txt",
  "comment": "这是一份说明文档"
}
```

### POST /api/files/mkdir

新建文件夹。

**请求体：**
```json
{
  "path": "/Files",
  "dirName": "新文件夹"
}
```

### POST /api/files/upload

上传文件（multipart/form-data）。

**表单字段：**
| 字段 | 类型 | 说明 |
|------|------|------|
| path | string | 目标目录 VFS 路径 |
| files | file[] | 上传的文件 |

### POST /api/files/batch-delete

批量删除。

**请求体：**
```json
{
  "paths": ["/Files/a.txt", "/Files/b.txt"]
}
```

---

## 4. 管理 API（需 admin 角色）

### GET /api/admin/users

获取用户列表。

### POST /api/admin/users

创建用户。

**请求体：**
```json
{
  "username": "newuser",
  "password": "123456",
  "role": "user"
}
```

### DELETE /api/admin/users?id=123

删除用户（不可删除自己或最后一个管理员）。

### GET /api/admin/logs?limit=200

获取访问日志（最近 N 条，最大 500）。

### GET /api/admin/config

获取当前配置。

### PUT /api/admin/config

更新配置（部分更新）。

### GET /api/admin/download-counts

获取全局下载统计。

**响应：**
```json
{
  "ok": true,
  "data": [{
    "path": "/Files/report.pdf",
    "count": 42,
    "lastDownloadAt": "2026-06-23 12:00:00"
  }]
}
```

### GET /api/admin/connections

获取当前活跃连接。

**响应：**
```json
{
  "ok": true,
  "data": [{
    "ip": "192.168.1.100",
    "username": "admin",
    "path": "/api/files",
    "method": "GET",
    "connected": "2026-06-23T12:00:00Z"
  }]
}
```

### GET /api/admin/disk-usage

获取磁盘空间信息。

**响应：**
```json
{
  "ok": true,
  "data": [{
    "mountPoint": "D:\\",
    "total": 294972813312,
    "free": 168863481856,
    "used": 126109331456,
    "label": "Files"
  }]
}
```

---

## 错误响应格式

所有错误响应格式统一：

```json
{
  "ok": "false",
  "error": "错误描述"
}
```

| 状态码 | 说明 |
|--------|------|
| 400 | 参数错误 |
| 401 | 未认证 |
| 403 | 权限不足（非公开资源 / 非管理员） |
| 404 | 资源不存在 |
| 409 | 用户名冲突 |
| 500 | 服务器内部错误 |
