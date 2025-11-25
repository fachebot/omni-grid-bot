package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/shopspring/decimal"
)

// Order holds the schema definition for the Order entity.
type Order struct {
	ent.Schema
}

func (Order) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.String("exchange"),
		field.String("account"),
		field.String("symbol"),
		field.String("orderId"),
		field.Int64("clientOrderId"),
		field.Enum("side").Values("buy", "sell"),
		field.String("price").GoType(decimal.Decimal{}),
		field.String("baseAmount").GoType(decimal.Decimal{}),
		field.String("filledBaseAmount").GoType(decimal.Decimal{}),
		field.String("filledQuoteAmount").GoType(decimal.Decimal{}),
		field.Enum("status").Values("in-progress", "pending", "open", "filled", "canceled"),
		field.Int64("timestamp"),
	}
}

// Edges of the Order.
func (Order) Edges() []ent.Edge {
	return nil
}

// Indexes of the Event.
func (Order) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("exchange", "account"),
		index.Fields("exchange", "account", "clientOrderId"),
		index.Fields("exchange", "symbol", "orderId").Unique(),
	}
}
