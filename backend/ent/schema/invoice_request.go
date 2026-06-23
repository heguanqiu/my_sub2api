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

// InvoiceRequest stores user invoice applications and SDK results.
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
		field.Int64("profile_id").
			Optional().
			Nillable(),
		field.String("status").
			MaxLen(30).
			Default("PENDING"),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}),
		field.Float("paid_total").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(0),
		field.Float("invoiced_total").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(0),
		field.Float("reserved_total").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(0),
		field.Float("available_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(0),
		field.String("currency").
			MaxLen(10).
			Default("CNY"),
		field.String("title_type").
			MaxLen(20).
			Default("personal"),
		field.String("title_name").
			MaxLen(255),
		field.String("tax_no").
			MaxLen(64).
			Default(""),
		field.String("address_phone").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("bank_account").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("email").
			MaxLen(255),
		field.String("content").
			MaxLen(255),
		field.String("remark").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Int("sdk_code").
			Optional().
			Nillable(),
		field.String("sdk_message").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.JSON("sdk_response", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("invoice_no").
			MaxLen(64).
			Default(""),
		field.String("invoice_date").
			MaxLen(32).
			Default(""),
		field.String("pdf_url").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("ofd_url").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("xml_url").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
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
		index.Fields("profile_id"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("user_id", "status"),
	}
}
