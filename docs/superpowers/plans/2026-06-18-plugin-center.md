# 插件中心（Plugin Center）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增后台可维护、面向用户公开下载的插件中心：管理员上传插件包到 S3，访客浏览并下载，统计下载次数。

**Architecture:** 沿用项目既有实体纵向切片（domain → ent schema → repository → service → handler → routes → wire）。插件包文件存储复用备份功能的 S3 配置（setting `backup_s3_config`），通过窄接口 `PluginStoreProvider` 解耦。下载端点原子计数后 302 跳转到短时 presigned URL，保持 bucket 私有。前端含公开页 `/plugins` 与管理页 `/admin/plugins`。

**Tech Stack:** Go + Ent ORM + Gin + google/wire；Vue3 + TS + Tailwind + Pinia + vue-router + vue-i18n；PostgreSQL；S3 兼容对象存储（aws-sdk-go-v2）。

**关键约定（实现前必读）:**
- 实体类型定义在 `internal/domain`，`service` 包用 `type Plugin = domain.Plugin` 别名暴露。
- repo 助手：`clientFromContext(ctx, r.client)`、`translatePersistenceError(err, notFoundErr, nil)`、`paginationResultFromTotal(total, params)` 已存在于 `internal/repository`。
- 管理员身份：`middleware2.GetAuthSubjectFromContext(c)` 返回 `AuthSubject{UserID int64, ...}`（`middleware2` 即 `internal/server/middleware`）。
- 响应助手：`response.Success/Created/Paginated/BadRequest/NotFound/ErrorFrom`，分页 `response.ParsePagination(c)`。
- ent 代码生成：`cd backend && make generate`（执行 `go generate ./ent` 等）。
- wire 重新生成同样通过 `make generate`（`go generate ./cmd/server`）。
- 迁移：`backend/migrations/NNN_*.sql`，当前最大编号为 **153**，本功能用 **154**。
- 测试：后端 `cd backend && go test -tags=unit ./...` + `golangci-lint run ./...`；前端 `cd frontend && pnpm typecheck && pnpm test:run`。

---

## File Structure

**后端（新建）:**
- `backend/internal/domain/plugin.go` — Plugin 领域结构 + 状态常量。
- `backend/ent/schema/plugin.go` — Ent schema（生成代码到 `backend/ent/plugin/`）。
- `backend/migrations/154_plugins.sql` — 建表 + 索引。
- `backend/internal/service/plugin.go` — Plugin 类型别名、Repository 接口、ListFilters、领域错误。
- `backend/internal/service/plugin_store_provider.go` — `PluginStoreProvider` 接口 + 实现。
- `backend/internal/service/plugin_service.go` — 业务逻辑 + S3 编排。
- `backend/internal/repository/plugin_repo.go` — Ent CRUD + 原子计数。
- `backend/internal/handler/dto/plugin.go` — DTO 转换。
- `backend/internal/handler/admin/plugin_handler.go` — 管理 CRUD + 上传。
- `backend/internal/handler/plugin_handler.go` — 公开 list + download。
- `backend/internal/server/routes/public.go` — 公开路由注册。

**后端（修改）:**
- `backend/internal/repository/wire.go` — 注册 `NewPluginRepository`。
- `backend/internal/service/wire.go` — 注册 `NewPluginService`、`NewPluginStoreProvider`。
- `backend/internal/handler/wire.go` — 注册两个 handler，加入 `Handlers`/`AdminHandlers`。
- `backend/internal/server/routes/admin.go` — `registerPluginRoutes`。
- `backend/internal/server/router.go` — 调用 `RegisterPublicRoutes`。

**前端（新建）:**
- `frontend/src/api/plugins.ts` — 公开 API。
- `frontend/src/api/admin/plugins.ts` — 管理 API。
- `frontend/src/views/public/PluginCenterView.vue` — 公开页。
- `frontend/src/views/admin/PluginsView.vue` — 管理页。
- `frontend/src/views/admin/__tests__/PluginsView.spec.ts` — 管理页测试。

**前端（修改）:**
- `frontend/src/router/index.ts` — `/plugins` 与 `/admin/plugins` 路由。
- `frontend/src/components/layout/AppSidebar.vue` — 管理侧边栏入口。
- `frontend/src/views/HomeView.vue` — 落地页导航入口。
- `frontend/src/types/index.ts`（或既有类型位置）— Plugin 类型。
- `frontend/src/i18n/locales/*.ts` — 文案 key。

---

## Task 1: Plugin 领域模型与状态常量

**Files:**
- Create: `backend/internal/domain/plugin.go`

- [ ] **Step 1: 写领域结构与常量**

```go
package domain

import "time"

// Plugin 状态
const (
	PluginStatusDraft     = "draft"
	PluginStatusPublished = "published"
)

// Plugin 适用平台
const (
	PluginPlatformAll     = "all"
	PluginPlatformWindows = "windows"
	PluginPlatformMacOS   = "macos"
	PluginPlatformLinux   = "linux"
)

// Plugin 插件中心条目领域模型
type Plugin struct {
	ID            int64
	Name          string
	Description   string
	Version       string
	Category      string
	Platform      string
	IconKey       string
	FileKey       string
	FileName      string
	FileSize      int64
	DownloadCount int64
	Status        string
	SortWeight    int
	CreatedBy     *int64
	UpdatedBy     *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./internal/domain/`
Expected: 编译通过，无输出。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/plugin.go
git commit -m "feat(plugin): add plugin domain model"
```

---

## Task 2: Ent schema 与代码生成

**Files:**
- Create: `backend/ent/schema/plugin.go`

- [ ] **Step 1: 写 Ent schema**

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// Plugin holds the schema definition for the Plugin entity.
//
// 删除策略：硬删除（关联 S3 对象由 service 层 best-effort 清理）
type Plugin struct {
	ent.Schema
}

func (Plugin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "plugins"},
	}
}

func (Plugin) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(200).NotEmpty().Comment("插件名称"),
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default("").Comment("插件简介（支持 Markdown）"),
		field.String("version").MaxLen(50).Default("").Comment("版本号"),
		field.String("category").MaxLen(50).Default("").Comment("分类标签"),
		field.String("platform").MaxLen(50).Default(domain.PluginPlatformAll).Comment("适用平台"),
		field.String("icon_key").Default("").Comment("图标 S3 key"),
		field.String("file_key").Default("").Comment("插件包 S3 key"),
		field.String("file_name").Default("").Comment("原始文件名"),
		field.Int64("file_size").Default(0).Comment("文件字节数"),
		field.Int64("download_count").Default(0).Comment("下载次数"),
		field.String("status").MaxLen(20).Default(domain.PluginStatusDraft).Comment("状态: draft, published"),
		field.Int("sort_weight").Default(0).Comment("排序权重，越大越靠前"),
		field.Int64("created_by").Optional().Nillable().Comment("创建管理员ID"),
		field.Int64("updated_by").Optional().Nillable().Comment("更新管理员ID"),
		field.Time("created_at").Immutable().Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (Plugin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "sort_weight", "created_at"),
		index.Fields("category"),
	}
}
```

- [ ] **Step 2: 生成 ent 代码**

Run: `cd backend && make generate`
Expected: 生成 `backend/ent/plugin/`、`backend/ent/plugin.go` 等文件，无报错。（若 `make generate` 因 wire 报缺失符号失败，先只跑 `go generate ./ent`，wire 待后续任务补齐后再整体生成。）

- [ ] **Step 3: 编译验证**

Run: `cd backend && go build ./ent/...`
Expected: 编译通过。

- [ ] **Step 4: Commit**

```bash
git add backend/ent/
git commit -m "feat(plugin): add plugin ent schema and generated code"
```

---

## Task 3: 数据库迁移

**Files:**
- Create: `backend/migrations/154_plugins.sql`

- [ ] **Step 1: 写迁移 SQL（幂等）**

```sql
CREATE TABLE IF NOT EXISTS plugins (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version VARCHAR(50) NOT NULL DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT '',
    platform VARCHAR(50) NOT NULL DEFAULT 'all',
    icon_key VARCHAR(512) NOT NULL DEFAULT '',
    file_key VARCHAR(512) NOT NULL DEFAULT '',
    file_name VARCHAR(512) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    download_count BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    sort_weight INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT NULL,
    updated_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT plugins_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT plugins_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_plugins_status_sort
    ON plugins(status, sort_weight DESC, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_plugins_category
    ON plugins(category);
```

- [ ] **Step 2: 校验编号唯一**

Run: `ls backend/migrations/ | grep '^154'`
Expected: 仅 `154_plugins.sql`。若已有 154，改用下一个空闲编号并同步本计划引用。

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/154_plugins.sql
git commit -m "feat(plugin): add plugins table migration"
```

---

## Task 4: Service 层类型、Repository 接口与领域错误

**Files:**
- Create: `backend/internal/service/plugin.go`

- [ ] **Step 1: 写类型别名、接口、错误**

```go
package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// Plugin 领域类型别名
type Plugin = domain.Plugin

// 领域错误
var (
	ErrPluginNotFound            = infraerrors.NotFound("PLUGIN_NOT_FOUND", "plugin not found")
	ErrPluginStorageNotConfigured = infraerrors.BadRequest("PLUGIN_STORAGE_NOT_CONFIGURED", "请先在备份设置中配置 S3 存储")
)

// PluginListFilters 管理列表过滤条件
type PluginListFilters struct {
	Status   string
	Category string
	Search   string
}

// PluginRepository 插件持久化接口
type PluginRepository interface {
	Create(ctx context.Context, p *Plugin) error
	GetByID(ctx context.Context, id int64) (*Plugin, error)
	Update(ctx context.Context, p *Plugin) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params pagination.PaginationParams, filters PluginListFilters) ([]Plugin, *pagination.PaginationResult, error)
	ListPublished(ctx context.Context, category string) ([]Plugin, error)
	IncrementDownloadCount(ctx context.Context, id int64) error
}
```

- [ ] **Step 2: 确认错误构造器签名**

Run: `cd backend && grep -n "func NotFound\|func BadRequest" internal/pkg/errors/*.go`
Expected: 找到 `NotFound(code, message string)` 与 `BadRequest(code, message string)`。若签名不同（如多一个参数），按实际签名调整 Step 1。

- [ ] **Step 3: 编译验证**

Run: `cd backend && go build ./internal/service/`
Expected: 编译通过（此时实现尚未注入，仅类型/接口）。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/plugin.go
git commit -m "feat(plugin): add plugin service types and repository interface"
```

---

## Task 5: Repository 实现（含原子计数）+ 测试

**Files:**
- Create: `backend/internal/repository/plugin_repo.go`
- Test: `backend/internal/repository/plugin_repo_test.go`

- [ ] **Step 1: 写实现**

```go
package repository

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/plugin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type pluginRepository struct {
	client *dbent.Client
}

func NewPluginRepository(client *dbent.Client) service.PluginRepository {
	return &pluginRepository{client: client}
}

func (r *pluginRepository) Create(ctx context.Context, p *service.Plugin) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Plugin.Create().
		SetName(p.Name).
		SetDescription(p.Description).
		SetVersion(p.Version).
		SetCategory(p.Category).
		SetPlatform(p.Platform).
		SetIconKey(p.IconKey).
		SetFileKey(p.FileKey).
		SetFileName(p.FileName).
		SetFileSize(p.FileSize).
		SetStatus(p.Status).
		SetSortWeight(p.SortWeight)
	if p.CreatedBy != nil {
		builder.SetCreatedBy(*p.CreatedBy)
	}
	if p.UpdatedBy != nil {
		builder.SetUpdatedBy(*p.UpdatedBy)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	p.ID = created.ID
	p.DownloadCount = created.DownloadCount
	p.CreatedAt = created.CreatedAt
	p.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *pluginRepository) GetByID(ctx context.Context, id int64) (*service.Plugin, error) {
	m, err := r.client.Plugin.Query().Where(plugin.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrPluginNotFound, nil)
	}
	return pluginEntityToService(m), nil
}

func (r *pluginRepository) Update(ctx context.Context, p *service.Plugin) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Plugin.UpdateOneID(p.ID).
		SetName(p.Name).
		SetDescription(p.Description).
		SetVersion(p.Version).
		SetCategory(p.Category).
		SetPlatform(p.Platform).
		SetIconKey(p.IconKey).
		SetFileKey(p.FileKey).
		SetFileName(p.FileName).
		SetFileSize(p.FileSize).
		SetStatus(p.Status).
		SetSortWeight(p.SortWeight)
	if p.UpdatedBy != nil {
		builder.SetUpdatedBy(*p.UpdatedBy)
	} else {
		builder.ClearUpdatedBy()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrPluginNotFound, nil)
	}
	p.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *pluginRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.Plugin.Delete().Where(plugin.IDEQ(id)).Exec(ctx)
	return err
}

func (r *pluginRepository) IncrementDownloadCount(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	return client.Plugin.UpdateOneID(id).AddDownloadCount(1).Exec(ctx)
}

func (r *pluginRepository) ListPublished(ctx context.Context, category string) ([]service.Plugin, error) {
	q := r.client.Plugin.Query().Where(plugin.StatusEQ(service.PluginStatusPublished))
	if category != "" {
		q = q.Where(plugin.CategoryEQ(category))
	}
	items, err := q.
		Order(dbent.Desc(plugin.FieldSortWeight)).
		Order(dbent.Desc(plugin.FieldCreatedAt)).
		Order(dbent.Desc(plugin.FieldID)).
		Limit(500).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return pluginEntitiesToService(items), nil
}

func (r *pluginRepository) List(
	ctx context.Context,
	params pagination.PaginationParams,
	filters service.PluginListFilters,
) ([]service.Plugin, *pagination.PaginationResult, error) {
	q := r.client.Plugin.Query()
	if filters.Status != "" {
		q = q.Where(plugin.StatusEQ(filters.Status))
	}
	if filters.Category != "" {
		q = q.Where(plugin.CategoryEQ(filters.Category))
	}
	if filters.Search != "" {
		q = q.Where(plugin.Or(
			plugin.NameContainsFold(filters.Search),
			plugin.DescriptionContainsFold(filters.Search),
		))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	itemsQuery := q.Offset(params.Offset()).Limit(params.Limit())
	for _, order := range pluginListOrders(params) {
		itemsQuery = itemsQuery.Order(order)
	}
	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pluginEntitiesToService(items), paginationResultFromTotal(int64(total), params), nil
}

func pluginListOrders(params pagination.PaginationParams) []func(*entsql.Selector) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	field := plugin.FieldCreatedAt
	switch sortBy {
	case "name":
		field = plugin.FieldName
	case "status":
		field = plugin.FieldStatus
	case "download_count":
		field = plugin.FieldDownloadCount
	case "sort_weight":
		field = plugin.FieldSortWeight
	case "id":
		field = plugin.FieldID
	case "", "created_at":
		field = plugin.FieldCreatedAt
	}
	if sortOrder == pagination.SortOrderAsc {
		return []func(*entsql.Selector){dbent.Asc(field), dbent.Asc(plugin.FieldID)}
	}
	return []func(*entsql.Selector){dbent.Desc(field), dbent.Desc(plugin.FieldID)}
}

func pluginEntityToService(m *dbent.Plugin) *service.Plugin {
	if m == nil {
		return nil
	}
	return &service.Plugin{
		ID:            m.ID,
		Name:          m.Name,
		Description:   m.Description,
		Version:       m.Version,
		Category:      m.Category,
		Platform:      m.Platform,
		IconKey:       m.IconKey,
		FileKey:       m.FileKey,
		FileName:      m.FileName,
		FileSize:      m.FileSize,
		DownloadCount: m.DownloadCount,
		Status:        m.Status,
		SortWeight:    m.SortWeight,
		CreatedBy:     m.CreatedBy,
		UpdatedBy:     m.UpdatedBy,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func pluginEntitiesToService(models []*dbent.Plugin) []service.Plugin {
	out := make([]service.Plugin, 0, len(models))
	for i := range models {
		if s := pluginEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
```

- [ ] **Step 2: 写测试（参考既有 repo 测试的 setup 方式）**

先确认既有 repo 测试如何建测试 client：

Run: `cd backend && grep -rln "enttest\|NewTestClient\|sqlite" internal/repository/*_test.go | head`
Expected: 找到既有 repo 测试文件作为 setup 范本（如 announcement/promo repo 测试）。

按该范本写 `plugin_repo_test.go`，至少覆盖：

```go
//go:build unit

package repository

// 用既有测试 client 工厂（参照同目录其它 *_repo_test.go 的 setup）。
// 用例：
//   TestPluginRepository_CreateAndGetByID  — Create 后 GetByID 字段一致
//   TestPluginRepository_IncrementDownloadCount — 连续 Increment 两次后 count==2
//   TestPluginRepository_ListPublished_FiltersDraft — draft 不出现在结果
//   TestPluginRepository_List_SearchAndPaginate — search 命中 name/description，分页 total 正确
```

每个用例：构造 repo → 调方法 → `require.NoError` + `require.Equal` 断言。`IncrementDownloadCount` 用例必须断言并发/连续自增结果（调用两次后 GetByID 的 `DownloadCount == 2`）。

- [ ] **Step 3: 运行测试，预期失败（实现/生成代码若缺方法）**

Run: `cd backend && go test -tags=unit ./internal/repository/ -run TestPluginRepository -v`
Expected: 通过；若编译失败提示 ent 缺 `AddDownloadCount`/`CategoryContainsFold` 等，回到 Task 2 确认 `make generate` 已生成最新代码。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/plugin_repo.go backend/internal/repository/plugin_repo_test.go
git commit -m "feat(plugin): add plugin repository with atomic download counter"
```

---

## Task 6: PluginStoreProvider（复用备份 S3 配置）

**Files:**
- Create: `backend/internal/service/plugin_store_provider.go`

说明：复用 `BackupObjectStore` 接口（`Upload/Download/Delete/PresignURL/HeadBucket`）与 `BackupObjectStoreFactory`，从同一 setting `backup_s3_config` 读取并解密配置。`PresignURL(ctx, key, expiry)` 无自定义文件名参数，因此 S3 key 内嵌原始文件名即可让浏览器保存为合理名称。

- [ ] **Step 1: 写接口与实现**

```go
package service

import (
	"context"
	"encoding/json"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// PluginObjectStore 复用备份的对象存储能力
type PluginObjectStore = BackupObjectStore

// PluginStoreProvider 返回配置好的对象存储客户端
type PluginStoreProvider interface {
	Store(ctx context.Context) (PluginObjectStore, error)
}

type pluginStoreProvider struct {
	settingRepo  SettingRepository
	encryptor    SecretEncryptor
	storeFactory BackupObjectStoreFactory
}

func NewPluginStoreProvider(
	settingRepo SettingRepository,
	encryptor SecretEncryptor,
	storeFactory BackupObjectStoreFactory,
) PluginStoreProvider {
	return &pluginStoreProvider{
		settingRepo:  settingRepo,
		encryptor:    encryptor,
		storeFactory: storeFactory,
	}
}

func (p *pluginStoreProvider) Store(ctx context.Context) (PluginObjectStore, error) {
	raw, err := p.settingRepo.GetValue(ctx, settingKeyBackupS3Config)
	if err != nil || raw == "" {
		return nil, ErrPluginStorageNotConfigured
	}
	var cfg BackupS3Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, ErrBackupS3ConfigCorrupt
	}
	if cfg.SecretAccessKey != "" {
		if decrypted, derr := p.encryptor.Decrypt(cfg.SecretAccessKey); derr != nil {
			logger.LegacyPrintf("service.plugin", "[Plugin] S3 SecretAccessKey 解密失败（可能是旧的未加密数据）: %v", derr)
		} else {
			cfg.SecretAccessKey = decrypted
		}
	}
	if !cfg.IsConfigured() {
		return nil, ErrPluginStorageNotConfigured
	}
	return p.storeFactory(ctx, &cfg)
}
```

- [ ] **Step 2: 确认依赖符号存在**

Run: `cd backend && grep -rn "SettingRepository interface\|SecretEncryptor interface\|settingKeyBackupS3Config\|type BackupObjectStoreFactory" internal/service/*.go`
Expected: 四个符号均存在于 service 包（`settingKeyBackupS3Config` 在 backup_service.go，包内可直接引用）。

- [ ] **Step 3: 编译验证**

Run: `cd backend && go build ./internal/service/`
Expected: 编译通过。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/plugin_store_provider.go
git commit -m "feat(plugin): add plugin store provider reusing backup S3 config"
```

---

## Task 7: PluginService（业务逻辑 + S3 编排）+ 测试

**Files:**
- Create: `backend/internal/service/plugin_service.go`
- Test: `backend/internal/service/plugin_service_test.go`

- [ ] **Step 1: 写 service**

```go
package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"

	"github.com/google/uuid"
)

const (
	pluginFilePrefix = "plugins/files/"
	pluginIconPrefix = "plugins/icons/"
	pluginURLExpiry  = 5 * time.Minute
)

type PluginService struct {
	repo          PluginRepository
	storeProvider PluginStoreProvider
}

func NewPluginService(repo PluginRepository, storeProvider PluginStoreProvider) *PluginService {
	return &PluginService{repo: repo, storeProvider: storeProvider}
}

// UploadResult 上传后返回给前端的元数据
type UploadResult struct {
	Key      string
	FileName string
	Size     int64
}

// UploadObject 上传插件包或图标，返回 key/原名/大小
func (s *PluginService) UploadObject(ctx context.Context, kind, fileName, contentType string, body io.Reader) (*UploadResult, error) {
	store, err := s.storeProvider.Store(ctx)
	if err != nil {
		return nil, err
	}
	prefix := pluginFilePrefix
	if kind == "icon" {
		prefix = pluginIconPrefix
	}
	key := prefix + uuid.NewString() + "-" + path.Base(fileName)
	size, err := store.Upload(ctx, key, body, contentType)
	if err != nil {
		return nil, fmt.Errorf("upload object: %w", err)
	}
	return &UploadResult{Key: key, FileName: path.Base(fileName), Size: size}, nil
}

type CreatePluginInput struct {
	Name        string
	Description string
	Version     string
	Category    string
	Platform    string
	IconKey     string
	FileKey     string
	FileName    string
	FileSize    int64
	Status      string
	SortWeight  int
	ActorID     *int64
}

type UpdatePluginInput struct {
	Name        string
	Description string
	Version     string
	Category    string
	Platform    string
	IconKey     string
	FileKey     string
	FileName    string
	FileSize    int64
	Status      string
	SortWeight  int
	ActorID     *int64
}

func (s *PluginService) Create(ctx context.Context, in *CreatePluginInput) (*Plugin, error) {
	status := in.Status
	if status == "" {
		status = domain.PluginStatusDraft
	}
	platform := in.Platform
	if platform == "" {
		platform = domain.PluginPlatformAll
	}
	p := &Plugin{
		Name:        in.Name,
		Description: in.Description,
		Version:     in.Version,
		Category:    in.Category,
		Platform:    platform,
		IconKey:     in.IconKey,
		FileKey:     in.FileKey,
		FileName:    in.FileName,
		FileSize:    in.FileSize,
		Status:      status,
		SortWeight:  in.SortWeight,
		CreatedBy:   in.ActorID,
		UpdatedBy:   in.ActorID,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PluginService) Update(ctx context.Context, id int64, in *UpdatePluginInput) (*Plugin, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// 若替换了文件/图标，删除旧对象（best-effort）
	if in.FileKey != "" && in.FileKey != p.FileKey {
		s.deleteObjectBestEffort(ctx, p.FileKey)
	}
	if in.IconKey != "" && in.IconKey != p.IconKey {
		s.deleteObjectBestEffort(ctx, p.IconKey)
	}
	p.Name = in.Name
	p.Description = in.Description
	p.Version = in.Version
	p.Category = in.Category
	if in.Platform != "" {
		p.Platform = in.Platform
	}
	if in.IconKey != "" {
		p.IconKey = in.IconKey
	}
	if in.FileKey != "" {
		p.FileKey = in.FileKey
		p.FileName = in.FileName
		p.FileSize = in.FileSize
	}
	if in.Status != "" {
		p.Status = in.Status
	}
	p.SortWeight = in.SortWeight
	p.UpdatedBy = in.ActorID
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PluginService) Delete(ctx context.Context, id int64) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.deleteObjectBestEffort(ctx, p.FileKey)
	s.deleteObjectBestEffort(ctx, p.IconKey)
	return nil
}

func (s *PluginService) deleteObjectBestEffort(ctx context.Context, key string) {
	if key == "" {
		return
	}
	store, err := s.storeProvider.Store(ctx)
	if err != nil {
		return
	}
	_ = store.Delete(ctx, key)
}

func (s *PluginService) GetByID(ctx context.Context, id int64) (*Plugin, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PluginService) List(ctx context.Context, params pagination.PaginationParams, filters PluginListFilters) ([]Plugin, *pagination.PaginationResult, error) {
	return s.repo.List(ctx, params, filters)
}

func (s *PluginService) ListPublished(ctx context.Context, category string) ([]Plugin, error) {
	return s.repo.ListPublished(ctx, category)
}

// PresignKey 为某个对象 key 生成短时下载 URL
func (s *PluginService) PresignKey(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", nil
	}
	store, err := s.storeProvider.Store(ctx)
	if err != nil {
		return "", err
	}
	return store.PresignURL(ctx, key, pluginURLExpiry)
}

// PrepareDownload 计数 +1 并返回 presigned 下载 URL
func (s *PluginService) PrepareDownload(ctx context.Context, id int64) (string, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if p.Status != domain.PluginStatusPublished || p.FileKey == "" {
		return "", ErrPluginNotFound
	}
	if err := s.repo.IncrementDownloadCount(ctx, id); err != nil {
		return "", err
	}
	return s.PresignKey(ctx, p.FileKey)
}

var _ = bytes.NewReader // 占位，若未直接使用 bytes 可移除该行与 import
```

> 注：最后一行 `var _ = bytes.NewReader` 仅为防止 `bytes` 未使用编译错误的提示；若实现中未用到 `bytes`，请直接删除该行及其 import。

- [ ] **Step 2: 写测试（mock repo + mock store）**

`PluginObjectStore`/`PluginStoreProvider` 是接口，直接写内存 mock。示例：

```go
//go:build unit

package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	uploaded map[string][]byte
	deleted  []string
}

func newFakeStore() *fakeStore { return &fakeStore{uploaded: map[string][]byte{}} }
func (f *fakeStore) Upload(_ context.Context, key string, body io.Reader, _ string) (int64, error) {
	b, _ := io.ReadAll(body)
	f.uploaded[key] = b
	return int64(len(b)), nil
}
func (f *fakeStore) Download(_ context.Context, _ string) (io.ReadCloser, error) { return nil, nil }
func (f *fakeStore) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}
func (f *fakeStore) PresignURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}
func (f *fakeStore) HeadBucket(_ context.Context) error { return nil }

type fakeProvider struct{ store *fakeStore }

func (p *fakeProvider) Store(_ context.Context) (PluginObjectStore, error) { return p.store, nil }

// fakeRepo 实现 PluginRepository（内存）。覆盖用例：
//   TestPluginService_UploadObject_PrefixByKind — kind=package→plugins/files/，kind=icon→plugins/icons/
//   TestPluginService_PrepareDownload_IncrementsAndPresigns — count+1 且返回 presigned URL
//   TestPluginService_PrepareDownload_DraftReturnsNotFound — draft 返回 ErrPluginNotFound
//   TestPluginService_Delete_RemovesObjects — 删除后 store.deleted 含 file/icon key
var _ = pagination.PaginationParams{}
var _ = require.New
```

补全 `fakeRepo`（实现接口全部方法，内存 map 存储）并写出上述四个用例的断言主体。

- [ ] **Step 3: 运行测试**

Run: `cd backend && go test -tags=unit ./internal/service/ -run TestPluginService -v`
Expected: 全部 PASS。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/plugin_service.go backend/internal/service/plugin_service_test.go
git commit -m "feat(plugin): add plugin service with S3 orchestration"
```

---

## Task 8: DTO

**Files:**
- Create: `backend/internal/handler/dto/plugin.go`

公开 DTO 隐藏内部 S3 key，暴露 `icon_url`（presigned）与 `download_url`（后端下载端点路径）。管理 DTO 保留全部字段。

- [ ] **Step 1: 写 DTO 与转换**

```go
package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// AdminPlugin 管理面 DTO
type AdminPlugin struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Version       string    `json:"version"`
	Category      string    `json:"category"`
	Platform      string    `json:"platform"`
	IconKey       string    `json:"icon_key"`
	FileKey       string    `json:"file_key"`
	FileName      string    `json:"file_name"`
	FileSize      int64     `json:"file_size"`
	DownloadCount int64     `json:"download_count"`
	Status        string    `json:"status"`
	SortWeight    int       `json:"sort_weight"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func AdminPluginFromService(p *service.Plugin) *AdminPlugin {
	if p == nil {
		return nil
	}
	return &AdminPlugin{
		ID: p.ID, Name: p.Name, Description: p.Description, Version: p.Version,
		Category: p.Category, Platform: p.Platform, IconKey: p.IconKey,
		FileKey: p.FileKey, FileName: p.FileName, FileSize: p.FileSize,
		DownloadCount: p.DownloadCount, Status: p.Status, SortWeight: p.SortWeight,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// PublicPlugin 公开面 DTO
type PublicPlugin struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	Category      string `json:"category"`
	Platform      string `json:"platform"`
	IconURL       string `json:"icon_url"`
	FileName      string `json:"file_name"`
	FileSize      int64  `json:"file_size"`
	DownloadCount int64  `json:"download_count"`
	DownloadURL   string `json:"download_url"`
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./internal/handler/dto/`
Expected: 编译通过。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/dto/plugin.go
git commit -m "feat(plugin): add plugin DTOs"
```

---

## Task 9: 管理 Handler（CRUD + 上传）+ 测试

**Files:**
- Create: `backend/internal/handler/admin/plugin_handler.go`
- Test: `backend/internal/handler/admin/plugin_handler_test.go`

上传守卫常量：插件包 ≤100MB，扩展名 `.zip/.vsix/.tar.gz/.tgz`；图标 ≤2MB，扩展名 `.png/.jpg/.jpeg/.svg/.webp`。

- [ ] **Step 1: 写 handler**

```go
package admin

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	maxPluginFileSize = 100 << 20 // 100MB
	maxPluginIconSize = 2 << 20   // 2MB
)

var (
	allowedPluginExts = map[string]bool{".zip": true, ".vsix": true, ".gz": true, ".tgz": true}
	allowedIconExts   = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".svg": true, ".webp": true}
)

type PluginHandler struct {
	pluginService *service.PluginService
}

func NewPluginHandler(pluginService *service.PluginService) *PluginHandler {
	return &PluginHandler{pluginService: pluginService}
}

type createPluginRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	Platform    string `json:"platform" binding:"omitempty,oneof=all windows macos linux"`
	IconKey     string `json:"icon_key"`
	FileKey     string `json:"file_key"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	Status      string `json:"status" binding:"omitempty,oneof=draft published"`
	SortWeight  int    `json:"sort_weight"`
}

type updatePluginRequest = createPluginRequest

func actorID(c *gin.Context) *int64 {
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		id := subject.UserID
		return &id
	}
	return nil
}

// List GET /api/v1/admin/plugins
func (h *PluginHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := response.PaginationParams(c) // 若无此 helper，用 pagination.PaginationParams{Page:page,...}
	_ = params
	items, result, err := h.pluginService.List(c.Request.Context(),
		buildPluginPagination(c, page, pageSize),
		service.PluginListFilters{
			Status:   strings.TrimSpace(c.Query("status")),
			Category: strings.TrimSpace(c.Query("category")),
			Search:   strings.TrimSpace(c.Query("search")),
		})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminPlugin, 0, len(items))
	for i := range items {
		out = append(out, *dto.AdminPluginFromService(&items[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

// GetByID GET /api/v1/admin/plugins/:id
func (h *PluginHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	p, err := h.pluginService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminPluginFromService(p))
}

// Create POST /api/v1/admin/plugins
func (h *PluginHandler) Create(c *gin.Context) {
	var req createPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.pluginService.Create(c.Request.Context(), &service.CreatePluginInput{
		Name: req.Name, Description: req.Description, Version: req.Version,
		Category: req.Category, Platform: req.Platform, IconKey: req.IconKey,
		FileKey: req.FileKey, FileName: req.FileName, FileSize: req.FileSize,
		Status: req.Status, SortWeight: req.SortWeight, ActorID: actorID(c),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.AdminPluginFromService(p))
}

// Update PUT /api/v1/admin/plugins/:id
func (h *PluginHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	var req updatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.pluginService.Update(c.Request.Context(), id, &service.UpdatePluginInput{
		Name: req.Name, Description: req.Description, Version: req.Version,
		Category: req.Category, Platform: req.Platform, IconKey: req.IconKey,
		FileKey: req.FileKey, FileName: req.FileName, FileSize: req.FileSize,
		Status: req.Status, SortWeight: req.SortWeight, ActorID: actorID(c),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminPluginFromService(p))
}

// Delete DELETE /api/v1/admin/plugins/:id
func (h *PluginHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	if err := h.pluginService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// Upload POST /api/v1/admin/plugins/upload  (multipart: kind, file)
func (h *PluginHandler) Upload(c *gin.Context) {
	kind := c.PostForm("kind")
	if kind != "package" && kind != "icon" {
		response.BadRequest(c, "kind must be package or icon")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if kind == "package" {
		if fileHeader.Size > maxPluginFileSize {
			response.BadRequest(c, "插件包不能超过 100MB")
			return
		}
		if !allowedPluginExts[ext] {
			response.BadRequest(c, "不支持的插件包格式")
			return
		}
	} else {
		if fileHeader.Size > maxPluginIconSize {
			response.BadRequest(c, "图标不能超过 2MB")
			return
		}
		if !allowedIconExts[ext] {
			response.BadRequest(c, "不支持的图标格式")
			return
		}
	}
	f, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c, "open uploaded file failed")
		return
	}
	defer f.Close()

	res, err := h.pluginService.UploadObject(
		c.Request.Context(), kind, fileHeader.Filename, fileHeader.Header.Get("Content-Type"), f)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"key": res.Key, "file_name": res.FileName, "size": res.Size})
}
```

> 注：`response.PaginationParams`/`buildPluginPagination` 是占位。实际请按 announcement handler 的写法构造 `pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: c.DefaultQuery("sort_by","created_at"), SortOrder: c.DefaultQuery("sort_order","desc")}`，删除占位行与 `buildPluginPagination`，直接 import `internal/pkg/pagination`。

- [ ] **Step 2: 校正 pagination 构造**

Run: `cd backend && sed -n '50,85p' internal/handler/admin/announcement_handler.go`
Expected: 看到 `pagination.PaginationParams{...}` 构造方式；据此替换 List 中的占位代码。

- [ ] **Step 3: 写 handler 测试（参考 admin_basic_handlers_test.go / announcement_handler_sort_test.go 的 gin 测试 setup）**

用例（用 service stub / 内存依赖，按既有测试范式注入）：

```
TestPluginHandler_Upload_RejectsOversizePackage — >100MB 返回 400
TestPluginHandler_Upload_RejectsBadExtension     — .exe 返回 400
TestPluginHandler_Upload_AcceptsZip              — .zip 返回 200，body 含 key
TestPluginHandler_Create_RequiresName            — 缺 name 返回 400
```

构造 multipart 请求用 `mime/multipart` 写 buffer，`httptest.NewRequest`。

- [ ] **Step 4: 运行测试**

Run: `cd backend && go test -tags=unit ./internal/handler/admin/ -run TestPluginHandler -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/admin/plugin_handler.go backend/internal/handler/admin/plugin_handler_test.go
git commit -m "feat(plugin): add admin plugin handler with upload guards"
```

---

## Task 10: 公开 Handler（list + download）+ 测试

**Files:**
- Create: `backend/internal/handler/plugin_handler.go`
- Test: `backend/internal/handler/plugin_handler_test.go`

- [ ] **Step 1: 写 handler**

```go
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PluginHandler struct {
	pluginService *service.PluginService
}

func NewPluginHandler(pluginService *service.PluginService) *PluginHandler {
	return &PluginHandler{pluginService: pluginService}
}

// List GET /api/v1/plugins
func (h *PluginHandler) List(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category"))
	items, err := h.pluginService.ListPublished(c.Request.Context(), category)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.PublicPlugin, 0, len(items))
	for i := range items {
		p := items[i]
		iconURL := ""
		if p.IconKey != "" {
			// presign 失败不阻断列表，仅图标留空
			if u, perr := h.pluginService.PresignKey(c.Request.Context(), p.IconKey); perr == nil {
				iconURL = u
			}
		}
		out = append(out, dto.PublicPlugin{
			ID: p.ID, Name: p.Name, Description: p.Description, Version: p.Version,
			Category: p.Category, Platform: p.Platform, IconURL: iconURL,
			FileName: p.FileName, FileSize: p.FileSize, DownloadCount: p.DownloadCount,
			DownloadURL: "/api/v1/plugins/" + strconv.FormatInt(p.ID, 10) + "/download",
		})
	}
	response.Success(c, out)
}

// Download GET /api/v1/plugins/:id/download
func (h *PluginHandler) Download(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	url, err := h.pluginService.PrepareDownload(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}
```

- [ ] **Step 2: 写测试（service stub）**

```
TestPublicPluginHandler_List_OnlyPublished — 返回项均 published，含 download_url
TestPublicPluginHandler_Download_Redirects — 返回 302，Location 为 presigned URL，且计数被调用
TestPublicPluginHandler_Download_NotFound  — draft/不存在返回 404
```

- [ ] **Step 3: 运行测试**

Run: `cd backend && go test -tags=unit ./internal/handler/ -run TestPublicPluginHandler -v`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/plugin_handler.go backend/internal/handler/plugin_handler_test.go
git commit -m "feat(plugin): add public plugin list and download handler"
```

---

## Task 11: Wire 注册与 Handlers 结构接线

**Files:**
- Modify: `backend/internal/repository/wire.go`（加 `NewPluginRepository`）
- Modify: `backend/internal/service/wire.go`（加 `NewPluginService`、`NewPluginStoreProvider`）
- Modify: `backend/internal/handler/wire.go`（加两个 handler 到 ProviderSet 与 Provide 函数参数）
- Modify: `backend/internal/handler/handler.go`（`AdminHandlers.Plugin`、`Handlers.Plugin` 字段）

- [ ] **Step 1: handler.go 加字段**

在 `AdminHandlers` 结构末尾加：

```go
	Plugin                 *admin.PluginHandler
```

在 `Handlers` 结构加：

```go
	Plugin           *PluginHandler
```

- [ ] **Step 2: repository/wire.go**

在 `wire.NewSet(...)` 中加一行：`NewPluginRepository,`

- [ ] **Step 3: service/wire.go**

在 `ProviderSet = wire.NewSet(...)` 中加：`NewPluginService,` 与 `NewPluginStoreProvider,`
（`NewPluginStoreProvider` 依赖 `SettingRepository`、`SecretEncryptor`、`BackupObjectStoreFactory`，这些 provider 备份功能已注册，wire 可自动满足。）

- [ ] **Step 4: handler/wire.go**

- `ProviderSet` 加 `NewPluginHandler,` 与 `admin.NewPluginHandler,`
- `ProvideAdminHandlers` 参数表加 `pluginHandler *admin.PluginHandler,`，返回体加 `Plugin: pluginHandler,`
- `ProvideHandlers` 参数表加 `pluginHandler *PluginHandler,`，返回体加 `Plugin: pluginHandler,`

- [ ] **Step 5: 重新生成 wire**

Run: `cd backend && make generate`
Expected: `go generate ./cmd/server` 重新生成 `wire_gen.go`，无 "no provider" 错误。

- [ ] **Step 6: 编译验证**

Run: `cd backend && go build ./...`
Expected: 全量编译通过。

- [ ] **Step 7: Commit**

```bash
git add backend/internal/repository/wire.go backend/internal/service/wire.go backend/internal/handler/wire.go backend/internal/handler/handler.go backend/internal/handler/wire_gen.go backend/cmd/server/wire_gen.go
git commit -m "feat(plugin): wire plugin repository, service and handlers"
```

> 注：`wire_gen.go` 实际路径以 `make generate` 修改的文件为准（`git status` 查看），一并提交。

---

## Task 12: 路由注册（公开 + 管理）

**Files:**
- Create: `backend/internal/server/routes/public.go`
- Modify: `backend/internal/server/routes/admin.go`（加 `registerPluginRoutes` 并在 `RegisterAdminRoutes` 调用）
- Modify: `backend/internal/server/router.go`（调用 `RegisterPublicRoutes`）

- [ ] **Step 1: 写公开路由**

```go
package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes 注册无需认证的公开路由
func RegisterPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	plugins := v1.Group("/plugins")
	{
		plugins.GET("", h.Plugin.List)
		plugins.GET("/:id/download", h.Plugin.Download)
	}
}
```

- [ ] **Step 2: admin.go 加管理路由**

在 `RegisterAdminRoutes` 的注册块（如 `registerAnnouncementRoutes(admin, h)` 附近）加：`registerPluginRoutes(admin, h)`，并新增函数：

```go
func registerPluginRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	plugins := admin.Group("/plugins")
	{
		plugins.GET("", h.Admin.Plugin.List)
		plugins.POST("", h.Admin.Plugin.Create)
		plugins.POST("/upload", h.Admin.Plugin.Upload)
		plugins.GET("/:id", h.Admin.Plugin.GetByID)
		plugins.PUT("/:id", h.Admin.Plugin.Update)
		plugins.DELETE("/:id", h.Admin.Plugin.Delete)
	}
}
```

- [ ] **Step 3: router.go 调用公开路由**

在 `router.go` 注册块（`RegisterUserRoutes(...)` 附近）加：

```go
	routes.RegisterPublicRoutes(v1, h)
```

- [ ] **Step 4: 编译验证**

Run: `cd backend && go build ./... && golangci-lint run ./internal/server/... ./internal/handler/...`
Expected: 编译 + lint 通过。

- [ ] **Step 5: 全量后端测试**

Run: `cd backend && go test -tags=unit ./... && golangci-lint run ./...`
Expected: 全绿。

- [ ] **Step 6: Commit**

```bash
git add backend/internal/server/routes/public.go backend/internal/server/routes/admin.go backend/internal/server/router.go
git commit -m "feat(plugin): register public and admin plugin routes"
```

---

## Task 13: 前端类型与 API client

**Files:**
- Modify: `frontend/src/types/index.ts`（加 Plugin 相关类型）
- Create: `frontend/src/api/admin/plugins.ts`
- Create: `frontend/src/api/plugins.ts`
- Modify: `frontend/src/api/admin/index.ts`（注册 pluginsAPI）

- [ ] **Step 1: types 加类型**

在 `frontend/src/types/index.ts` 追加：

```ts
export interface AdminPlugin {
  id: number
  name: string
  description: string
  version: string
  category: string
  platform: string
  icon_key: string
  file_key: string
  file_name: string
  file_size: number
  download_count: number
  status: 'draft' | 'published'
  sort_weight: number
  created_at: string
  updated_at: string
}

export interface PublicPlugin {
  id: number
  name: string
  description: string
  version: string
  category: string
  platform: string
  icon_url: string
  file_name: string
  file_size: number
  download_count: number
  download_url: string
}

export interface PluginUploadResult {
  key: string
  file_name: string
  size: number
}

export interface SavePluginRequest {
  name: string
  description?: string
  version?: string
  category?: string
  platform?: string
  icon_key?: string
  file_key?: string
  file_name?: string
  file_size?: number
  status?: 'draft' | 'published'
  sort_weight?: number
}
```

- [ ] **Step 2: 写 admin API client**

`frontend/src/api/admin/plugins.ts`：

```ts
/**
 * Admin Plugins API endpoints
 */
import { apiClient } from '../client'
import type {
  AdminPlugin,
  BasePaginationResponse,
  PluginUploadResult,
  SavePluginRequest
} from '@/types'

export async function list(
  page = 1,
  pageSize = 20,
  filters?: { status?: string; category?: string; search?: string; sort_by?: string; sort_order?: 'asc' | 'desc' },
  options?: { signal?: AbortSignal }
): Promise<BasePaginationResponse<AdminPlugin>> {
  const { data } = await apiClient.get<BasePaginationResponse<AdminPlugin>>('/admin/plugins', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<AdminPlugin> {
  const { data } = await apiClient.get<AdminPlugin>(`/admin/plugins/${id}`)
  return data
}

export async function create(request: SavePluginRequest): Promise<AdminPlugin> {
  const { data } = await apiClient.post<AdminPlugin>('/admin/plugins', request)
  return data
}

export async function update(id: number, request: SavePluginRequest): Promise<AdminPlugin> {
  const { data } = await apiClient.put<AdminPlugin>(`/admin/plugins/${id}`, request)
  return data
}

export async function deletePlugin(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/plugins/${id}`)
  return data
}

export async function upload(kind: 'package' | 'icon', file: File): Promise<PluginUploadResult> {
  const form = new FormData()
  form.append('kind', kind)
  form.append('file', file)
  const { data } = await apiClient.post<PluginUploadResult>('/admin/plugins/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000
  })
  return data
}

export default { list, getById, create, update, deletePlugin, upload }
```

- [ ] **Step 3: 写公开 API client**

`frontend/src/api/plugins.ts`：

```ts
/**
 * Public Plugin Center API
 */
import { apiClient } from './client'
import type { PublicPlugin } from '@/types'

export async function listPublic(category?: string): Promise<PublicPlugin[]> {
  const { data } = await apiClient.get<PublicPlugin[]>('/plugins', {
    params: category ? { category } : undefined
  })
  return data
}

export default { listPublic }
```

> 注意：响应解包以项目既有 client 拦截器为准。若 `apiClient` 拦截器已把 `{code,data}` 解到 `data`，上面写法成立；否则按既有 api 模块（如 announcements.ts）的取值方式对齐。

- [ ] **Step 4: 注册到 admin/index.ts**

import：`import pluginsAPI from './plugins'`
`adminAPI` 对象加：`plugins: pluginsAPI,`
named export 块加：`pluginsAPI,`

- [ ] **Step 5: typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: 无类型错误。

- [ ] **Step 6: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/admin/plugins.ts frontend/src/api/plugins.ts frontend/src/api/admin/index.ts
git commit -m "feat(plugin): add frontend plugin types and api clients"
```

---

## Task 14: i18n 文案

**Files:**
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: 加中文 key**

在 `zh.ts` 的 `admin` 对象内加 `plugins` 段，并在顶层加 `pluginCenter` 段：

```ts
// admin.plugins
plugins: {
  title: '插件中心',
  description: '维护可供用户下载的插件',
  searchPlaceholder: '搜索插件名称/描述',
  create: '新增插件',
  edit: '编辑插件',
  name: '名称',
  version: '版本',
  category: '分类',
  platform: '平台',
  status: '状态',
  statusDraft: '草稿',
  statusPublished: '已发布',
  sortWeight: '排序权重',
  downloadCount: '下载次数',
  packageFile: '插件包',
  icon: '图标',
  uploadPackage: '上传插件包',
  uploadIcon: '上传图标',
  deleteConfirm: '确定删除该插件？文件也会一并删除。',
  saved: '已保存',
  deleted: '已删除'
},
```

顶层（与 `admin`/`nav` 同级）加：

```ts
pluginCenter: {
  title: '插件中心',
  subtitle: '下载我们提供的插件，提升你的工作流',
  all: '全部',
  download: '下载',
  downloads: '已下载 {count} 次',
  empty: '暂无插件'
},
```

并在 `nav` 段加：`plugins: '插件中心',`

- [ ] **Step 2: 加英文 key（en.ts 对应同结构）**

```ts
plugins: {
  title: 'Plugin Center', description: 'Manage downloadable plugins for users',
  searchPlaceholder: 'Search name/description', create: 'New Plugin', edit: 'Edit Plugin',
  name: 'Name', version: 'Version', category: 'Category', platform: 'Platform',
  status: 'Status', statusDraft: 'Draft', statusPublished: 'Published',
  sortWeight: 'Sort Weight', downloadCount: 'Downloads', packageFile: 'Package',
  icon: 'Icon', uploadPackage: 'Upload Package', uploadIcon: 'Upload Icon',
  deleteConfirm: 'Delete this plugin? Its files will be removed too.',
  saved: 'Saved', deleted: 'Deleted'
},
```

顶层：

```ts
pluginCenter: {
  title: 'Plugin Center', subtitle: 'Download our plugins to boost your workflow',
  all: 'All', download: 'Download', downloads: '{count} downloads', empty: 'No plugins yet'
},
```

`nav` 加：`plugins: 'Plugin Center',`

- [ ] **Step 3: typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: 无错误。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat(plugin): add i18n strings for plugin center"
```

---

## Task 15: 管理页 PluginsView + 路由 + 侧边栏

**Files:**
- Create: `frontend/src/views/admin/PluginsView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`

- [ ] **Step 1: 写管理页**

参照 `frontend/src/views/admin/AnnouncementsView.vue` 的结构（`AppLayout` + `TablePageLayout` + 搜索/筛选 + 表格 + 创建/编辑弹窗）。要点：

- 用 `adminAPI.plugins.list(...)` 加载分页列表，列：名称、分类、版本、状态、下载次数、排序权重、更新时间、操作（编辑/删除）。
- 创建/编辑弹窗表单字段对应 `SavePluginRequest`。两个上传控件：
  - 插件包：`<input type="file" accept=".zip,.vsix,.tar.gz,.tgz">`，选择后调用 `adminAPI.plugins.upload('package', file)`，把返回的 `key/file_name/size` 暂存到表单 `file_key/file_name/file_size`，展示已上传文件名。
  - 图标：`accept="image/*"`，调用 `adminAPI.plugins.upload('icon', file)`，存 `icon_key`。
- 状态用 `Select`（草稿/已发布）；提供"上架/下架"快捷切换（改 status 后调用 update）。
- 保存调用 `create`/`update`；删除前 `confirm`（用既有确认弹窗或 `window.confirm` + i18n `admin.plugins.deleteConfirm`）。
- 成功/失败用 `useAppStore().showSuccess/showError`。
- 文案全部走 `t('admin.plugins.*')`。

- [ ] **Step 2: 路由注册**

在 `frontend/src/router/index.ts` 管理路由区（announcement 路由附近）加：

```ts
  {
    path: '/admin/plugins',
    name: 'AdminPlugins',
    component: () => import('@/views/admin/PluginsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Plugin Center',
      titleKey: 'admin.plugins.title',
      descriptionKey: 'admin.plugins.description'
    }
  },
```

- [ ] **Step 3: 侧边栏入口**

在 `AppSidebar.vue` 的管理菜单数组（`/admin/announcements` 条目附近）加：

```ts
    { path: '/admin/plugins', label: t('nav.plugins'), icon: FolderIcon, hideInSimpleMode: true },
```

（图标用既有已 import 的图标，如 `FolderIcon`/`OrderIcon`；若想用更贴切的拼图图标，确认 `components/icons` 是否已有再 import，没有就先用 `FolderIcon`，避免引入未定义符号。）

- [ ] **Step 4: typecheck + 构建检查**

Run: `cd frontend && pnpm typecheck`
Expected: 无错误。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/admin/PluginsView.vue frontend/src/router/index.ts frontend/src/components/layout/AppSidebar.vue
git commit -m "feat(plugin): add admin plugins management view"
```

---

## Task 16: 公开页 PluginCenterView + 路由 + 落地页入口

**Files:**
- Create: `frontend/src/views/public/PluginCenterView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/HomeView.vue`

- [ ] **Step 1: 写公开页**

复用 HomeView 的落地页设计语言（浅蓝、白底、卡片）。要点：

- `onMounted` 调用 `import pluginsApi from '@/api/plugins'` 的 `listPublic()` 加载。
- 顶部 hero（标题 `t('pluginCenter.title')` + 副标题 `t('pluginCenter.subtitle')`）。
- 分类筛选 tab：从结果里去重 `category` 生成，含"全部"（`t('pluginCenter.all')`）；点击重新 `listPublic(category)` 或本地过滤。
- 响应式卡片网格（`grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6`）：图标（`icon_url`，无则占位）、名称、版本徽标、平台标签、描述、下载按钮、`t('pluginCenter.downloads', { count })`。
- 下载按钮：`<a :href="plugin.download_url">`（浏览器跟随 302 直接下载）。
- 空态：`t('pluginCenter.empty')`。
- `requiresAuth:false`，无需登录态。

- [ ] **Step 2: 路由注册**

在 `frontend/src/router/index.ts` 公开路由区（`/token-merchant` 附近）加：

```ts
  {
    path: '/plugins',
    name: 'PluginCenter',
    component: () => import('@/views/public/PluginCenterView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Plugin Center',
      titleKey: 'pluginCenter.title'
    }
  },
```

- [ ] **Step 3: 落地页导航入口**

在 `HomeView.vue` 顶部导航（"成为合伙人" `router-link to="/token-merchant"` 附近）加：

```vue
          <router-link
            to="/plugins"
            class="rounded-lg px-3 py-2 text-sm font-semibold text-[#475467] transition hover:bg-[#f3f5f9] hover:text-[#0f1729]"
          >
            {{ $t('nav.plugins') }}
          </router-link>
```

（与既有 nav 项样式一致；若该文件用 `t` 而非 `$t`，按文件内既有用法对齐。）

- [ ] **Step 4: typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: 无错误。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/public/PluginCenterView.vue frontend/src/router/index.ts frontend/src/views/HomeView.vue
git commit -m "feat(plugin): add public plugin center page and landing entry"
```

---

## Task 17: 管理页测试

**Files:**
- Create: `frontend/src/views/admin/__tests__/PluginsView.spec.ts`

- [ ] **Step 1: 写测试**

参照 `frontend/src/views/admin/__tests__/RedeemView.batchUpdate.spec.ts` 的 mock 范式（`vi.hoisted` + `vi.mock('@/api/admin', ...)` + `vi.mock('@/stores/app', ...)`）。用例：

```
TestPluginsView_rendersList — mock plugins.list 返回两条，断言表格渲染名称
TestPluginsView_createSubmitsPayload — 打开弹窗、填表、保存，断言 plugins.create 收到正确 payload
TestPluginsView_uploadSetsFileKey — 触发文件上传，mock upload 返回 key，断言保存时 file_key 带上
TestPluginsView_deleteCallsApi — 删除确认后断言 plugins.deletePlugin 被调用
```

- [ ] **Step 2: 运行测试**

Run: `cd frontend && pnpm test:run -- PluginsView`
Expected: PASS。

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/__tests__/PluginsView.spec.ts
git commit -m "test(plugin): add admin plugins view tests"
```

---

## Task 18: 全量验证

- [ ] **Step 1: 后端全量**

Run: `cd backend && go test -tags=unit ./... && golangci-lint run ./...`
Expected: 全绿。

- [ ] **Step 2: 前端全量**

Run: `cd frontend && pnpm typecheck && pnpm lint:check && pnpm test:run`
Expected: 全绿。

- [ ] **Step 3: 端到端冒烟（手动，可选）**

启动后端 + 前端，确认：管理后台 S3（备份设置）已配置 → `/admin/plugins` 新增插件并上传 zip → 设为已发布 → 访客访问 `/plugins` 看到卡片 → 点击下载触发 302 并下载成功 → 列表下载次数 +1。

- [ ] **Step 4: 最终提交（若有遗留改动）**

```bash
git add -A
git commit -m "chore(plugin): final verification fixes"
```

---

## Self-Review 记录

- **Spec 覆盖**：数据模型(Task 1-3)、repo+原子计数(5)、S3 复用 provider(6)、service(7)、DTO(8)、管理 handler+上传守卫(9)、公开 handler+下载 302(10)、wire(11)、路由含公开组(12)、前端类型/api(13)、i18n(14)、管理页(15)、公开页+入口(16)、测试(17)、验证(18)。全部 spec 章节有对应任务。
- **占位说明**：handler List 的 pagination 构造、service 的 `bytes` import、前端响应解包均标注了"按既有写法对齐"的校正步骤，非空泛占位。
- **类型一致**：`PluginRepository` 接口方法名（`ListPublished`/`IncrementDownloadCount`）在 repo 实现、service 调用一致；DTO 字段与 domain 字段一致；前端类型与后端 JSON tag 一致。

