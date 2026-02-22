package helper

import (
	"context"
	"errors"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/shopspring/decimal"
)

// ExchangeAdapter 交易所适配器
// 封装对不同交易所的订单操作，提供统一的接口
type ExchangeAdapter struct {
	svcCtx  *svc.ServiceContext  // 服务上下文
	helper  OrderHelperInterface // 订单操作接口实现
	account string               // 账户标识
}

// NewExchangeAdapter 创建交易所适配器实例
// svcCtx 服务上下文，helper 订单操作接口实现，account 账户标识
func NewExchangeAdapter(svcCtx *svc.ServiceContext, helper OrderHelperInterface, account string) *ExchangeAdapter {
	return &ExchangeAdapter{svcCtx: svcCtx, helper: helper, account: account}
}

// NewExchangeAdapterFromStrategy 根据策略记录创建交易所适配器
// 根据策略中指定的交易所类型，初始化对应的订单操作客户端
// svcCtx 服务上下文，s 策略记录
// 返回值: 交易所适配器实例，错误信息
func NewExchangeAdapterFromStrategy(svcCtx *svc.ServiceContext, s *ent.Strategy) (*ExchangeAdapter, error) {
	helper, err := NewExchangeClient(svcCtx, s.Exchange, s)
	if err != nil {
		return nil, err
	}

	account := getAccountFromStrategy(s)
	exchangeProxy := NewExchangeAdapter(svcCtx, helper, account)
	return exchangeProxy, nil
}

// NewExchangeClient 创建交易所订单操作客户端(工厂函数)
// 根据交易所类型返回对应的订单操作接口实现
// svcCtx 服务上下文，exchangeName 交易所名称，record 策略记录
// 返回值: 订单操作接口实现，错误信息
func NewExchangeClient(svcCtx *svc.ServiceContext, exchangeName string, record *ent.Strategy) (OrderHelperInterface, error) {
	switch exchangeName {
	case exchange.Lighter:
		signer, err := GetLighterClient(svcCtx, record)
		if err != nil {
			return nil, err
		}
		return NewLighterOrderHelper(svcCtx, signer), nil
	case exchange.Paradex:
		userClient, err := GetParadexClient(svcCtx, record)
		if err != nil {
			return nil, err
		}
		return NewParadexOrderHelper(svcCtx, userClient), nil
	case exchange.Variational:
		userClient, err := GetVariationalClient(svcCtx, record)
		if err != nil {
			return nil, err
		}
		return NewVariationalOrderHelper(svcCtx, userClient), nil
	default:
		return nil, errors.New("exchange unsupported")
	}
}

// getAccountFromStrategy 从策略记录中获取账户标识
func getAccountFromStrategy(s *ent.Strategy) string {
	switch s.Exchange {
	case exchange.Lighter:
		return s.ExchangeApiKey
	case exchange.Paradex, exchange.Variational:
		return s.ExchangeApiKey
	default:
		return ""
	}
}

// UpdateLeverage 更新杠杆
func (adapter *ExchangeAdapter) UpdateLeverage(ctx context.Context, symbol string, leverage uint, marginMode exchange.MarginMode) error {
	return adapter.helper.UpdateLeverage(ctx, symbol, leverage, marginMode)
}

// CancalAllOrders 取消所有订单
func (adapter *ExchangeAdapter) CancalAllOrders(ctx context.Context, symbol string) error {
	return adapter.helper.CancalAllOrders(ctx, symbol)
}

// CreateOrderBatch 批量创建订单
func (adapter *ExchangeAdapter) CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]string, []string, error) {
	return adapter.helper.CreateOrderBatch(ctx, limitOrders, marketOrders)
}

// CreateLimitOrder 创建限价单
func (adapter *ExchangeAdapter) CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (string, error) {
	return adapter.helper.CreateLimitOrder(ctx, symbol, isAsk, reduceOnly, price, size)
}

// SyncUserOrders 同步用户订单
func (adapter *ExchangeAdapter) SyncUserOrders(ctx context.Context) error {
	return adapter.helper.SyncUserOrders(ctx)
}

// ClosePosition 平仓
func (adapter *ExchangeAdapter) ClosePosition(ctx context.Context, symbol string, side Side, slippageBps int) error {
	return adapter.helper.ClosePosition(ctx, symbol, side, slippageBps)
}
