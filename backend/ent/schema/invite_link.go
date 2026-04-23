package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type InviteLink struct {
	ent.Schema
}

func (InviteLink) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "invite_links"},
	}
}

func (InviteLink) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			MaxLen(128).
			Unique(),
		field.Int64("created_by_user_id"),
		field.String("creator_role").
			MaxLen(20),
		field.Int64("owner_sales_id").
			Optional().
			Nillable(),
		field.String("status").
			MaxLen(20).
			Default("active"),
		field.String("notes").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (InviteLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_by_user_id"),
		index.Fields("owner_sales_id"),
		index.Fields("status"),
	}
}
