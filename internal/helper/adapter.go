package helper

import (
	"context"
	"errors"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/shopspring/decimal"
)

type MarketMetadata struct {
	MinBaseAmount          decimal.Decimal // 最小基础货币数量
	MinQuoteAmount         decimal.Decimal // 最小计价货币数量
	OrderQuoteLimit        string          // 订单计价限制（可选字段）
	SupportedSizeDecimals  uint8           // 支持的数量小数位数
	SupportedPriceDecimals uint8           // 支持的价格小数位数
	SupportedQuoteDecimals uint8           // 支持的计价小数位数
}

func GetMarketMetadata(ctx context.Context, svcCtx *svc.ServiceContext, exchangeType, symbol string) (MarketMetadata, error) {
	switch exchangeType {
	case exchange.Lighter:
		metadata, err := svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
		if err != nil {
			return MarketMetadata{}, err
		}

		ret := MarketMetadata{
			MinBaseAmount:          metadata.MinBaseAmount,
			MinQuoteAmount:         metadata.MinQuoteAmount,
			OrderQuoteLimit:        metadata.OrderQuoteLimit,
			SupportedSizeDecimals:  metadata.SupportedSizeDecimals,
			SupportedPriceDecimals: metadata.SupportedPriceDecimals,
			SupportedQuoteDecimals: metadata.SupportedQuoteDecimals,
		}
		return ret, nil
	default:
		return MarketMetadata{}, errors.New("exchange unsupported")
	}
}

type AmbiguousAccount struct {
	Signer *lighter.Signer
}

type ExchangeAdapter struct {
	svcCtx  *svc.ServiceContext
	Account AmbiguousAccount
}

func NewExchangeAdapter(svcCtx *svc.ServiceContext, account AmbiguousAccount) *ExchangeAdapter {
	return &ExchangeAdapter{svcCtx: svcCtx, Account: account}
}

func NewExchangeAdapterFromStrategy(svcCtx *svc.ServiceContext, s *ent.Strategy) (*ExchangeAdapter, error) {
	var account AmbiguousAccount

	switch s.Exchange {
	case exchange.Lighter:
		signer, err := GetLighterClient(svcCtx, s)
		if err != nil {
			return nil, err
		}
		account.Signer = signer
	default:
		return nil, errors.New("exchange unsupported")
	}

	exchangeProxy := NewExchangeAdapter(svcCtx, account)
	return exchangeProxy, nil
}

func (adapter *ExchangeAdapter) UpdateLeverage(ctx context.Context, symbol string, leverage uint, marginMode exchange.MarginMode) error {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).UpdateLeverage(ctx, symbol, leverage, marginMode)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CancelOrderBatch(ctx context.Context, orders []CancelOrderParams) error {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CancelOrderBatch(ctx, orders)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CancelOrder(ctx context.Context, symbol string, orderId int64) error {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CancelOrder(ctx, symbol, orderId)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]int64, []int64, error) {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CreateOrderBatch(ctx, limitOrders, marketOrders)
	}

	return nil, nil, errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (int64, error) {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CreateLimitOrder(ctx, symbol, isAsk, reduceOnly, price, size)
	}

	return 0, errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateMarketOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, acceptableExecutionPrice, size decimal.Decimal) (int64, error) {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CreateMarketOrder(ctx, symbol, isAsk, reduceOnly, acceptableExecutionPrice, size)
	}

	return 0, errors.New("route not found")
}

func (adapter *ExchangeAdapter) SyncInactiveOrders(ctx context.Context) error {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).SyncInactiveOrders(ctx)
	}

	return errors.New("route not found")
}
