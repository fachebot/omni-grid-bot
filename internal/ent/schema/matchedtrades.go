package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/shopspring/decimal"
)

// MatchedTrades holds the schema definition for the MatchedTrades entity.
type MatchedTrades struct {
	ent.Schema
}

// Fields of the MatchedTrades.
func (MatchedTrades) Fields() []ent.Field {
	return []ent.Field{
		field.String("strategyId").MaxLen(50),
		field.String("price").GoType(decimal.Decimal{}),
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

// Edges of the MatchedTrades.
func (MatchedTrades) Edges() []ent.Edge {
	return nil
}
