package helper

import "github.com/shopspring/decimal"

type Side int

const (
	LONG  Side = 1
	SHORT Side = -1
)

type CancelOrderParams struct {
	Symbol  string
	OrderID int64
}

type CreateLimitOrderParams struct {
	Symbol     string
	IsAsk      bool
	ReduceOnly bool
	Price      decimal.Decimal
	Size       decimal.Decimal
}

type CreateMarketOrderParams struct {
	Symbol                   string
	IsAsk                    bool
	ReduceOnly               bool
	AcceptableExecutionPrice decimal.Decimal
	Size                     decimal.Decimal
}
