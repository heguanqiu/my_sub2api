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

type InvoiceRequest struct {
	ent.Schema
}

func (InvoiceRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "invoice_requests"},
	}
}

func (InvoiceRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("order_id"),
		field.Int64("profile_id"),
		field.String("status").
			MaxLen(30).
			Default("requested"),
		field.String("provider").
			MaxLen(50).
			Default("baiwang"),
		field.String("provider_request_id").
			MaxLen(255).
			Optional().
			Nillable(),
		field.String("provider_invoice_id").
			MaxLen(255).
			Optional().
			Nillable(),
		field.String("fail_reason").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int("retry_count").
			Default(0),
		field.Time("requested_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("issued_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
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

func (InvoiceRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("order_id"),
		index.Fields("status"),
		index.Fields("provider"),
	}
}
