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

// InvoiceProfile stores user-managed invoice title information.
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
		field.String("title_type").
			MaxLen(20).
			Default("personal"),
		field.String("name").
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
			MaxLen(255).
			Default(""),
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
		field.Time("deleted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (InvoiceProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "is_default"),
		index.Fields("deleted_at"),
	}
}
