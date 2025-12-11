package exchange

import (
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/shopspring/decimal"
)

// 保证金模式
type MarginMode int

const (
	MarginModeCross    MarginMode = 0 // 全仓模式
	MarginModeIsolated MarginMode = 1 // 逐仓模式
)

// 持仓方向
type PositionSide int

const (
	PositionSideLong  PositionSide = 1  // 多头持仓
	PositionSideShort PositionSide = -1 // 空头持仓
)

// 订单信息
type Order struct {
	Symbol            string          `json:"symbol"`            // 交易对符号
	OrderID           string          `json:"orderId"`           // 订单ID
	ClientOrderID     string          `json:"clientOrderId"`     // 客户端订单ID
	Side              order.Side      `json:"side"`              // 订单方向
	Price             decimal.Decimal `json:"price"`             // 订单价格
	BaseAmount        decimal.Decimal `json:"baseAmount"`        // 基础资产数量
	FilledBaseAmount  decimal.Decimal `json:"filledBaseAmount"`  // 已成交基础资产数量
	FilledQuoteAmount decimal.Decimal `json:"filledQuoteAmount"` // 已成交计价资产数量
	Timestamp         int64           `json:"timestamp"`         // 时间戳
	Status            order.Status    `json:"status"`            // 订单状态
}

// 用户订单集合
type UserOrders struct {
	Exchange   string   `json:"exchange"`   // 交易所名称
	Account    string   `json:"account"`    // 账户标识
	Orders     []*Order `json:"orders"`     // 订单列表
	IsSnapshot bool     `json:"isSnapshot"` // 是否为快照数据
}

// 市场统计数据
type MarketStats struct {
	Symbol    string          `json:"symbol"`    // 交易对符号
	Price     decimal.Decimal `json:"price"`     // 当前价格
	MarkPrice decimal.Decimal `json:"markPrice"` // 标记价格
}

// 订阅消息
type SubMessage struct {
	UserOrders  *UserOrders  `json:"userOrders"`  // 用户订单信息
	MarketStats *MarketStats `json:"marketStats"` // 市场统计信息
}

// 保证金信息
type Position struct {
	Symbol              string          `json:"symbol"`              // 交易对符号
	Side                PositionSide    `json:"side"`                // 持仓方向标识（1为多头，-1为空头）
	Position            decimal.Decimal `json:"position"`            // 持仓数量
	AvgEntryPrice       decimal.Decimal `json:"avgEntryPrice"`       // 平均入场价格
	UnrealizedPnl       decimal.Decimal `json:"unrealizedPnl"`       // 未实现盈亏
	RealizedPnl         decimal.Decimal `json:"realizedPnl"`         // 已实现盈亏
	LiquidationPrice    decimal.Decimal `json:"liquidationPrice"`    // 强平价格
	TotalFundingPaidOut decimal.Decimal `json:"totalFundingPaidOut"` // 总资金费用支出
	MarginMode          MarginMode      `json:"marginMode"`          // 保证金模式（0为全仓，1为逐仓）
}

// 用户账户信息
type Account struct {
	AvailableBalance decimal.Decimal `json:"availableBalance"` // 可用余额
	Positions        []*Position     `json:"positions"`        // 仓位列表
	TotalAssetValue  decimal.Decimal `json:"totalAssetValue"`  // 总资产价值
}
