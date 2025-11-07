package model

import (
	"context"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/matchedtrade"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/predicate"
	"github.com/shopspring/decimal"
)

type MatchedTradeModel struct {
	client *ent.MatchedTradeClient
}

func NewMatchedTradeModel(client *ent.MatchedTradeClient) *MatchedTradeModel {
	return &MatchedTradeModel{client: client}
}

func (m *MatchedTradeModel) Create(ctx context.Context, args ent.MatchedTrade) error {
	return m.client.Create().
		SetStrategyId(args.StrategyId).
		SetSymbol(args.Symbol).
		SetNillableBuyClientOrderId(args.BuyClientOrderId).
		SetNillableBuyBaseAmount(args.BuyBaseAmount).
		SetNillableBuyQuoteAmount(args.BuyQuoteAmount).
		SetNillableBuyOrderTimestamp(args.BuyOrderTimestamp).
		SetNillableSellClientOrderId(args.SellClientOrderId).
		SetNillableSellBaseAmount(args.SellBaseAmount).
		SetNillableSellQuoteAmount(args.SellQuoteAmount).
		SetNillableSellOrderTimestamp(args.SellOrderTimestamp).
		Exec(ctx)
}

func (m *MatchedTradeModel) EnsureBuyOrder(ctx context.Context, strategyId string, buyOrder *ent.Order) (newCreated bool, err error) {
	ps := []predicate.MatchedTrade{
		matchedtrade.StrategyIdEQ(strategyId),
		matchedtrade.BuyClientOrderIdEQ(buyOrder.ClientOrderId),
	}
	count, err := m.client.Update().
		Where(ps...).
		SetBuyBaseAmount(buyOrder.FilledBaseAmount).
		SetBuyQuoteAmount(buyOrder.FilledQuoteAmount).
		SetBuyOrderTimestamp(buyOrder.Timestamp).
		Save(ctx)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return false, nil
	}

	args := ent.MatchedTrade{
		StrategyId:        strategyId,
		Symbol:            buyOrder.Symbol,
		BuyClientOrderId:  &buyOrder.ClientOrderId,
		BuyBaseAmount:     &buyOrder.FilledBaseAmount,
		BuyQuoteAmount:    &buyOrder.FilledQuoteAmount,
		BuyOrderTimestamp: &buyOrder.Timestamp,
	}
	return true, m.Create(ctx, args)
}

func (m *MatchedTradeModel) EnsureSellOrder(ctx context.Context, strategyId string, sellOrder *ent.Order) (newCreated bool, err error) {
	ps := []predicate.MatchedTrade{
		matchedtrade.StrategyIdEQ(strategyId),
		matchedtrade.SellClientOrderIdEQ(sellOrder.ClientOrderId),
	}
	count, err := m.client.Update().
		Where(ps...).
		SetSellBaseAmount(sellOrder.FilledBaseAmount).
		SetSellQuoteAmount(sellOrder.FilledQuoteAmount).
		SetSellOrderTimestamp(sellOrder.Timestamp).
		Save(ctx)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return false, nil
	}

	args := ent.MatchedTrade{
		StrategyId:         strategyId,
		Symbol:             sellOrder.Symbol,
		SellClientOrderId:  &sellOrder.ClientOrderId,
		SellBaseAmount:     &sellOrder.FilledBaseAmount,
		SellQuoteAmount:    &sellOrder.FilledQuoteAmount,
		SellOrderTimestamp: &sellOrder.Timestamp,
	}
	return true, m.Create(ctx, args)
}

func (m *MatchedTradeModel) UpdateByBuyOrder(
	ctx context.Context,
	strategyId string,
	buyOrder *ent.Order,
	sellClientOrderId int64,
	sellBaseAmount,
	sellQuoteAmount *decimal.Decimal,
	sellOrderTimestamp *int64,
) error {
	return m.client.Update().
		Where(matchedtrade.StrategyIdEQ(strategyId), matchedtrade.BuyClientOrderIdEQ(buyOrder.ClientOrderId)).
		SetNillableBuyBaseAmount(&buyOrder.FilledBaseAmount).
		SetNillableBuyQuoteAmount(&buyOrder.FilledQuoteAmount).
		SetNillableBuyOrderTimestamp(&buyOrder.Timestamp).
		SetSellClientOrderId(sellClientOrderId).
		SetNillableSellBaseAmount(sellBaseAmount).
		SetNillableSellQuoteAmount(sellQuoteAmount).
		SetNillableSellOrderTimestamp(sellOrderTimestamp).
		Exec(ctx)
}

func (m *MatchedTradeModel) UpdateBySellOrder(
	ctx context.Context,
	strategyId string,
	sellOrder *ent.Order,
	buyClientOrderId int64,
	buyBaseAmount,
	buyQuoteAmount *decimal.Decimal,
	buyOrderTimestamp *int64,
) error {
	return m.client.Update().
		Where(matchedtrade.StrategyIdEQ(strategyId), matchedtrade.SellClientOrderIdEQ(sellOrder.ClientOrderId)).
		SetNillableSellBaseAmount(&sellOrder.FilledBaseAmount).
		SetNillableSellQuoteAmount(&sellOrder.FilledQuoteAmount).
		SetNillableSellOrderTimestamp(&sellOrder.Timestamp).
		SetBuyClientOrderId(buyClientOrderId).
		SetNillableBuyBaseAmount(buyBaseAmount).
		SetNillableBuyQuoteAmount(buyQuoteAmount).
		SetNillableBuyOrderTimestamp(buyOrderTimestamp).
		Exec(ctx)
}
