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

type InvoiceProfile struct {
	ent.Schema
}

func (InvoiceProfile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "invoice_profiles"},
	}
}

func (InvoiceProfile) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("title").
			MaxLen(255),
		field.String("tax_no").
			MaxLen(100).
			Optional().
			Nillable(),
		field.String("email").
			MaxLen(255).
			Optional().
			Nillable(),
		field.String("phone").
			MaxLen(50).
			Optional().
			Nillable(),
		field.String("address").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("bank_name").
			MaxLen(255).
			Optional().
			Nillable(),
		field.String("bank_account").
			MaxLen(255).
			Optional().
			Nillable(),
		field.String("invoice_type").
			MaxLen(30),
		field.Bool("is_default").
			Default(false),
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

func (InvoiceProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "is_default"),
	}
}
