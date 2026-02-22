package helper

import (
	"context"

	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/shopspring/decimal"
)

// Side 持仓方向
// LONG 表示多头持仓，SHORT 表示空头持仓
type Side int

const (
	LONG  Side = 1  // 多头持仓
	SHORT Side = -1 // 空头持仓
)

// OrderHelperInterface 交易所订单操作统一接口
// 所有交易所的订单操作都需要实现此接口
type OrderHelperInterface interface {
	// UpdateLeverage 更新指定交易对的杠杆倍数和保证金模式
	UpdateLeverage(ctx context.Context, symbol string, leverage uint, marginMode exchange.MarginMode) error

	// CancalAllOrders 取消指定交易对的所有活跃订单
	CancalAllOrders(ctx context.Context, symbol string) error

	// CreateOrderBatch 批量创建订单
	// limitOrders 限价单列表，marketOrders 市价单列表
	// 返回值: 限价单客户端订单ID列表，市价单客户端订单ID列表，错误信息
	CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]string, []string, error)

	// CreateLimitOrder 创建单个限价单
	// symbol 交易对名称，isAsk 是否卖单，reduceOnly 是否只减仓
	// price 订单价格，size 订单数量
	// 返回值: 客户端订单ID，错误信息
	CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (string, error)

	// SyncUserOrders 同步用户的订单数据到本地数据库
	SyncUserOrders(ctx context.Context) error

	// ClosePosition 平仓
	// symbol 交易对名称，side 持仓方向，slippageBps 滑点容忍度(基点)
	ClosePosition(ctx context.Context, symbol string, side Side, slippageBps int) error
}

// CancelOrderParams 取消订单参数
type CancelOrderParams struct {
	Symbol  string // 交易对名称
	OrderID int64  // 订单ID
}

// CreateLimitOrderParams 创建限价单参数
type CreateLimitOrderParams struct {
	Symbol     string          // 交易对名称
	IsAsk      bool            // 是否卖单 (true=卖, false=买)
	ReduceOnly bool            // 是否只减仓
	Price      decimal.Decimal // 订单价格
	Size       decimal.Decimal // 订单数量
}

// CreateMarketOrderParams 创建市价单参数
type CreateMarketOrderParams struct {
	Symbol                   string          // 交易对名称
	IsAsk                    bool            // 是否卖单 (true=卖, false=买)
	ReduceOnly               bool            // 是否只减仓
	SlippageBps              int             // 滑点容忍度(基点)，例如 50 表示 0.5%
	AcceptableExecutionPrice decimal.Decimal // 可接受的最大成交价格(市价单用)
	Size                     decimal.Decimal // 订单数量
}
