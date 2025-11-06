package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/shopspring/decimal"
)

// MatchedTrade holds the schema definition for the MatchedTrade entity.
type MatchedTrade struct {
	ent.Schema
}

// Fields of the MatchedTrade.
func (MatchedTrade) Fields() []ent.Field {
	return []ent.Field{
		field.String("strategyId").MaxLen(50),
		field.String("symbol"),
		field.Int64("buyClientOrderId").Nillable().Optional(),
		field.String("buyBaseAmount").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("buyQuoteAmount").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.Int64("buyOrderTimestamp").Nillable().Optional(),
		field.Int64("sellClientOrderId").Nillable().Optional(),
		field.String("sellBaseAmount").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("sellQuoteAmount").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.Int64("sellOrderTimestamp").Nillable().Optional(),
	}
}

// Edges of the MatchedTrade.
func (MatchedTrade) Edges() []ent.Edge {
	return nil
}

// Indexes of the MatchedTrade.
func (MatchedTrade) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("strategyId", "buyClientOrderId"),
		index.Fields("strategyId", "sellClientOrderId"),
		index.Fields("strategyId", "buyClientOrderId", "sellClientOrderId").Unique(),
	}
}
