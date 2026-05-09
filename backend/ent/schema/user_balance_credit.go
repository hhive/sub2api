package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserBalanceCredit holds one positive balance grant with its remaining amount.
type UserBalanceCredit struct {
	ent.Schema
}

func (UserBalanceCredit) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_balance_credits"},
	}
}

func (UserBalanceCredit) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("source_type").
			MaxLen(32).
			NotEmpty(),
		field.String("source_id").
			MaxLen(128).
			Default(""),
		field.String("source_code").
			MaxLen(128).
			Default(""),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("remaining_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Time("settled_until_date").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "date"}),
		field.Time("expires_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("expired_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("status").
			MaxLen(20).
			Default("active"),
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

func (UserBalanceCredit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("balance_credits").
			Field("user_id").
			Required().
			Unique(),
	}
}

func (UserBalanceCredit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "status", "settled_until_date", "expires_at"),
		index.Fields("user_id", "status", "expires_at"),
		index.Fields("status", "expires_at"),
		index.Fields("source_type", "source_id"),
	}
}
