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

type InvoiceDocument struct {
	ent.Schema
}

func (InvoiceDocument) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "invoice_documents"},
	}
}

func (InvoiceDocument) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("invoice_request_id"),
		field.String("invoice_no").
			MaxLen(100).
			Optional().
			Nillable(),
		field.String("invoice_code").
			MaxLen(100).
			Optional().
			Nillable(),
		field.String("file_url").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("file_type").
			MaxLen(20).
			Optional().
			Nillable(),
		field.JSON("raw_payload_summary", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (InvoiceDocument) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("invoice_request_id"),
		index.Fields("invoice_no"),
	}
}
