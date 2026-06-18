# 插件中心（Plugin Center）设计文档

- **日期**: 2026-06-18
- **状态**: 已评审，待实现
- **作者**: brainstorming session

## 1. 背景与目标

新增一个**面向用户的插件中心**页面，用于发布可下载的插件打包文件（如 IDE 插件、CLI 工具等）。

核心诉求：

- 管理员可在后台维护（增删改查）插件，并上传插件打包文件。
- 普通访客（无需登录）可浏览插件列表并下载。
- 统计每个插件的下载次数。

### 成功标准

1. 管理员能在后台新增插件、上传插件包与图标、编辑、上下架、删除。
2. 访客打开 `/plugins` 能看到已发布插件的卡片网格，可按分类筛选并下载。
3. 下载次数被准确统计（并发安全）。
4. 插件包文件存储在 S3 / 对象存储，复用现有备份功能的 S3 配置。
5. 全部通过后端单测 + lint，前端 typecheck + 单测。

### 非目标（YAGNI）

- 不做插件版本历史 / 多版本并存（只保留当前版本字段）。
- 不做付费 / 权限分级下载（页面公开）。
- 不做独立的插件专用 S3 配置（复用备份配置）。
- 不做插件评论 / 评分。

## 2. 技术栈与现有约定

- 后端：Go + Ent ORM + Gin，依赖注入用 google/wire。
- 每个实体的纵向切片：`ent/schema` → `repository` → `service` → `handler`（用户面 `handler/`，管理面 `handler/admin/`）→ `routes` → wire 注册。
- 迁移：`backend/migrations/NNN_*.sql`，编号递增、幂等（`IF NOT EXISTS`），通过 `go:embed` 嵌入。
- 前端：Vue3 + TS + Tailwind + Pinia + vue-router + vue-i18n。管理页在 `views/admin/`，公开页在 `views/public/`（`requiresAuth:false`），API client 在 `api/` 与 `api/admin/`，侧边栏在 `components/layout/AppSidebar.vue`。
- S3：现有 `repository/backup_s3_store.go` 实现了 `Upload / Download / Delete / PresignURL`；配置存于 setting 表 `backup_s3_config`，由 `BackupService.GetS3Config()` 加载并解密；`BackupS3Config` 已含 `Prefix` 字段。

参考实体（CRUD 范式）：`announcement`（schema/repo/service/handler/dto/前端 AnnouncementsView）。

## 3. 数据模型

新增 Ent 实体 `Plugin` → 表 `plugins`。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | int64 PK (BIGSERIAL) | |
| `name` | string(200), NOT NULL | 插件名称 |
| `description` | text | 简介（支持 Markdown）|
| `version` | string(50) | 版本号，如 `v1.2.0` |
| `category` | string(50) | 分类标签（如 Claude Code / Codex / Cursor）|
| `platform` | string(50), default `all` | 适用平台：`all` / `windows` / `macos` / `linux` |
| `icon_key` | string | 图标 S3 key，可空 |
| `file_key` | string | 插件包 S3 key |
| `file_name` | string | 原始文件名，下载时作为 attachment 名 |
| `file_size` | int64, default 0 | 字节 |
| `download_count` | int64, default 0 | 下载次数 |
| `status` | string(20), default `draft` | `draft` / `published` |
| `sort_weight` | int, default 0 | 排序权重，越大越靠前 |
| `created_by` | *int64 | 创建管理员 ID |
| `updated_by` | *int64 | 更新管理员 ID |
| `created_at` | timestamptz, default now | |
| `updated_at` | timestamptz, default now, update now | |

索引：`(status, sort_weight DESC, created_at DESC)`，用于公开列表查询。

删除策略：硬删除。删除插件时同步删除 S3 上的 `file_key` 与 `icon_key` 对象（best-effort，删除失败仅记日志不阻断）。

迁移文件 `migrations/154_plugins.sql`（占位编号，实现时取当前最大编号 +1），幂等建表 + 建索引 + status CHECK 约束。

## 4. 后端结构

新增文件：

- `backend/ent/schema/plugin.go` — 实体定义，随后 `go generate` 重新生成 ent。
- `backend/migrations/154_plugins.sql` — 见上。
- `backend/internal/repository/plugin_repo.go` — CRUD、`ListPublished(ctx, category)`、`List(ctx, params, filters)`（分页，管理面）、`IncrementDownloadCount(ctx, id)`（原子 `UPDATE ... SET download_count = download_count + 1`）。
- `backend/internal/service/plugin_service.go` — 业务逻辑 + S3 编排（上传/删除对象、生成 presigned URL）。
- `backend/internal/service/plugin_store_provider.go` — 见 §5。
- `backend/internal/handler/admin/plugin_handler.go` — 管理面 CRUD + 上传。
- `backend/internal/handler/plugin_handler.go` — 公开面 List + Download。
- `backend/internal/handler/dto/plugin.go` — DTO（公开 DTO 隐藏内部 S3 key，暴露 `icon_url` 与 `download_url`）。
- wire 注册：`repository/wire.go`、`service/wire.go`、`handler/wire.go`。
- 路由：新增 `routes/public.go`（见 §7），管理路由加到 `routes/admin.go`。

## 5. S3 复用 —— 共享存储 Provider

为避免 PluginService 直接耦合 BackupService，抽取一个窄接口：

```go
// service 包内
type PluginStoreProvider interface {
    // Store 返回一个配置好的对象存储客户端；S3 未配置时返回明确错误。
    Store(ctx context.Context) (BackupObjectStore, error)
}
```

实现 `pluginStoreProvider`：

- 依赖 `SettingRepository` + `SecretEncryptor` + `BackupObjectStoreFactory`（均为现有 provider）。
- 读取并解密 `backup_s3_config`（与 BackupService 同一来源/同一解密方式）。
- 调用 `BackupObjectStoreFactory` 构建 store；插件对象 key 统一加前缀 `plugins/`（文件 `plugins/files/<uuid>-<filename>`，图标 `plugins/icons/<uuid>-<filename>`）。
- 若 `!cfg.IsConfigured()`，返回领域错误 `ErrPluginStorageNotConfigured`（"请先在备份设置中配置 S3 存储"）。

`BackupObjectStore` 已提供 `Upload / Download / Delete / PresignURL`，PluginService 直接使用。这样 S3 plumbing 单一来源，插件与备份解耦。

## 6. 下载与统计流程

公开下载端点 `GET /api/v1/plugins/:id/download`：

1. 校验插件存在且 `status = published`，否则 404。
2. `IncrementDownloadCount`（原子 SQL，并发安全）。
3. 生成短时有效（如 5 分钟）的 presigned GET URL，`ResponseContentDisposition` 设为 `attachment; filename="<file_name>"`。
4. HTTP 302 重定向到该 URL。

如此 bucket 保持私有（无需 public-read ACL），计数准确，且大文件不经过后端内存。

图标展示：公开列表 DTO 的 `icon_url` 为图标的 presigned GET URL（短时有效）。为减少每次列表都批量 presign 的开销，列表查询时对每个插件生成一次 presigned URL；如后续有性能问题再加缓存（当前 YAGNI）。

## 7. API 接口

**公开（无需认证）** —— 新增 `RegisterPublicRoutes(v1, h)`，挂载在 `/api/v1` 下、不经 JWT 中间件：

- `GET  /api/v1/plugins` — 已发布插件列表，可选 `?category=`。返回元数据 + `icon_url` + `download_url`。
- `GET  /api/v1/plugins/:id/download` — 计数 +1，302 跳转到 presigned S3 URL。

**管理（`/api/v1/admin`，requiresAdmin）** —— 加到 `registerPluginRoutes(admin, h)`：

- `GET    /admin/plugins` — 分页列表（filter: status、search、category；sort）。
- `POST   /admin/plugins` — 新增。
- `GET    /admin/plugins/:id` — 详情。
- `PUT    /admin/plugins/:id` — 编辑。
- `DELETE /admin/plugins/:id` — 删除（含 S3 对象清理）。
- `POST   /admin/plugins/upload` — multipart 上传，表单字段 `kind=package|icon` + `file`；返回 `{ key, file_name, size }`，前端再在保存表单时带上 key。

上传守卫（在 handler 层校验）：

- 插件包：最大尺寸默认 100MB（常量，可后续配置化）；允许扩展名 `.zip` / `.vsix` / `.tar.gz` / `.tgz`。
- 图标：必须为图片（`.png` / `.jpg` / `.jpeg` / `.svg` / `.webp`），最大 2MB。
- 路由设置 multipart 内存上限，超大文件走临时文件。

## 8. 前端结构

**公开页** `frontend/src/views/public/PluginCenterView.vue`：

- 路由 `/plugins`，`meta.requiresAuth=false`，`title`/`titleKey`。
- 复用 HomeView 落地页设计语言（浅蓝、卡片、间距）：顶部 hero + 分类筛选 tab + 响应式卡片网格。
- 卡片：图标、名称、版本徽标、简介、平台标签、下载按钮、"已下载 N 次"。
- 下载按钮指向 `download_url`（浏览器跟随 302 直接下载）。
- 在落地页导航（`HomeView.vue` 的 nav，"成为合伙人" 旁）加入口链接。

**管理页** `frontend/src/views/admin/PluginsView.vue`：

- 路由 `/admin/plugins`（`requiresAuth:true, requiresAdmin:true`），侧边栏 `AppSidebar.vue` 新增条目（拼图/扩展图标）。
- 表格列表：名称、分类、版本、状态、下载次数、排序权重、更新时间、操作。
- 新建/编辑弹窗：表单字段对应数据模型；含两个上传控件（插件包、图标，拖拽 + 进度），调用 `/admin/plugins/upload` 拿 key 再随表单提交。
- 上下架开关、删除确认。整体镜像 `AnnouncementsView.vue` 的交互模式。

**API client**：

- `frontend/src/api/plugins.ts` — 公开（list / 下载链接）。
- `frontend/src/api/admin/plugins.ts` — 管理 CRUD + upload。

**i18n**：locale 文件新增 `admin.plugins.*`（标题、字段、操作）与 `pluginCenter.*`（公开页文案），中英（及现有其他语言）补齐。

## 9. 测试与验证

**后端**：

- repo 测试：CRUD + `IncrementDownloadCount` 原子性 + `ListPublished` 过滤排序。
- service 测试：用现有 `BackupObjectStore` 接口 mock S3（不连真实 S3），覆盖上传 key 生成、删除清理、未配置 S3 报错。
- handler 测试：上传校验（尺寸/扩展名）、下载端点的计数 + 302 行为、管理 CRUD 鉴权。
- 运行 `go test -tags=unit ./...` 与 `golangci-lint run ./...`。

**前端**：

- `PluginsView` spec 镜像现有 admin view 测试；公开页关键渲染断言。
- 运行 `pnpm typecheck` 与 `pnpm test:run`。

## 10. 实现顺序（建议）

1. ent schema + 生成 + 迁移。
2. repository（含原子计数）+ 测试。
3. PluginStoreProvider + service + 测试。
4. 管理 handler（CRUD + upload）+ 路由 + wire + 测试。
5. 公开 handler（list + download）+ public 路由 + 测试。
6. 前端 API client + 管理页 + 公开页 + 侧边栏/导航入口 + i18n。
7. 前端测试，全量 lint/typecheck/test。
