package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/shopspring/decimal"
)

// Grid holds the schema definition for the Grid entity.
type Grid struct {
	ent.Schema
}

func (Grid) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Grid.
func (Grid) Fields() []ent.Field {
	return []ent.Field{
		field.String("strategyId").MaxLen(50),
		field.String("exchange").MaxLen(50),
		field.String("symbol").MaxLen(32),
		field.String("account"),
		field.Int("level"),
		field.String("price").GoType(decimal.Decimal{}),
		field.String("quantity").GoType(decimal.Decimal{}),
		field.String("buyClientOrderId").Nillable().Optional(),
		field.String("sellClientOrderId").Nillable().Optional(),
	}
}

// Edges of the Grid.
func (Grid) Edges() []ent.Edge {
	return nil
}

// Indexes of the Grid.
func (Grid) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("strategyId"),
		index.Fields("strategyId", "level").Unique(),
		index.Fields("exchange", "symbol", "account"),
	}
}
