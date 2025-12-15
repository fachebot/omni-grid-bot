package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/shopspring/decimal"
)

// Strategy holds the schema definition for the Strategy entity.
type Strategy struct {
	ent.Schema
}

func (Strategy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Strategy.
func (Strategy) Fields() []ent.Field {
	return []ent.Field{
		field.String("guid").MaxLen(50).Unique(),
		field.Int64("owner"),
		field.String("exchange").MaxLen(50),
		field.String("symbol").MaxLen(32),
		field.String("account"),
		field.Enum("mode").Values("long", "short"),
		field.Enum("marginMode").Values("cross", "isolated"),
		field.Enum("quantityMode").Values("arithmetic", "geometric"),
		field.String("priceUpper").GoType(decimal.Decimal{}),
		field.String("priceLower").GoType(decimal.Decimal{}),
		field.Int("gridNum").Min(1).Default(10),
		field.Int("leverage").Min(1).Default(1),
		field.String("initialOrderSize").GoType(decimal.Decimal{}),
		field.Int("slippageBps").Min(0).Max(10000).Nillable().Optional(),
		field.String("entryPrice").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("triggerStopLossPrice").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("triggerTakeProfitPrice").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.Bool("enablePushNotification"),
		field.Bool("enablePushMatchedNotification").Nillable().Optional(),
		field.Time("lastLowerThresholdAlertTime").Nillable().Optional(),
		field.Time("lastUpperThresholdAlertTime").Nillable().Optional(),
		field.Enum("status").Values("active", "inactive"),
		field.String("exchangeApiKey"),
		field.String("exchangeSecretKey"),
		field.String("exchangePassphrase"),
		field.Time("startTime").Nillable().Optional(),
	}
}

// Edges of the Strategy.
func (Strategy) Edges() []ent.Edge {
	return nil
}

// Indexes of the Strategy.
func (Strategy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("owner"),
		index.Fields("exchange", "account"),
		index.Fields("exchange", "symbol", "account"),
	}
}
