package exchange

import (
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/shopspring/decimal"
)

// 保证金模式
type MarginMode int

const (
	MarginModeCross    MarginMode = 0
	MarginModeIsolated MarginMode = 1
)

// 持仓方向
type PositionSide int

const (
	PositionSideLong  PositionSide = 1
	PositionSideShort PositionSide = 1
)

type Order struct {
	Symbol            string
	OrderID           int64
	ClientOrderID     int64
	Side              order.Side
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

// 保证金信息
type Position struct {
	Symbol                string          // 交易对符号
	InitialMarginFraction decimal.Decimal // 初始保证金比例
	Side                  PositionSide    // 持仓方向标识（1为多头，-1为空头）
	Position              decimal.Decimal // 持仓数量
	AvgEntryPrice         decimal.Decimal // 平均入场价格
	PositionValue         decimal.Decimal // 持仓价值
	UnrealizedPnl         decimal.Decimal // 未实现盈亏
	RealizedPnl           decimal.Decimal // 已实现盈亏
	LiquidationPrice      decimal.Decimal // 强平价格
	TotalFundingPaidOut   decimal.Decimal // 总资金费用支出
	MarginMode            MarginMode      // 保证金模式（0为全仓，1为逐仓）
	AllocatedMargin       decimal.Decimal // 分配的保证金
}

// 用户账户信息
type Account struct {
	AvailableBalance decimal.Decimal // 可用余额
	Collateral       decimal.Decimal // 抵押品金额
	Positions        []*Position     // 仓位列表
	TotalAssetValue  decimal.Decimal // 总资产价值
	CrossAssetValue  decimal.Decimal // 全仓资产价值
}
