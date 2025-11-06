package exchange

import (
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/order"
	"github.com/shopspring/decimal"
)

// 保证金模式
type MarginMode int

const (
	MarginModeCross    MarginMode = 0
	MarginModeIsolated MarginMode = 1
)

type Order struct {
	Symbol            string
	OrderID           int64
	ClientOrderID     int64
	Side              string
	Price             decimal.Decimal
	BaseAmount        decimal.Decimal
	FilledBaseAmount  decimal.Decimal
	FilledQuoteAmount decimal.Decimal
	Timestamp         int64
	Status            order.Status
}

type UserOrders struct {
	Exchange   string
	Account    string
	Orders     []*Order
	IsSnapshot bool
}
