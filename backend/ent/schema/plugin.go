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
