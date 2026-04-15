# PicHub · 图床

自托管图床，Linear 风格设计，单二进制部署，飞牛 NAS 原生应用。

灵感来自 [EasyImages2.0](https://github.com/icret/EasyImages2.0)。

## 功能

- 🖱 拖拽 / 粘贴 / 多图批量上传
- 🔗 多格式分享链接：URL · Markdown · HTML · BBCode · 缩略图
- 🖼 图片处理：自动压缩 · 缩略图 · 水印 · WebP 转换 · EXIF 清除
- 📁 相册管理、批量移动 / 删除
- 🔑 API Token（第三方客户端如 PicGo 可直接上传）
- 👥 管理后台：用户 · 配额 · 存储 · 限流 · 站点设置
- 👻 匿名上传（可开关）
- 📦 存储后端：本地磁盘（预留 S3 / 又拍云接口）
- 🎨 Linear 深色设计，信息密度高，响应迅速

## 技术栈

- **后端**：Go（标准库 + `modernc.org/sqlite` 纯 Go 驱动），单二进制，无 CGO 依赖
- **前端**：Vue 3 + Vite + TypeScript，design tokens 基于 Linear
- **数据库**：SQLite (WAL)
- **部署**：飞牛 NAS fpk 原生应用 / 任何 Linux 主机

## 开发

### 前置依赖

- Go 1.22+
- Node.js 20+
- pnpm
- （打包 fpk）`fnpack.exe`，位于 `../fnpack.exe`

### 启动开发环境

```bash
# 并发启动前后端（后端 :7800，前端 :5173，前端代理 /api 和 /i 到后端）
bash scripts/dev.sh
```

访问 http://localhost:5173 ，使用 `config.dev.yaml` 中的管理员账号（默认 `admin / admin123`）登录。

### 构建发布版本

```bash
bash scripts/build.sh
```

产出：
- `backend/bin/pichub` — Windows 调试用二进制
- `levis.pichub/app/bin/pichub` — Linux AMD64 生产二进制
- `levis.pichub.fpk` — 飞牛安装包

### 版本递增

```bash
BUMP=1 bash scripts/build.sh
# 自动把 manifest 中 version 的 patch 号 +1（例：1.0.0 → 1.0.1）
```

## 目录结构

```
飞牛/图床/
├── README.md
├── config.dev.yaml            开发环境配置
├── backend/                   Go 后端
│   ├── cmd/pichub/main.go
│   ├── internal/
│   │   ├── config/            YAML 配置 + 环境变量覆盖
│   │   ├── db/                SQLite schema + 连接
│   │   ├── auth/              密码 / JWT / 中间件 / 引导
│   │   ├── storage/           存储抽象 + Local 实现
│   │   ├── image/             处理管线（压缩/缩略/水印/WebP）
│   │   └── server/            HTTP 路由 + 各 handler + embed 前端
│   ├── go.mod
│   └── Makefile
├── frontend/                  Vue3 前端
│   ├── src/
│   │   ├── api/               fetch 封装 + token 管理
│   │   ├── stores/            Pinia
│   │   ├── styles/            tokens / base / components
│   │   ├── components/        AppLayout
│   │   └── pages/             Upload / Gallery / Albums / Tokens / Admin / Login
│   └── package.json
├── levis.pichub/              fpk 源目录
│   ├── manifest               元数据
│   ├── ICON.PNG / ICON_256.PNG
│   ├── app/                   会被打包为 app.tgz
│   │   ├── bin/pichub         Go 二进制
│   │   ├── config.default.yaml
│   │   └── ui/                desktop 应用入口配置
│   ├── cmd/                   生命周期脚本
│   └── config/                权限 / 资源
├── design-preview/            初期 3 种风格的 HTML 原型（对比用）
└── scripts/
    ├── dev.sh                 并发启动前后端
    ├── build.sh               一键构建
    └── fpk.sh                 仅打包 fpk
```

## API 快速参考

### 鉴权

```bash
# 登录取 JWT
curl -X POST http://localhost:7800/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
```

返回：`{"token":"<jwt>", "user":{...}}`

### 上传图片

```bash
# 匿名或 Bearer JWT 或 X-Token API Token
curl -X POST http://localhost:7800/api/upload \
  -H 'X-Token: pk_xxxxxxxxxxxx' \
  -F 'file=@photo.jpg'
```

返回包含 URL / Markdown / HTML / BBCode / 缩略图链接。

### 管理图库

- `GET  /api/images?page=1&size=48&q=foo&album=3`
- `DELETE /api/images/:id`
- `POST /api/images/batch-delete` body: `{"ids":[1,2,3]}`
- `POST /api/images/move` body: `{"image_ids":[...], "album_id":0}`

### 相册

- `GET/POST /api/albums`
- `PUT/DELETE /api/albums/:id`

### API Token

- `GET/POST /api/tokens`
- `DELETE /api/tokens/:id`

### 管理员

- `GET/POST /api/admin/users`
- `PUT/DELETE /api/admin/users/:id`
- `GET/PUT /api/admin/settings`（动态写入，热生效）

## 飞牛 NAS 安装

1. 运行 `bash scripts/build.sh` 得到 `levis.pichub.fpk`。
2. 在飞牛 NAS 应用中心「本地安装」上传该 fpk。
3. 安装完成后，桌面会出现「图床 PicHub」图标，点击启动（默认 7800 端口）。
4. 首次访问会进入上传页（允许匿名）。登录使用 `config.default.yaml` 中的默认管理员账号，**首次登录后请立即修改密码**。

### 环境变量覆盖

在 fpk 的 `cmd/main` 中通过 `export` 设置，或直接编辑 `${TRIM_PKGETC}/config.yaml`：

| 变量 | 含义 |
|------|------|
| `PICHUB_ADDR` | 监听地址，默认 `0.0.0.0:7800` |
| `PICHUB_DATA_DIR` | 数据根目录（SQLite 数据库 + uploads） |
| `PICHUB_ADMIN_USER` | 初始管理员用户名 |
| `PICHUB_ADMIN_PASSWORD` | 初始管理员密码 |

## 生产清单

- [ ] 替换 `levis.pichub/ICON.PNG` 等占位图标为 PicHub 专属图标
- [ ] 首次启动后修改默认管理员密码
- [ ] 配置 `public_url` 为真实域名（使用反代时开启 `trust_proxy`）
- [ ] 生产配置关闭匿名上传（如非预期）
- [ ] 定期备份 `${TRIM_PKGVAR}/pichub.db` 和 `${TRIM_PKGVAR}/uploads/`

## 许可

MIT
