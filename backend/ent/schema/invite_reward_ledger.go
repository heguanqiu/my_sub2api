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

type InviteRewardLedger struct {
	ent.Schema
}

func (InviteRewardLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "invite_reward_ledger"},
	}
}

func (InviteRewardLedger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("inviter_user_id"),
		field.Int64("invitee_user_id"),
		field.Int64("trigger_order_id"),
		field.String("reward_type").
			MaxLen(20),
		field.Float("reward_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.String("status").
			MaxLen(30),
		field.String("reason").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("confirmed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("reversed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (InviteRewardLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("inviter_user_id"),
		index.Fields("invitee_user_id"),
		index.Fields("trigger_order_id"),
		index.Fields("status"),
	}
}
