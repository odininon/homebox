package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

// EntityPriceHistory holds the schema definition for the EntityPriceHistory entity.
type EntityPriceHistory struct {
	ent.Schema
}

func (EntityPriceHistory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.BaseMixin{},
	}
}

func (EntityPriceHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("recorded_at"),
		index.Fields("source"),
		index.Fields("entity_id", "recorded_at"),
	}
}

// Fields of the EntityPriceHistory.
func (EntityPriceHistory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("entity_id", uuid.UUID{}),
		field.Float("price").
			Default(0.0),
		field.Float("market_low").
			Optional().
			Default(0.0),
		field.Float("market_mid").
			Optional().
			Default(0.0),
		field.Float("market_high").
			Optional().
			Default(0.0),
		field.Float("direct_low").
			Optional().
			Default(0.0),
		field.String("source").
			MaxLen(100).
			Default("tcgplayer"),
		field.String("source_id").
			MaxLen(100).
			Optional(),
		field.Time("recorded_at"),
		field.String("notes").
			MaxLen(500).
			Optional(),
	}
}

// Edges of the EntityPriceHistory.
func (EntityPriceHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entity", Entity.Type).
			Field("entity_id").
			Ref("price_history").
			Required().
			Unique(),
	}
}
