package exchange

import (
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
	"github.com/shopspring/decimal"
)

// MarginMode 保证金模式
// 定义账户的保证金类型：全仓或逐仓
type MarginMode int

const (
	MarginModeCross    MarginMode = 0 // 全仓模式(所有仓位共享保证金)
	MarginModeIsolated MarginMode = 1 // 逐仓模式(每个仓位独立保证金)
)

// PositionSide 持仓方向
// 定义持仓是多头还是空头
type PositionSide int

const (
	PositionSideLong  PositionSide = 1  // 多头持仓(买入后持有)
	PositionSideShort PositionSide = -1 // 空头持仓(卖出后持有)
)

// Order 订单信息
// 通用订单数据结构，用于存储订单基本信息
type Order struct {
	Symbol            string          `json:"symbol"`            // 交易对符号(如 BTC-USDT)
	OrderID           string          `json:"orderId"`           // 交易所订单ID
	ClientOrderID     string          `json:"clientOrderId"`     // 客户端订单ID(用于本地追踪)
	Side              order.Side      `json:"side"`              // 订单方向(买入/卖出)
	Price             decimal.Decimal `json:"price"`             // 订单价格
	BaseAmount        decimal.Decimal `json:"baseAmount"`        // 基础资产数量(委托数量)
	FilledBaseAmount  decimal.Decimal `json:"filledBaseAmount"`  // 已成交基础资产数量
	FilledQuoteAmount decimal.Decimal `json:"filledQuoteAmount"` // 已成交计价资产数量
	Timestamp         int64           `json:"timestamp"`         // 订单创建时间戳(毫秒)
	Status            order.Status    `json:"status"`            // 订单状态
}

// UserOrders 用户订单集合
// 用于WebSocket推送或批量查询时的订单数据包装
type UserOrders struct {
	Exchange   string   `json:"exchange"`   // 交易所名称
	Account    string   `json:"account"`    // 账户标识
	Orders     []*Order `json:"orders"`     // 订单列表
	IsSnapshot bool     `json:"isSnapshot"` // 是否为快照数据(true表示全量数据)
}

// MarketStats 市场统计数据
// 包含交易对的市场价格信息
type MarketStats struct {
	Symbol    string          `json:"symbol"`    // 交易对符号
	Price     decimal.Decimal `json:"price"`     // 当前市场价格
	MarkPrice decimal.Decimal `json:"markPrice"` // 标记价格(用于计算盈亏)
}

// SubMessage 订阅消息
// WebSocket推送的统一消息格式
type SubMessage struct {
	Exchange    string       `json:"exchange"`    // 交易所名称
	UserOrders  *UserOrders  `json:"userOrders"`  // 用户订单信息
	MarketStats *MarketStats `json:"marketStats"` // 市场统计信息
}

// Position 持仓信息
// 用户在某个交易对的持仓数据
type Position struct {
	Symbol              string          `json:"symbol"`              // 交易对符号
	Side                PositionSide    `json:"side"`                // 持仓方向(1=多头,-1=空头)
	Position            decimal.Decimal `json:"position"`            // 持仓数量
	AvgEntryPrice       decimal.Decimal `json:"avgEntryPrice"`       // 平均入场价格
	UnrealizedPnl       decimal.Decimal `json:"unrealizedPnl"`       // 未实现盈亏
	RealizedPnl         decimal.Decimal `json:"realizedPnl"`         // 已实现盈亏
	LiquidationPrice    decimal.Decimal `json:"liquidationPrice"`    // 强平价格
	TotalFundingPaidOut decimal.Decimal `json:"totalFundingPaidOut"` // 已支付资金费用累计
	MarginMode          MarginMode      `json:"marginMode"`          // 保证金模式(0=全仓,1=逐仓)
}

// Account 用户账户信息
// 用户的整体账户状态
type Account struct {
	AvailableBalance decimal.Decimal `json:"availableBalance"` // 可用余额(可用于开仓)
	Positions        []*Position     `json:"positions"`        // 当前持仓列表
	TotalAssetValue  decimal.Decimal `json:"totalAssetValue"`  // 总资产价值(持仓+余额)
}
