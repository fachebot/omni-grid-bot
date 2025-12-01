package helper

import (
	"context"
	"errors"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/shopspring/decimal"
)

type AmbiguousAccount struct {
	Signer     *lighter.Signer
	ParaClient *paradex.UserClient
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
	case exchange.Paradex:
		userClient, err := GetParadexClient(svcCtx, s)
		if err != nil {
			return nil, err
		}
		account.ParaClient = userClient
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

	if adapter.Account.ParaClient != nil {
		return NewParadexOrderHelper(adapter.svcCtx, adapter.Account.ParaClient).UpdateLeverage(ctx, symbol, leverage, marginMode)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CancalAllOrders(ctx context.Context, symbol string) error {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CancalAllOrders(ctx, symbol)
	}

	if adapter.Account.ParaClient != nil {
		return NewParadexOrderHelper(adapter.svcCtx, adapter.Account.ParaClient).CancalAllOrders(ctx, symbol)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateOrderBatch(ctx context.Context, limitOrders []CreateLimitOrderParams, marketOrders []CreateMarketOrderParams) ([]string, []string, error) {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CreateOrderBatch(ctx, limitOrders, marketOrders)
	}

	if adapter.Account.ParaClient != nil {
		return NewParadexOrderHelper(adapter.svcCtx, adapter.Account.ParaClient).CreateOrderBatch(ctx, limitOrders, marketOrders)
	}

	return nil, nil, errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateLimitOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, price, size decimal.Decimal) (string, error) {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CreateLimitOrder(ctx, symbol, isAsk, reduceOnly, price, size)
	}

	if adapter.Account.ParaClient != nil {
		return NewParadexOrderHelper(adapter.svcCtx, adapter.Account.ParaClient).CreateLimitOrder(ctx, symbol, isAsk, reduceOnly, price, size)
	}

	return "", errors.New("route not found")
}

func (adapter *ExchangeAdapter) CreateMarketOrder(ctx context.Context, symbol string, isAsk, reduceOnly bool, acceptableExecutionPrice, size decimal.Decimal) (string, error) {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).CreateMarketOrder(ctx, symbol, isAsk, reduceOnly, acceptableExecutionPrice, size)
	}

	if adapter.Account.ParaClient != nil {
		return NewParadexOrderHelper(adapter.svcCtx, adapter.Account.ParaClient).CreateMarketOrder(ctx, symbol, isAsk, reduceOnly, acceptableExecutionPrice, size)
	}

	return "", errors.New("route not found")
}

func (adapter *ExchangeAdapter) SyncUserOrders(ctx context.Context) error {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).SyncInactiveOrders(ctx)
	}

	if adapter.Account.ParaClient != nil {
		return NewParadexOrderHelper(adapter.svcCtx, adapter.Account.ParaClient).SyncUserOrders(ctx)
	}

	return errors.New("route not found")
}

func (adapter *ExchangeAdapter) ClosePosition(ctx context.Context, symbol string, side Side, slippageBps int) error {
	if adapter.Account.Signer != nil {
		return NewLighterOrderHelper(adapter.svcCtx, adapter.Account.Signer).ClosePosition(ctx, symbol, side, slippageBps)
	}

	if adapter.Account.ParaClient != nil {
		return NewParadexOrderHelper(adapter.svcCtx, adapter.Account.ParaClient).ClosePosition(ctx, symbol, side, slippageBps)
	}

	return errors.New("route not found")
}
