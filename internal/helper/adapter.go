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

type AmbiguousAccount struct {
	Signer *lighter.Signer
}

type ExchangeAdapter struct {
	svcCtx  *svc.ServiceContext
	account AmbiguousAccount
}

func NewExchangeAdapter(svcCtx *svc.ServiceContext, account AmbiguousAccount) *ExchangeAdapter {
	return &ExchangeAdapter{svcCtx: svcCtx, account: account}
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
	if adapter.account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.account.Signer).UpdateLeverage(ctx, symbol, leverage, marginMode)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CancelOrderBatch(ctx context.Context, orders []CancelOrderParams) error {
	if adapter.account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.account.Signer).CancelOrderBatch(ctx, orders)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CancelOrder(ctx context.Context, symbol string, orderId int64) error {
	if adapter.account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.account.Signer).CancelOrder(ctx, symbol, orderId)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]int64, []int64, error) {
	if adapter.account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.account.Signer).CreateOrderBatch(ctx, limitOrders, marketOrders)
	}

	return nil, nil, errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (int64, error) {
	if adapter.account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.account.Signer).CreateLimitOrder(ctx, symbol, isAsk, reduceOnly, price, size)
	}

	return 0, errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateMarketOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, acceptableExecutionPrice, size decimal.Decimal) (int64, error) {
	if adapter.account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.account.Signer).CreateMarketOrder(ctx, symbol, isAsk, reduceOnly, acceptableExecutionPrice, size)
	}

	return 0, errors.New("route not found")
}

func (adapter *ExchangeAdapter) SyncInactiveOrders(ctx context.Context) error {
	if adapter.account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.account.Signer).SyncInactiveOrders(ctx)
	}

	return errors.New("route not found")
}
